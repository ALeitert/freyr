package binance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/risingwavelabs/eris"

	"ob-chache/internal/utils"
)

type bookSnapshot struct {
	LastUpdateID int        `json:"lastUpdateId,omitempty,omitzero"`
	Asks         [][]string `json:"asks,omitempty,omitzero"`
	Bids         [][]string `json:"bids,omitempty,omitzero"`
}

func (s *Binance) fetchFullOrderBook(
	ctx context.Context,
	signalChan <-chan struct{},
	fullOBChan chan<- bookSnapshot,
) error {
	defer close(fullOBChan)

	for range utils.CtxChanIter(ctx, signalChan) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			spotSnapshotURL,
			nil,
		)
		if err != nil {
			return eris.Wrap(err, "failed to build request")
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return eris.Wrap(err, "failed to execute request")
		} else if res.StatusCode != http.StatusOK {
			return eris.Errorf("invalid status code: %d", res.StatusCode)
		}
		defer func() { _ = res.Body.Close() }()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return eris.Wrap(err, "failed to read response")
		}

		var ob bookSnapshot
		err = json.Unmarshal(data, &ob)
		if err != nil {
			return eris.Wrap(err, "failed to unmarshal response")
		}

		fullOBChan <- ob
	}

	return nil
}
