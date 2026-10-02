ALTER TABLE orders ADD COLUMN idempotency_key TEXT;
ALTER TABLE orders ADD COLUMN request_hash TEXT;

UPDATE orders
SET
    idempotency_key = 'legacy-' || id,
    request_hash = 'legacy'
WHERE idempotency_key IS NULL;

ALTER TABLE orders ALTER COLUMN idempotency_key SET NOT NULL;
ALTER TABLE orders ALTER COLUMN request_hash SET NOT NULL;
ALTER TABLE orders ADD CONSTRAINT orders_idempotency_key_key UNIQUE (idempotency_key);
