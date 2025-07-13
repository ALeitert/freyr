package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ob-chache/internal/exchanges"
	"ob-chache/internal/services"
)

func main() {
	fmt.Println("Demonstrator for OB-Cache")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	//
	// Run services.

	err := services.Run(ctx, []services.Service{
		&exchanges.Binance{},
	})
	if err != nil {
		fmt.Println("Error while running services:", err)
		os.Exit(1) //nolint:gocritic
	}
}
