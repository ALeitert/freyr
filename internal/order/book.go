package order

import (
	"strconv"

	"github.com/google/btree"
)

// TODO: Better types.
type (
	Price  float64
	Amount float64
)

type bookEntry struct {
	price  Price
	amount Amount
}

func bookEntryLess(a, b bookEntry) bool {
	return a.price < b.price
}

type Book struct {
	// TODO: Better B-Tree?
	curAsks *btree.BTreeG[bookEntry]
	curBids *btree.BTreeG[bookEntry]

	// TODO: Cache
}

func NewBook() Book {
	return Book{
		curAsks: btree.NewG(64, bookEntryLess),
		curBids: btree.NewG(64, bookEntryLess),
	}
}

func (b *Book) Update(
	rawAsks, rawBids [][]string,
	_ int64, // TODO: Utilise timestamp for caching.
) {
	applyUpdates(b.curAsks, rawAsks)
	applyUpdates(b.curBids, rawBids)
}

func (b *Book) MinAsk() Price {
	minEntry, _ := b.curAsks.Min()
	return minEntry.price
}

func (b *Book) MaxBid() Price {
	maxEntry, _ := b.curBids.Max()
	return maxEntry.price
}

func applyUpdates(cur *btree.BTreeG[bookEntry], rawUpdates [][]string) {
	for _, update := range rawUpdates {
		price, _ := strconv.ParseFloat(update[0], 64)
		amount, _ := strconv.ParseFloat(update[1], 64)

		entry := bookEntry{Price(price), Amount(amount)}

		if amount == 0.0 {
			cur.Delete(entry)
		} else {
			cur.ReplaceOrInsert(entry)
		}
	}
}
