package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	// DefaultPort is the default HTTP server port.
	DefaultPort = 9876

	// DefaultReadTimeout is the maximum duration for reading the entire request,
	// including the body. A zero or negative value means there will be no timeout.
	// This helps prevent slowloris attacks.
	DefaultReadTimeout = 10 * time.Second

	// DefaultWriteTimeout is the maximum duration before timing out writes of the response.
	// This should be set higher than ReadTimeout to account for handler execution time.
	DefaultWriteTimeout = 10 * time.Second

	// DefaultIdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alives are enabled. If IdleTimeout is zero, ReadTimeout is used.
	DefaultIdleTimeout = 60 * time.Second

	// DefaultShutdownTimeout is the maximum duration to wait for active connections
	// to gracefully close during server shutdown. Should be less than Kubernetes
	// terminationGracePeriodSeconds to allow proper pod termination.
	DefaultShutdownTimeout = 5 * time.Second

	// DefaultMaxHeaderBytes controls the maximum number of bytes the server will
	// read parsing the request header's keys and values, including the request line.
	// 1 MB is a conservative default to prevent header-based DoS attacks.
	DefaultMaxHeaderBytes = 1 << 20 // 1 MB
)

// Server defines the interface for an HTTP server that handles metrics and health checks.
// Implementations must support graceful shutdown via context cancellation.
type Server interface {
	// Serve starts the HTTP server and blocks until the context is canceled.
	// It returns an error if the server fails to start or encounters an error
	// during shutdown. Returns nil on successful graceful shutdown.
	Serve(ctx context.Context) error

	// IsRunning returns true if the server is currently accepting connections.
	// This method is thread-safe and can be called concurrently.
	// Returns true only after the socket has been successfully bound.
	IsRunning() bool
}

// HealthChecker defines the interface for components that can report their health status.
// This is typically used for Kubernetes liveness probes to determine if a pod should be restarted.
//
// Implementations should return nil if healthy, or an error describing the problem.
// The health check should be lightweight and not depend on external services.
type HealthChecker interface {
	// Healthy checks if the component is healthy and capable of functioning.
	// Returns nil if healthy, or an error describing the health issue.
	// The context can be used to implement timeouts for the health check.
	Healthy(ctx context.Context) error
}

// ReadinessChecker defines the interface for components that can report their readiness status.
// This is typically used for Kubernetes readiness probes to determine if a pod can receive traffic.
//
// Implementations should return nil if ready, or an error describing why not ready.
// Unlike health checks, readiness checks may depend on external services (databases, caches, etc.).
type ReadinessChecker interface {
	// Ready checks if the component is ready to handle requests.
	// Returns nil if ready, or an error describing why the component is not ready.
	// The context can be used to implement timeouts for the readiness check.
	Ready(ctx context.Context) error
}

// server is the internal implementation of the Server interface.
// It uses the standard library http.Server with additional lifecycle management.
type server struct {
	mux             *http.ServeMux       // HTTP request multiplexer
	port            int                  // Port to listen on
	readTimeout     time.Duration        // Maximum duration for reading requests
	writeTimeout    time.Duration        // Maximum duration for writing responses
	idleTimeout     time.Duration        // Maximum idle time for keep-alive connections
	shutdownTimeout time.Duration        // Grace period for shutdown
	maxHeaderBytes  int                  // Maximum header size in bytes
	errLog          *log.Logger          // Optional error logger
	tlsConfig       *TLSConfig           // Optional TLS configuration
	mu              sync.RWMutex         // Protects running state
	running         bool                 // Indicates if server is currently running
	registry        *prometheus.Registry // Prometheus registry for metrics
}

// TLSConfig contains the certificate and key file paths for TLS/HTTPS support.
type TLSConfig struct {
	CertFile string // Path to the TLS certificate file
	KeyFile  string // Path to the TLS private key file
}

// Option is a functional option for configuring the Server.
// This pattern allows for flexible, backward-compatible configuration.
type Option func(*server)

// WithPort sets the port number for the HTTP server.
// If not specified, DefaultPort (9876) is used.
func WithPort(port int) Option {
	return func(s *server) { s.port = port }
}

// WithReadTimeout sets the maximum duration for reading the entire request.
// This includes reading the request headers and body.
// If not specified, DefaultReadTimeout (10s) is used.
func WithReadTimeout(d time.Duration) Option {
	return func(s *server) { s.readTimeout = d }
}

// WithWriteTimeout sets the maximum duration before timing out writes of the response.
// This should be set higher than ReadTimeout to account for handler execution time.
// If not specified, DefaultWriteTimeout (10s) is used.
func WithWriteTimeout(d time.Duration) Option {
	return func(s *server) { s.writeTimeout = d }
}

// WithIdleTimeout sets the maximum time to wait for the next request when keep-alives are enabled.
// If not specified, DefaultIdleTimeout (60s) is used.
func WithIdleTimeout(d time.Duration) Option {
	return func(s *server) { s.idleTimeout = d }
}

// WithShutdownTimeout sets the maximum duration to wait for graceful shutdown.
// This should be less than Kubernetes terminationGracePeriodSeconds to ensure
// proper pod termination before SIGKILL. If not specified, DefaultShutdownTimeout (5s) is used.
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *server) { s.shutdownTimeout = d }
}

// WithMaxHeaderBytes sets the maximum number of bytes to read from request headers.
// This helps prevent header-based DoS attacks. If not specified, DefaultMaxHeaderBytes (1 MB) is used.
func WithMaxHeaderBytes(n int) Option {
	return func(s *server) { s.maxHeaderBytes = n }
}

// WithHandler registers a custom HTTP handler for the specified pattern.
// Multiple handlers can be registered by calling this option multiple times.
//
// Example:
//
//	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.Write([]byte("custom response"))
//	})
//	srv := NewServer(WithHandler("/custom", customHandler))
func WithHandler(pattern string, handler http.Handler) Option {
	return func(s *server) {
		s.mux.Handle(pattern, handler)
	}
}

// WithSimpleHealth adds a simple health check endpoint at /healthz that always returns 200 OK.
// This is suitable for stateless services or services that don't need complex health checks.
// For services that need to verify dependencies, use WithHealthCheck instead.
//
// The endpoint returns:
//   - 200 OK with body "ok"
//
// Example:
//
//	srv := NewServer(WithSimpleHealth())
func WithSimpleHealth() Option {
	return func(s *server) {
		s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
	}
}

// WithTLS configures the server to use TLS/HTTPS with the provided certificate and key files.
// The server will call ListenAndServeTLS instead of ListenAndServe.
//
// Example:
//
//	srv := NewServer(
//	    WithPort(8443),
//	    WithTLS(server.TLSConfig{
//	        CertFile: "/path/to/cert.pem",
//	        KeyFile:  "/path/to/key.pem",
//	    }),
//	)
func WithTLS(cfg TLSConfig) Option {
	return func(s *server) {
		s.tlsConfig = &cfg
	}
}

// New creates a new HTTP server with the provided options.
// If no options are provided, the server uses sensible defaults suitable for most services.
//
// Default configuration:
//   - Port: 9876
//   - ReadTimeout: 10s
//   - WriteTimeout: 10s
//   - IdleTimeout: 60s
//   - ShutdownTimeout: 5s
//   - MaxHeaderBytes: 1 MB
//
// Example:
//
//	srv := server.New(
//	    server.WithPort(9876),
//	    server.WithPrometheusMetrics(),
//	    server.WithSimpleHealth(),
//	)
func New(opts ...Option) Server {
	// Create a new registry for this server instance to avoid conflicts in tests
	reg := prometheus.NewRegistry()

	s := &server{
		port:            DefaultPort,
		readTimeout:     DefaultReadTimeout,
		writeTimeout:    DefaultWriteTimeout,
		idleTimeout:     DefaultIdleTimeout,
		shutdownTimeout: DefaultShutdownTimeout,
		maxHeaderBytes:  DefaultMaxHeaderBytes,
		mux:             http.NewServeMux(),
		registry:        reg,
		errLog:          log.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	slog.Info("server initialized",
		"port", s.port,
		"read_timeout", s.readTimeout,
		"write_timeout", s.writeTimeout)

	return s
}

// AddHandler registers an HTTP handler for the specified pattern.
// This method is thread-safe and can be called concurrently from multiple goroutines.
//
// Example:
//
//	srv := server.NewServer(server.WithPort(9876))
//	srv.AddHandler("/custom", customHandler)
func (s *server) AddHandler(pattern string, handler http.Handler) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.mux.Handle(pattern, handler)
}

// IsRunning returns true if the server is currently running and accepting connections.
// This method is thread-safe and can be called concurrently from multiple goroutines.
//
// The server is considered "running" after the socket has been successfully bound and
// the server has started accepting connections. It returns false before the socket is
// bound and after the server has stopped.
//
// Example:
//
//	srv := server.NewServer(server.WithPort(9876))
//	go srv.Serve(ctx)
//	time.Sleep(100 * time.Millisecond) // Give server time to start
//	if srv.IsRunning() {
//	    log.Println("Server is accepting connections")
//	}
func (s *server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.running
}

// Serve starts the HTTP server and blocks until the context is canceled or an error occurs.
//
// The server uses errgroup to manage two goroutines:
//  1. Server goroutine: Runs the HTTP server (ListenAndServe or ListenAndServeTLS)
//  2. Shutdown goroutine: Waits for context cancellation and initiates graceful shutdown
//
// When the context is canceled (e.g., SIGTERM), the shutdown goroutine:
//   - Calls Shutdown() with a timeout to gracefully close active connections
//   - Waits for in-flight requests to complete (up to shutdownTimeout)
//   - Logs the shutdown progress
//
// This method returns:
//   - nil on successful graceful shutdown
//   - An error if the server fails to start or shutdown encounters an error
//
// Error handling:
//   - http.ErrServerClosed is not considered an error (it's expected during shutdown)
//   - All other errors are returned to the caller
//
// Example usage with errgroup for multiple services:
//
//	g, gCtx := errgroup.WithContext(ctx)
//
//	g.Go(func() error {
//	    return srv.Serve(gCtx)
//	})
//
//	g.Go(func() error {
//	    return otherService.Run(gCtx)
//	})
//
//	if err := g.Wait(); err != nil {
//	    log.Fatal(err)
//	}
func (s *server) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", s.port),
		Handler:        s.mux,
		ReadTimeout:    s.readTimeout,
		WriteTimeout:   s.writeTimeout,
		IdleTimeout:    s.idleTimeout,
		MaxHeaderBytes: s.maxHeaderBytes,
		ErrorLog:       s.errLog,
	}

	// Create listener first so we can set running=true only after socket is bound
	var (
		listener net.Listener
		err      error
	)

	if s.tlsConfig != nil {
		// For TLS, create a regular listener and wrap it with TLS
		listener, err = net.Listen("tcp", srv.Addr)
		if err != nil {
			return fmt.Errorf("failed to create listener: %w", err)
		}

		// Load TLS certificate
		cert, certErr := tls.LoadX509KeyPair(s.tlsConfig.CertFile, s.tlsConfig.KeyFile)
		if certErr != nil {
			listener.Close()
			return fmt.Errorf("failed to load TLS certificate: %w", certErr)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		listener = tls.NewListener(listener, tlsConfig)

		slog.Info("starting TLS server", "addr", srv.Addr)
	} else {
		listener, err = net.Listen("tcp", srv.Addr)
		if err != nil {
			return fmt.Errorf("failed to create listener: %w", err)
		}

		slog.Info("starting server", "addr", srv.Addr)
	}

	g, gCtx := errgroup.WithContext(ctx)

	// Server goroutine
	g.Go(func() error {
		// Mark server as running AFTER socket is successfully bound
		s.mu.Lock()
		s.running = true
		s.mu.Unlock()

		defer func() {
			// Mark server as not running when it stops
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()

		// Serve using the pre-created listener
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	})

	// Shutdown goroutine
	g.Go(func() error {
		<-gCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		slog.Info("shutting down server", "grace_period", s.shutdownTimeout)

		shutdownStart := time.Now()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}

		slog.Info("server shutdown complete", "duration", time.Since(shutdownStart))

		return nil
	})

	return g.Wait()
}
