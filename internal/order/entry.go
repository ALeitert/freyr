package order

import (
	"strconv"
)

type BookEntry struct {
	// TODO: Better data types.
	Price  float64
	Amount float64
}

func Parse(rawAsks, rawBids [][]string) (asks, bids []BookEntry) {
	asks = make([]BookEntry, len(rawAsks))
	for i, ask := range rawAsks {
		asks[i].Price, _ = strconv.ParseFloat(ask[0], 64)
		asks[i].Amount, _ = strconv.ParseFloat(ask[1], 64)
	}

	bids = make([]BookEntry, len(rawBids))
	for i, bid := range rawBids {
		bids[i].Price, _ = strconv.ParseFloat(bid[0], 64)
		bids[i].Amount, _ = strconv.ParseFloat(bid[1], 64)
	}

	return
}
