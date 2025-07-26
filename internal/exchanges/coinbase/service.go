package coinbase

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/risingwavelabs/eris"

	"freyr/internal/utils"
)

// const (
// )

type Coinbase struct {
	httpClient *http.Client
}

func (Coinbase) Name() string { return "Candle Collector" }
func (Coinbase) Stop() error  { return nil }

func (svc *Coinbase) Init(_ context.Context) error {
	svc.httpClient = &http.Client{}
	return nil
}

func (svc *Coinbase) Run(ctx context.Context) error {
	// TODO: Move into config.
	pairs := []string{"btc-usd", "sol-usd", "sol-btc", "eth-usd"}

	ticker := time.NewTicker(time.Duration(Granularity) * time.Second)
	for range utils.CtxChanIter(ctx, ticker.C) {
		wg := sync.WaitGroup{}
		wg.Add(len(pairs))

		for _, pair := range pairs {
			go func(pair string) {
				defer wg.Done()

				err := svc.collectRecentCandles(ctx, pair)
				if err != nil {
					fmt.Printf("Error when collecting '%s': %s\n", pair, eris.ToString(err, true))
				}
			}(pair)
		}

		wg.Wait()
	}

	return nil
}
