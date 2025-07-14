package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/risingwavelabs/eris"

	"ob-chache/internal/utils"
)

const (
	// Binance's websocket address for a combined stream of spot market data.
	// See https://developers.binance.com/docs/binance-spot-api-docs/web-socket-streams.
	spotSocketURL = "wss://stream.binance.com/stream"
)

type Binance struct {
	wsCon   *websocket.Conn
	idCtr   atomic.Int64
	msgChan chan []byte
}

func (s *Binance) Name() string { return "Binance Websocket" }
func (s *Binance) Init(_ context.Context) error {
	s.msgChan = make(chan []byte, 8)
	return nil
}

func (s *Binance) Run(ctx context.Context) (err error) {
	//
	// Connect.

	s.wsCon, _, err = websocket.DefaultDialer.Dial(spotSocketURL, nil)
	if err != nil {
		return eris.Wrap(err, "failed to start websocket")
	}
	defer func() { err = eris.Join(err, s.wsCon.Close()) }()

	fmt.Printf("Connected to Binance via %s.\n", spotSocketURL)

	//
	// Goroutine to receive messages.

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := sync.WaitGroup{}
	var lErr, sErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		lErr = s.listenForMessages(ctx)
	}()

	//
	// Working loop.

	// Send subscription message.
	_, err = s.sendMessage("SUBSCRIBE", "btcusdt@depth@100ms")

	// Temporary order book until it is properly established.
	// obMinVersion := math.MaxInt
	obVersion := 0
	obChanges := map[float64]float64{}

	for rawMsg := range utils.CtxChanIter(ctx, s.msgChan) {
		var msg subMessage
		err := json.Unmarshal(rawMsg, &msg)
		if err != nil {
			fmt.Println(err)
			return eris.Wrap(err, "failed to unmarshal message")
		}

		if msg.Data == nil {
			// Ignore for now.
			continue
		}

		// Verify and update version.
		if obVersion != 0 && obVersion+1 != msg.Data.UF {
			return eris.New("skipped ob version")
		}
		obVersion = msg.Data.UL

		// Update order book.
		for _, ask := range msg.Data.Asks {
			price, _ := strconv.ParseFloat(ask[0], 64)
			amount, _ := strconv.ParseFloat(ask[1], 64)

			if amount == 0.0 {
				delete(obChanges, price)
			} else {
				obChanges[price] = amount
			}
		}
	}

	//
	// Shut down.

	wg.Wait()
	return eris.Join(lErr, sErr)
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
