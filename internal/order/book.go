package order

import (
	"cmp"
	"math"
	"slices"
	"strconv"

	"github.com/google/btree"
)

// TODO: Better types.
type (
	Price  float64
	Amount float64
)

type bookSide byte

const (
	askSide = iota
	bidSide
)

type bookEntry struct {
	price  Price
	amount Amount
}

func bookEntryLess(a, b bookEntry) bool {
	return a.price < b.price
}

type Book struct {
	granularity Price

	// TODO: Better B-Tree?
	curAsks *btree.BTreeG[bookEntry]
	curBids *btree.BTreeG[bookEntry]

	// TODO: Cache
}

func NewBook(granularity Price) Book {
	return Book{
		granularity: granularity,
		curAsks:     btree.NewG(64, bookEntryLess),
		curBids:     btree.NewG(64, bookEntryLess),
	}
}

func (b *Book) MinAsk() Price {
	minEntry, _ := b.curAsks.Min()
	return minEntry.price
}

func (b *Book) MaxBid() Price {
	maxEntry, _ := b.curBids.Max()
	return maxEntry.price
}

func (b *Book) Update(
	rawAsks, rawBids [][]string,
	_ int64, // TODO: Utilise timestamp for caching.
) {
	b.applyUpdates(b.curAsks, rawAsks, askSide)
	b.applyUpdates(b.curBids, rawBids, bidSide)
}

func (b *Book) applyUpdates(cur *btree.BTreeG[bookEntry], rawUpdates [][]string, side bookSide) {
	var roundFunc func(x float64) float64
	switch side {
	case askSide:
		roundFunc = math.Ceil
	case bidSide:
		roundFunc = math.Floor
	}

	entries := make([]bookEntry, len(rawUpdates))
	for i, update := range rawUpdates {
		price, _ := strconv.ParseFloat(update[0], 64)
		amount, _ := strconv.ParseFloat(update[1], 64)

		entries[i] = bookEntry{
			price:  Price(roundFunc(price/float64(b.granularity))) * b.granularity,
			amount: Amount(amount),
		}
	}

	slices.SortFunc(entries, func(a, b bookEntry) int { return cmp.Compare(a.price, b.price) })

	for i := 0; i < len(entries); {
		aggEntry := bookEntry{entries[i].price, 0.0}

		for ; i < len(entries) && entries[i].price == aggEntry.price; i++ {
			aggEntry.amount += entries[i].amount
		}

		if aggEntry.amount == 0.0 {
			cur.Delete(aggEntry)
		} else {
			cur.ReplaceOrInsert(aggEntry)
		}
	}
}
