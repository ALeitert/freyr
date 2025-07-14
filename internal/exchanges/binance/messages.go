package binance

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/risingwavelabs/eris"
)

type subMessage struct {
	Result any `json:"result,omitempty,omitzero"`
	ID     int `json:"id,omitempty,omitzero"`

	Stream string `json:"stream,omitempty,omitzero"`
	Data   *struct {
		EType string `json:"e,omitempty,omitzero"` // event type
		ETime int64  `json:"E,omitempty,omitzero"` // event time
		UF    int    `json:"U,omitempty,omitzero"` // first update ID in event
		UL    int    `json:"u,omitempty,omitzero"` // final update ID in event

		Asks [][]string `json:"a,omitempty,omitzero"`
		Bids [][]string `json:"b,omitempty,omitzero"`
	} `json:"data,omitempty,omitzero"`
}

func (s *Binance) listenForMessages(ctx context.Context, msgChan chan<- []byte) error {
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

		msgChan <- message
	}
}

func (s *Binance) sendMessage(method string, params ...string) (int, error) {
	//
	// Convert message to JSON.

	msgBuilder := strings.Builder{}
	msgBuilder.WriteString("{")

	jsonMethod, err := json.Marshal(method)
	if err != nil {
		return 0, eris.Wrapf(err, "failed to marshal method string: '%s'", method)
	}

	msgBuilder.WriteString(`"method":`)
	msgBuilder.Write(jsonMethod)

	msgID := int(s.idCtr.Add(1))
	msgBuilder.WriteString(`,"id":`)
	msgBuilder.WriteString(strconv.Itoa(msgID))

	if len(params) > 0 {
		jsonParams, err := json.Marshal(params)
		if err != nil {
			return 0, eris.Wrapf(err, "failed to marshal param strings: '%s'", params)
		}

		msgBuilder.WriteString(`,"params":`)
		msgBuilder.Write(jsonParams)
	}

	msgBuilder.WriteString("}")

	//
	// Submit message.

	err = s.wsCon.WriteMessage(websocket.TextMessage, []byte(msgBuilder.String()))
	if err != nil {
		return 0, eris.Wrap(err, "failed to submit message")
	}

	return msgID, nil
}
