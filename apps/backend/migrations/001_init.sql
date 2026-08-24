CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM ('ADMIN', 'OPERATOR');
CREATE TYPE printer_status AS ENUM ('IDLE', 'PRINTING', 'PAUSED', 'OFFLINE', 'ERROR', 'MAINTENANCE');
CREATE TYPE order_status AS ENUM ('DRAFT', 'NEW', 'CONFIRMED', 'WAITING', 'READY_TO_PRINT', 'PRINTING', 'POST_PROCESSING', 'READY', 'COMPLETED', 'CANCELLED');
CREATE TYPE print_job_status AS ENUM ('QUEUED', 'READY', 'PRINTING', 'PAUSED', 'SUCCESS', 'FAILED', 'CANCELLED');
CREATE TYPE inventory_transaction_type AS ENUM ('PURCHASE', 'PRINT_USAGE', 'MANUAL_ADJUSTMENT', 'WRITE_OFF', 'RETURN');

CREATE SEQUENCE order_number_seq START 2;

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  role user_role NOT NULL DEFAULT 'OPERATOR',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens(user_id);

CREATE TABLE settings (
  id boolean PRIMARY KEY DEFAULT true CHECK (id),
  company_name text NOT NULL DEFAULT 'PrintForge Studio',
  currency varchar(3) NOT NULL DEFAULT 'MDL',
  electricity_price_per_kwh numeric(12,4) NOT NULL DEFAULT 2.58 CHECK (electricity_price_per_kwh >= 0),
  machine_rate_per_hour numeric(12,2) NOT NULL DEFAULT 25 CHECK (machine_rate_per_hour >= 0),
  labour_rate_per_hour numeric(12,2) NOT NULL DEFAULT 50 CHECK (labour_rate_per_hour >= 0),
  default_markup_percent numeric(7,2) NOT NULL DEFAULT 40 CHECK (default_markup_percent >= 0),
  low_stock_threshold_grams numeric(12,2) NOT NULL DEFAULT 200 CHECK (low_stock_threshold_grams >= 0),
  updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO settings (id) VALUES (true);

CREATE TABLE printers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  manufacturer text NOT NULL,
  model text NOT NULL,
  serial_number text UNIQUE,
  status printer_status NOT NULL DEFAULT 'IDLE',
  technology text NOT NULL DEFAULT 'FDM',
  build_x_mm numeric(8,2) NOT NULL DEFAULT 0,
  build_y_mm numeric(8,2) NOT NULL DEFAULT 0,
  build_z_mm numeric(8,2) NOT NULL DEFAULT 0,
  nozzle_mm numeric(5,2) NOT NULL DEFAULT 0.4,
  power_watts numeric(10,2) NOT NULL DEFAULT 150 CHECK (power_watts >= 0),
  purchase_price numeric(14,2) NOT NULL DEFAULT 0 CHECK (purchase_price >= 0),
  depreciation_hours numeric(12,2) NOT NULL DEFAULT 5000 CHECK (depreciation_hours > 0),
  total_hours numeric(12,2) NOT NULL DEFAULT 0,
  location text,
  ip_address inet,
  notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX printers_status_idx ON printers(status);

CREATE TABLE printer_photos (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  printer_id uuid NOT NULL REFERENCES printers(id) ON DELETE CASCADE,
  path text NOT NULL,
  is_primary boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE printer_maintenance (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  printer_id uuid NOT NULL REFERENCES printers(id) ON DELETE CASCADE,
  performed_at date NOT NULL,
  maintenance_type text NOT NULL,
  description text NOT NULL,
  replaced_part text,
  cost numeric(14,2) NOT NULL DEFAULT 0,
  printer_hours numeric(12,2),
  next_service_at date,
  notes text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE customers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  company text,
  phone text,
  email text,
  telegram text,
  address text,
  notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX customers_name_idx ON customers USING gin (to_tsvector('simple', name));

CREATE TABLE filament_spools (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  manufacturer text NOT NULL,
  product_name text NOT NULL,
  material text NOT NULL,
  color_name text NOT NULL,
  color_hex varchar(7) NOT NULL DEFAULT '#808080',
  diameter_mm numeric(4,2) NOT NULL DEFAULT 1.75,
  initial_weight_grams numeric(12,2) NOT NULL CHECK (initial_weight_grams > 0),
  remaining_weight_grams numeric(12,2) NOT NULL CHECK (remaining_weight_grams >= 0),
  empty_spool_weight_grams numeric(12,2) NOT NULL DEFAULT 0,
  purchase_price numeric(14,2) NOT NULL CHECK (purchase_price >= 0),
  supplier text,
  storage_location text,
  lot_number text,
  status text NOT NULL DEFAULT 'ACTIVE',
  purchased_at date,
  notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX filament_spools_material_idx ON filament_spools(material);
CREATE INDEX filament_spools_remaining_idx ON filament_spools(remaining_weight_grams);

CREATE TABLE orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  number text NOT NULL UNIQUE,
  customer_id uuid REFERENCES customers(id) ON DELETE SET NULL,
  status order_status NOT NULL DEFAULT 'NEW',
  deadline timestamptz,
  selling_price numeric(14,2) NOT NULL DEFAULT 0,
  paid_amount numeric(14,2) NOT NULL DEFAULT 0,
  payment_method text,
  notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_status_idx ON orders(status);
CREATE INDEX orders_created_idx ON orders(created_at DESC);

CREATE TABLE models (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  original_filename text NOT NULL,
  storage_path text NOT NULL,
  mime_type text NOT NULL,
  file_size_bytes bigint NOT NULL,
  format varchar(8) NOT NULL,
  dimensions_x_mm numeric(12,3),
  dimensions_y_mm numeric(12,3),
  dimensions_z_mm numeric(12,3),
  volume_cm3 numeric(16,3),
  triangle_count bigint,
  version integer NOT NULL DEFAULT 1,
  notes text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE print_jobs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid REFERENCES orders(id) ON DELETE SET NULL,
  model_id uuid REFERENCES models(id) ON DELETE SET NULL,
  printer_id uuid NOT NULL REFERENCES printers(id) ON DELETE RESTRICT,
  spool_id uuid NOT NULL REFERENCES filament_spools(id) ON DELETE RESTRICT,
  quantity integer NOT NULL DEFAULT 1 CHECK (quantity > 0),
  status print_job_status NOT NULL DEFAULT 'QUEUED',
  estimated_minutes integer NOT NULL CHECK (estimated_minutes >= 0),
  actual_minutes integer CHECK (actual_minutes >= 0),
  estimated_filament_grams numeric(12,2) NOT NULL CHECK (estimated_filament_grams >= 0),
  actual_filament_grams numeric(12,2) CHECK (actual_filament_grams >= 0),
  power_watts numeric(10,2) NOT NULL CHECK (power_watts >= 0),
  electricity_price_per_kwh numeric(12,4) NOT NULL CHECK (electricity_price_per_kwh >= 0),
  estimated_energy_kwh numeric(14,5) NOT NULL DEFAULT 0,
  actual_energy_kwh numeric(14,5),
  material_cost numeric(14,2) NOT NULL DEFAULT 0,
  electricity_cost numeric(14,2) NOT NULL DEFAULT 0,
  machine_cost numeric(14,2) NOT NULL DEFAULT 0,
  labour_cost numeric(14,2) NOT NULL DEFAULT 0,
  post_processing_cost numeric(14,2) NOT NULL DEFAULT 0,
  packaging_cost numeric(14,2) NOT NULL DEFAULT 0,
  other_cost numeric(14,2) NOT NULL DEFAULT 0,
  total_cost numeric(14,2) NOT NULL DEFAULT 0,
  markup_percent numeric(7,2) NOT NULL DEFAULT 0,
  suggested_price numeric(14,2) NOT NULL DEFAULT 0,
  started_at timestamptz,
  completed_at timestamptz,
  notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX print_jobs_status_idx ON print_jobs(status);
CREATE INDEX print_jobs_printer_idx ON print_jobs(printer_id);

CREATE TABLE inventory_transactions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  spool_id uuid NOT NULL REFERENCES filament_spools(id) ON DELETE RESTRICT,
  print_job_id uuid REFERENCES print_jobs(id) ON DELETE SET NULL,
  type inventory_transaction_type NOT NULL,
  quantity_grams numeric(12,2) NOT NULL,
  balance_after_grams numeric(12,2) NOT NULL,
  reason text,
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX inventory_transactions_spool_idx ON inventory_transactions(spool_id, created_at DESC);

CREATE TABLE expenses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  category text NOT NULL,
  amount numeric(14,2) NOT NULL CHECK (amount >= 0),
  description text,
  spent_at date NOT NULL DEFAULT current_date,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  action text NOT NULL,
  entity_type text NOT NULL,
  entity_id uuid,
  old_values jsonb,
  new_values jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_entity_idx ON audit_logs(entity_type, entity_id, created_at DESC);
