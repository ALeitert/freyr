package exchanges

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/risingwavelabs/eris"

	"ob-chache/internal/utils"
)

const (
	// Binance's websocket address for a combined stream of spot market data.
	// See https://developers.binance.com/docs/binance-spot-api-docs/web-socket-streams.
	binanceSpotSocketURL = "wss://stream.binance.com/stream"
)

type Binance struct {
	wsCon   *websocket.Conn
	idCtr   atomic.Int64
	msgChan chan string
}

func (s *Binance) Name() string { return "Binance Websocket" }
func (s *Binance) Init(_ context.Context) error {
	s.msgChan = make(chan string, 1)
	return nil
}

func (s *Binance) Run(ctx context.Context) (err error) {
	//
	// Connect.

	s.wsCon, _, err = websocket.DefaultDialer.Dial(binanceSpotSocketURL, nil)
	if err != nil {
		return eris.Wrap(err, "failed to start websocket")
	}
	defer func() { err = eris.Join(err, s.wsCon.Close()) }()

	fmt.Printf("Connected to Binance via %s.\n", binanceSpotSocketURL)

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

	for msg := range utils.CtxChanIter(ctx, s.msgChan) {
		// TODO: Process message.
		fmt.Println(time.Now().Format(time.RFC3339), msg)
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

func (s *Binance) listenForMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_, message, err := s.wsCon.ReadMessage() // blocking
		if err != nil {
			closeErr, ok := err.(*websocket.CloseError)
			if ok && closeErr.Code == websocket.CloseNormalClosure {
				// All good.
				return nil
			}
			return eris.Wrap(err, "error reading message")
		}

		s.msgChan <- string(message)
	}
}
