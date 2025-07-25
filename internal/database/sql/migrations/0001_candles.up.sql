BEGIN;

CREATE TABLE candles
(
    pair         TEXT              NOT NULL,

    start        TIMESTAMPTZ       NOT NULL,

    -- Note that `double precision` is not the ideal datatype for trading data.
    -- Some exchanges explicitly advise against it. However, for our current
    -- goals, they are sufficient and easier to work with.

    price_open   double precision  NOT NULL,
    price_close  double precision  NOT NULL,
    price_low    double precision  NOT NULL,
    price_high   double precision  NOT NULL,

    volume       double precision  NOT NULL,

    PRIMARY KEY (pair, start)
);

COMMIT;
