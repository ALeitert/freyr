-- name: GetCandles :many
SELECT * FROM candles WHERE pair = $1 AND start >= sqlc.arg(start_time) AND start < sqlc.arg(end_time);

-- name: InsertCandles :copyfrom
INSERT INTO candles (
    pair, start,
    price_open, price_close, price_low, price_high,
    volume
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetLatestCandle :one
SELECT start FROM candles WHERE pair = $1
ORDER BY start DESC  LIMIT 1;
