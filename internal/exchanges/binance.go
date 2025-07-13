package exchanges

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/risingwavelabs/eris"
)

const (
	// Binance's websocket address for a combined stream of spot market data.
	// See https://developers.binance.com/docs/binance-spot-api-docs/web-socket-streams.
	binanceSpotSocketURL = "wss://stream.binance.com/stream"
)

type Binance struct {
	wsCon *websocket.Conn
}

func (s *Binance) Name() string                 { return "Binance Websocket" }
func (s *Binance) Init(_ context.Context) error { return nil }

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
	// Listen to messages.

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_, message, err := s.wsCon.ReadMessage()
		if err != nil {
			closeErr, ok := err.(*websocket.CloseError)
			if ok && closeErr.Code == websocket.CloseNormalClosure {
				// All good.
				break
			}
			return eris.Wrap(err, "error reading message")
		}

		// TODO: Process message.
		fmt.Println(time.Now().Format(time.RFC3339), string(message))
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
