package order

import (
	"strconv"
)

// TODO: Better types.
type (
	Price  float64
	Amount float64
)

type Book struct {
	// TODO: B-Tree?
	curAsks map[Price]Amount
	curBids map[Price]Amount

	// TODO: Cache
}

func NewBook() Book {
	return Book{
		curAsks: make(map[Price]Amount),
		curBids: make(map[Price]Amount),
	}
}

func (b *Book) Update(
	rawAsks, rawBids [][]string,
	_ int64, // TODO: Utilise timestamp for caching.
) {
	applyUpdates(b.curAsks, rawAsks)
	applyUpdates(b.curBids, rawBids)
}

func applyUpdates(cur map[Price]Amount, rawUpdates [][]string) {
	for _, update := range rawUpdates {
		price, _ := strconv.ParseFloat(update[0], 64)
		amount, _ := strconv.ParseFloat(update[1], 64)

		if amount == 0.0 {
			delete(cur, Price(price))
		} else {
			cur[Price(price)] = Amount(amount)
		}
	}
}
