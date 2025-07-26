package coinbase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/risingwavelabs/eris"

	"freyr/internal/database"
	"freyr/internal/database/querier"
	"freyr/internal/metrics"
)

const (
	// The timestamp of the oldest candle we care about.
	MinTimestamp int64 = 1577836800 // 2020-01-01 00:00:00 UTC

	// The width of a candle in seconds.
	Granularity int64 = 60

	// The maximum number of candles in a single request.
	MaxPayload = 300

	// The API endpoint provided by Coinbase.
	Url string = "https://api.exchange.coinbase.com/products/%s/candles/?granularity=%d&start=%d&end=%d"
)

// Collects the candles for a given pair since the latest candle.
func (svc *Coinbase) collectRecentCandles(ctx context.Context, pair string) error {
	start, err := svc.startTimestamp(ctx, pair)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	now -= now % Granularity

	for ; start <= now; start += Granularity * MaxPayload {
		candles, err := svc.downloadCandles(ctx, pair, start, Granularity)
		if err != nil {
			return err
		}

		err = database.InsertCandles(ctx, candles)
		if err != nil {
			return err
		}

		if len(candles) == 0 {
			continue
		}

		maxTS := time.Time{}
		for _, candle := range candles {
			if maxTS.Before(candle.Start) {
				maxTS = candle.Start
			}
		}
		metrics.LatestCandleCollected.
			WithLabelValues("coinbase", pair).
			Set(float64(maxTS.Unix()))
	}

	return nil
}

func (svc *Coinbase) startTimestamp(ctx context.Context, pair string) (int64, error) {
	lastOld, err := database.GetLatestCandle(ctx, pair)
	if err != nil {
		return 0, err
	}

	if lastOld != (time.Time{}) {
		return lastOld.Unix() + Granularity, nil
	}

	//
	// No timestamp known: Find first available day.

	const GranDay int64 = 24 * 60 * 60
	today := time.Now().Unix()
	today -= today % GranDay

	var candles []querier.InsertCandlesParams

	for start := MinTimestamp; start <= today && len(candles) == 0; start += GranDay * MaxPayload {
		candles, err = svc.downloadCandles(ctx, pair, MinTimestamp, GranDay)
		if err != nil {
			return 0, err
		}
	}

	minTS := today
	for _, candle := range candles {
		minTS = min(minTS, candle.Start.Unix())
	}

	return minTS, nil
}

func (svc *Coinbase) downloadCandles(ctx context.Context, pair string, start, granularity int64) ([]querier.InsertCandlesParams, error) {
	// Compute/normalise timestamps.
	start -= start % Granularity
	end := start + Granularity*(MaxPayload-1) // `end` is inclusive.

	//
	// Build and execute request.

	req, err := buildRequest(ctx, pair, start, end)
	if err != nil {
		return nil, err
	}

	metrics.LatestCandleQueried.
		WithLabelValues("coinbase", pair).
		Set(float64(end))

	res, err := svc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	//
	// Process response.

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	candles, err := parseCandles(pair, body)
	if err != nil {
		return nil, err
	}

	return candles, nil
}

func buildRequest(ctx context.Context, pair string, start, end int64) (*http.Request, error) {
	url := fmt.Sprintf(Url, pair, 60, start, end)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, eris.Wrap(err, "failed to build HTTP request")
	}
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func parseCandles(pair string, body []byte) ([]querier.InsertCandlesParams, error) {
	candles := []querier.InsertCandlesParams{}

	depth := 0
	for i := 0; i < len(body); i++ {
		char := body[i]

		switch depth {
		case 0:
			if char != '[' {
				return nil, eris.Errorf("failed to parse response: expected '[' but found '%c'", char)
			}
			depth++

		case 1:
			switch char {
			case '[':
				depth++
			case ']':
				depth--
			case ',':
				// ignore
			default:
				return nil, eris.Errorf("failed to parse response: unexpected character in response: %c", char)
			}

		case 2:
			// Find end of candle.
			j := i
			for ; j < len(body); j++ {
				if body[j] == ']' {
					break
				}
			}
			if j >= len(body) {
				return nil, eris.Errorf("failed to parse response: EOF before end of candle")
			}

			fields := strings.Split(string(body[i:j]), ",")

			timestamp, err := strconv.ParseInt(fields[0], 10, 64)
			if err != nil {
				return nil, eris.Wrapf(err, "failed to parse string as timestamp: '%s'", fields[0])
			}

			candle := querier.InsertCandlesParams{
				Pair:  pair,
				Start: time.Unix(timestamp, 0),
			}

			for i, dst := range []*float64{
				&candle.PriceLow, &candle.PriceHigh,
				&candle.PriceOpen, &candle.PriceClose,
				&candle.Volume,
			} {
				*dst, err = strconv.ParseFloat(fields[i+1], 64)
				if err != nil {
					return nil, eris.Wrapf(err, "failed to parse string as price/volume: '%s'", fields[i+1])
				}
			}

			candles = append(candles, candle)

			i = j
			depth-- // We already read ']'.
		}
	}

	return candles, nil
}
