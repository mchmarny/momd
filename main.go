package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mchmarny/macos-menu/pkg/server"
)

func main() {
	if err := server.New().Serve(context.Background()); err != nil {
		fmt.Printf("server error: %v", err)
		os.Exit(1)
	}
}
