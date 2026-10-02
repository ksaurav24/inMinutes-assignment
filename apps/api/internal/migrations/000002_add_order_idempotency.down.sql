ALTER TABLE orders DROP CONSTRAINT orders_idempotency_key_key;
ALTER TABLE orders DROP COLUMN request_hash;
ALTER TABLE orders DROP COLUMN idempotency_key;
