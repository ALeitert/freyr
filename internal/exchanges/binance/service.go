package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/risingwavelabs/eris"

	"freyr/internal/order"
	"freyr/internal/utils"
)

const (
	// Binance's websocket address for a combined stream of spot market data.
	// See https://developers.binance.com/docs/binance-spot-api-docs/web-socket-streams.
	spotSocketURL = "wss://stream.binance.com/stream"

	// API to receive the current order book.
	spotSnapshotURL = "https://api.binance.com/api/v3/depth?symbol=BTCUSDT&limit=5000"
)

type Binance struct {
	wsCon *websocket.Conn
	idCtr atomic.Int64
}

func (s *Binance) Name() string                 { return "Binance Websocket" }
func (s *Binance) Init(_ context.Context) error { return nil }

func (s *Binance) Run(ctx context.Context) (err error) {
	//
	// Connect.

	var httpRes *http.Response
	s.wsCon, httpRes, err = websocket.DefaultDialer.DialContext(ctx, spotSocketURL, nil)
	if err != nil {
		return eris.Wrap(err, "failed to start websocket")
	}

	fmt.Printf("Connected to Binance via %s.\n", spotSocketURL)

	//
	// Goroutine to receive messages and fetch full order book.

	wg := sync.WaitGroup{}
	var listenErr, fetchErr error
	ctx, cancel := context.WithCancel(ctx)

	signalChan := make(chan struct{}, 1)
	fullOBChan := make(chan bookSnapshot, 1)
	msgChan := make(chan []byte, 1)

	defer func() {
		cancel()
		wg.Wait()
		err = eris.Join(err, listenErr, fetchErr, httpRes.Body.Close())
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		listenErr = s.listenForMessages(ctx, msgChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		fetchErr = s.fetchFullOrderBook(ctx, signalChan, fullOBChan)
	}()

	//
	// Build order book.

	// Send subscription message.
	_, err = s.sendMessage("SUBSCRIBE", "btcusdt@depth@100ms")

	minUpdateVersion := 0
	maxUpdateVersion := 0
	msgBuffer := []*subMessage{}

	snapshotVersion := 0
	orderBook := order.NewBook(1.0)

	for obComplete := false; !obComplete; {
		select {
		case <-ctx.Done():
			return nil

		case rawMsg, more := <-msgChan:
			if !more {
				return eris.Errorf("msgChan closed")
			}

			msg := &subMessage{}
			err := json.Unmarshal(rawMsg, msg)
			if err != nil {
				return eris.Wrap(err, "failed to unmarshal message")
			}

			if msg.Data == nil {
				// Ignore for now.
				continue
			}

			if minUpdateVersion == 0 {
				// First update: Start downloading full book.
				signalChan <- struct{}{}
				minUpdateVersion = msg.Data.UF
				maxUpdateVersion = msg.Data.UF - 1 // Pretend we have the previous already.
			}

			if msg.Data.UF != maxUpdateVersion+1 {
				return eris.Errorf(
					"invalid ob update version: expected %d, actual %d",
					maxUpdateVersion+1, msg.Data.UF,
				)
			}

			maxUpdateVersion = msg.Data.UL
			msgBuffer = append(msgBuffer, msg)

		case rawOB, more := <-fullOBChan:
			if !more {
				return eris.Errorf("fullOBChan closed")
			}

			if minUpdateVersion == 0 || minUpdateVersion > rawOB.LastUpdateID+1 {
				signalChan <- struct{}{}
				continue
			}

			orderBook.Update(rawOB.Asks, rawOB.Bids, -1)

			obComplete = true
			snapshotVersion = rawOB.LastUpdateID
		}
	}

	//
	// Merge updates and snapshot.

	for _, msg := range msgBuffer {
		if msg.Data.UL <= snapshotVersion {
			continue
		}

		orderBook.Update(msg.Data.Asks, msg.Data.Bids, msg.Data.ETime)
	}
	msgBuffer = nil //nolint: ineffassign

	fmt.Println("Order book complete.")

	//
	// Continue listening to updates.

	for rawMsg := range utils.CtxChanIter(ctx, msgChan) {
		var msg subMessage
		err := json.Unmarshal(rawMsg, &msg)
		if err != nil {
			return eris.Wrap(err, "failed to unmarshal message")
		}

		if msg.Data == nil {
			// Ignore for now.
			continue
		}

		// Verify and update version.
		if msg.Data.UF != maxUpdateVersion+1 {
			return eris.Errorf(
				"invalid ob update version: expected %d, actual %d",
				maxUpdateVersion+1, msg.Data.UF,
			)
		}

		//
		// Process updates.

		orderBook.Update(msg.Data.Asks, msg.Data.Bids, msg.Data.ETime)
		maxUpdateVersion = msg.Data.UL
	}

	return nil
}

func (s *Binance) Stop() error {
	if s.wsCon == nil {
		return nil
	}

	// Clean close
	err := s.wsCon.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return eris.Wrap(err, "failed to close connection")
	}

	return nil
}
