package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"freyr/internal/exchanges/binance"
	"freyr/internal/services"

	"github.com/risingwavelabs/eris"
)

func main() {
	fmt.Println("Demonstrator for OB-Cache")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	//
	// Run services.

	err := services.Run(ctx, []services.Service{
		&binance.Binance{},
	})
	if err != nil {
		fmt.Println("Error while running services:", eris.ToString(err, true))
		os.Exit(1) //nolint:gocritic
	}
}
