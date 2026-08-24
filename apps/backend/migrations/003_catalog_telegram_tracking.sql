ALTER TABLE settings
  ADD COLUMN IF NOT EXISTS public_base_url text NOT NULL DEFAULT 'http://localhost',
  ADD COLUMN IF NOT EXISTS telegram_bot_token bytea,
  ADD COLUMN IF NOT EXISTS telegram_bot_username text,
  ADD COLUMN IF NOT EXISTS telegram_bot_enabled boolean NOT NULL DEFAULT false;

ALTER TABLE printers
  ADD COLUMN IF NOT EXISTS catalog_key text,
  ADD COLUMN IF NOT EXISTS image_url text;
CREATE INDEX IF NOT EXISTS printers_catalog_key_idx ON printers(catalog_key);

ALTER TABLE models
  ADD COLUMN IF NOT EXISTS customer_id uuid REFERENCES customers(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS preview_path text;
CREATE INDEX IF NOT EXISTS models_customer_idx ON models(customer_id, created_at DESC);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS tracking_code varchar(12);
UPDATE orders
SET tracking_code = upper(translate(substr(replace(id::text, '-', ''), 1, 10), '01', '23'))
WHERE tracking_code IS NULL;
ALTER TABLE orders ALTER COLUMN tracking_code SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS orders_tracking_code_idx ON orders(tracking_code);

CREATE TABLE IF NOT EXISTS order_models (
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  model_id uuid NOT NULL REFERENCES models(id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (order_id, model_id)
);
CREATE INDEX IF NOT EXISTS order_models_model_idx ON order_models(model_id);
