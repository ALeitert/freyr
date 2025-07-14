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

	"ob-chache/internal/order"
	"ob-chache/internal/utils"
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

	// TODO: Proper order book type.
	snapshotVersion := 0
	obAskSnapshot := map[float64]float64{}
	obBidSnapshot := map[float64]float64{}

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

			obComplete = true
			snapshotVersion = rawOB.LastUpdateID

			askUpdates, bidUpdate := order.Parse(rawOB.Asks, rawOB.Bids)

			for _, entry := range askUpdates {
				if entry.Amount == 0.0 {
					delete(obAskSnapshot, entry.Price)
				} else {
					obAskSnapshot[entry.Price] = entry.Amount
				}
			}
			for _, entry := range bidUpdate {
				if entry.Amount == 0.0 {
					delete(obBidSnapshot, entry.Price)
				} else {
					obBidSnapshot[entry.Price] = entry.Amount
				}
			}
		}
	}

	//
	// Merge updates and snapshot.

	for _, msg := range msgBuffer {
		if msg.Data.UL <= snapshotVersion {
			continue
		}

		askUpdates, bidUpdate := order.Parse(msg.Data.Asks, msg.Data.Bids)

		for _, entry := range askUpdates {
			if entry.Amount == 0.0 {
				delete(obAskSnapshot, entry.Price)
			} else {
				obAskSnapshot[entry.Price] = entry.Amount
			}
		}
		for _, entry := range bidUpdate {
			if entry.Amount == 0.0 {
				delete(obBidSnapshot, entry.Price)
			} else {
				obBidSnapshot[entry.Price] = entry.Amount
			}
		}
	}
	msgBuffer = nil //nolint: ineffassign

	// TODO: Send updates somewhere.
	fmt.Println("order book complete")

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

		maxUpdateVersion = msg.Data.UL
		askUpdates, bidUpdate := order.Parse(msg.Data.Asks, msg.Data.Bids)

		for _, entry := range askUpdates {
			if entry.Amount == 0.0 {
				delete(obAskSnapshot, entry.Price)
			} else {
				obAskSnapshot[entry.Price] = entry.Amount
			}
		}
		for _, entry := range bidUpdate {
			if entry.Amount == 0.0 {
				delete(obBidSnapshot, entry.Price)
			} else {
				obBidSnapshot[entry.Price] = entry.Amount
			}
		}
		// TODO: Send updates somewhere.
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
