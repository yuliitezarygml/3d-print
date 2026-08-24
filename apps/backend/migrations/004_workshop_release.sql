ALTER TABLE models
  ADD COLUMN IF NOT EXISTS estimated_print_minutes integer,
  ADD COLUMN IF NOT EXISTS estimated_filament_grams numeric(12,2),
  ADD COLUMN IF NOT EXISTS slicer_metadata jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'ADMIN',
  ADD COLUMN IF NOT EXISTS requested_material text,
  ADD COLUMN IF NOT EXISTS requested_color text,
  ADD COLUMN IF NOT EXISTS requested_quantity integer NOT NULL DEFAULT 1 CHECK (requested_quantity > 0);

ALTER TABLE print_jobs
  ADD COLUMN IF NOT EXISTS scheduled_start timestamptz,
  ADD COLUMN IF NOT EXISTS scheduled_end timestamptz;
CREATE INDEX IF NOT EXISTS print_jobs_schedule_idx ON print_jobs(scheduled_start, scheduled_end);

CREATE TABLE IF NOT EXISTS order_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  event_type text NOT NULL DEFAULT 'NOTE',
  status order_status,
  title text NOT NULL,
  message text NOT NULL DEFAULT '',
  is_public boolean NOT NULL DEFAULT true,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS order_events_order_idx ON order_events(order_id, created_at DESC);

CREATE OR REPLACE FUNCTION record_order_status_event() RETURNS trigger AS $$
DECLARE
  status_title text;
BEGIN
  IF TG_OP = 'INSERT' OR OLD.status IS DISTINCT FROM NEW.status THEN
    status_title := CASE NEW.status
      WHEN 'DRAFT' THEN 'Заявка получена'
      WHEN 'NEW' THEN 'Заказ принят'
      WHEN 'CONFIRMED' THEN 'Заказ подтверждён'
      WHEN 'WAITING' THEN 'Ожидаем материалы'
      WHEN 'READY_TO_PRINT' THEN 'Подготовлен к печати'
      WHEN 'PRINTING' THEN 'Печать началась'
      WHEN 'POST_PROCESSING' THEN 'Постобработка'
      WHEN 'READY' THEN 'Заказ готов'
      WHEN 'COMPLETED' THEN 'Заказ выдан'
      WHEN 'CANCELLED' THEN 'Заказ отменён'
      ELSE NEW.status::text
    END;
    INSERT INTO order_events(order_id,event_type,status,title,message,is_public)
    VALUES(NEW.id,'STATUS',NEW.status,status_title,'',true);
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS orders_status_history ON orders;
CREATE TRIGGER orders_status_history
AFTER INSERT OR UPDATE OF status ON orders
FOR EACH ROW EXECUTE FUNCTION record_order_status_event();

INSERT INTO order_events(order_id,event_type,status,title,message,is_public,created_at)
SELECT o.id,'STATUS',o.status,
  CASE o.status
    WHEN 'DRAFT' THEN 'Заявка получена'
    WHEN 'NEW' THEN 'Заказ принят'
    WHEN 'CONFIRMED' THEN 'Заказ подтверждён'
    WHEN 'WAITING' THEN 'Ожидаем материалы'
    WHEN 'READY_TO_PRINT' THEN 'Подготовлен к печати'
    WHEN 'PRINTING' THEN 'Печать началась'
    WHEN 'POST_PROCESSING' THEN 'Постобработка'
    WHEN 'READY' THEN 'Заказ готов'
    WHEN 'COMPLETED' THEN 'Заказ выдан'
    WHEN 'CANCELLED' THEN 'Заказ отменён'
    ELSE o.status::text
  END,'',true,o.created_at
FROM orders o
WHERE NOT EXISTS (SELECT 1 FROM order_events e WHERE e.order_id=o.id);

CREATE TABLE IF NOT EXISTS order_photos (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  event_id uuid REFERENCES order_events(id) ON DELETE SET NULL,
  storage_path text NOT NULL,
  original_filename text NOT NULL,
  mime_type text NOT NULL,
  file_size_bytes bigint NOT NULL,
  caption text NOT NULL DEFAULT '',
  is_public boolean NOT NULL DEFAULT true,
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS order_photos_order_idx ON order_photos(order_id, created_at DESC);

CREATE TABLE IF NOT EXISTS telegram_subscriptions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  chat_id bigint NOT NULL,
  last_notified_status order_status,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(order_id, chat_id)
);
CREATE INDEX IF NOT EXISTS telegram_subscriptions_order_idx ON telegram_subscriptions(order_id);
