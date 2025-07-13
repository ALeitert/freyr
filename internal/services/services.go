package services

import (
	"context"
	"sync"

	"github.com/risingwavelabs/eris"
)

type Service interface {
	Name() string
	Init(ctx context.Context) error
	Run(ctx context.Context) error
	Stop() error
}

func Run(ctx context.Context, services []Service) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//
	// Initialise services.

	for _, svc := range services {
		err := svc.Init(ctx)
		if err != nil {
			return eris.Wrapf(err, "failed to initialise %s", svc.Name())
		}
	}

	//
	// Run services.

	errors := make([]error, len(services))

	wg := sync.WaitGroup{}
	wg.Add(len(services))
	for i, svc := range services {
		go func() {
			defer wg.Done()
			defer cancel()
			errors[i] = svc.Run(ctx)
		}()
	}

	// Triggered when at least one service ends or termination signal is received.
	<-ctx.Done()

	//
	// Stop services (if `ctx` does not do so).

	for i, svc := range services {
		err := svc.Stop()
		if err != nil {
			errors[i] = eris.Wrapf(err, "failed to shut down %s", svc.Name())
		}
	}
	wg.Wait()

	return eris.Join(errors...)
}
