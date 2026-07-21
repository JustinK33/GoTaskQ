-- Demo tables for the built-in sql.etl task.
-- These are optional and are intended for local development and demos.

CREATE SCHEMA IF NOT EXISTS raw;
CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS raw.orders (
    id              BIGSERIAL PRIMARY KEY,
    customer_id     TEXT        NOT NULL,
    order_total     NUMERIC     NOT NULL,
    status          TEXT        NOT NULL,
    ordered_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS analytics.daily_revenue (
    revenue_day     DATE        PRIMARY KEY,
    order_count     BIGINT      NOT NULL,
    gross_revenue   NUMERIC     NOT NULL,
    loaded_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO raw.orders (id, customer_id, order_total, status, ordered_at)
VALUES
    (1, 'cust-001', 42.50, 'paid', NOW() - INTERVAL '2 days'),
    (2, 'cust-002', 19.99, 'paid', NOW() - INTERVAL '2 days'),
    (3, 'cust-003', 75.00, 'refunded', NOW() - INTERVAL '1 day'),
    (4, 'cust-004', 120.00, 'paid', NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

SELECT setval('raw.orders_id_seq', GREATEST((SELECT MAX(id) FROM raw.orders), 1));
