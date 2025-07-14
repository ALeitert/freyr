package utils

import (
	"context"
	"iter"
)

// Creates an iterator over the given channel which discontinues the iteration
// if the given context is cancelled. It allows to use the channel within the
// head of a `for`-loop while still obeying the context. Hence, we can write
//
//	for obj := range CtxChanIter(ctx, ch) {
//		// ...
//	}
//
// instead of
//
//	for {
//		var obj T
//		var more bool
//		select {
//		case <-ctx.Done():
//			more = false
//		case obj, more = <-ch:
//		}
//		if !more {
//			break
//		}
//		// ...
//	}
func CtxChanIter[T any](ctx context.Context, ch <-chan T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case t, more := <-ch:
				if !more || !yield(t) {
					return
				}
			}
		}
	}
}
