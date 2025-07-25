package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/risingwavelabs/eris"

	"freyr/internal/database/querier"
)

func GetCandles(ctx context.Context, pair string, start, end time.Time) ([]*querier.Candle, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	res, err := model.GetCandles(ctx, querier.GetCandlesParams{
		Pair:      pair,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		return nil, eris.Wrapf(err, "failed to query candles")
	}

	return res, nil
}

func InsertCandles(ctx context.Context, candles []querier.InsertCandlesParams) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	_, err := model.InsertCandles(ctx, candles)
	if err != nil {
		return eris.Wrapf(err, "failed to insert candles")
	}

	return nil
}

func GetLatestCandle(ctx context.Context, pair string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	ts, err := model.GetLatestCandle(ctx, pair)
	if err == pgx.ErrNoRows {
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, eris.Wrapf(err, "failed to query latest candle %T", err)
	}

	return ts, nil
}
