use std::path::Path;

use chrono::Utc;
use rand::Rng;
use rusqlite::{Connection, OptionalExtension, params};
use uuid::Uuid;

use crate::models::{
    Customer, Dashboard, DashboardPrinter, ImportModel, ModelAsset, NewCustomer, NewOrder,
    NewOrderEvent, NewPrintJob, NewPrinter, NewSpool, Order, OrderEvent, OrderModel, PrintJob,
    Printer, Settings, Spool,
};

const SCHEMA: &str = r#"
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  company_name TEXT NOT NULL,
  currency TEXT NOT NULL,
  electricity_price_micros INTEGER NOT NULL CHECK (electricity_price_micros >= 0),
  machine_rate_cents INTEGER NOT NULL CHECK (machine_rate_cents >= 0),
  labour_rate_cents INTEGER NOT NULL CHECK (labour_rate_cents >= 0),
  default_markup_basis_points INTEGER NOT NULL CHECK (default_markup_basis_points >= 0),
  low_stock_threshold_milligrams INTEGER NOT NULL CHECK (low_stock_threshold_milligrams >= 0)
);

CREATE TABLE IF NOT EXISTS customers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  company TEXT,
  phone TEXT,
  email TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS printers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  manufacturer TEXT NOT NULL,
  model TEXT NOT NULL,
  status TEXT NOT NULL,
  power_watts_milli INTEGER NOT NULL CHECK (power_watts_milli >= 0),
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS spools (
  id TEXT PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  material TEXT NOT NULL,
  color_name TEXT NOT NULL,
  color_hex TEXT NOT NULL,
  initial_milligrams INTEGER NOT NULL CHECK (initial_milligrams > 0),
  remaining_milligrams INTEGER NOT NULL CHECK (remaining_milligrams >= 0),
  purchase_price_cents INTEGER NOT NULL CHECK (purchase_price_cents >= 0),
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  number TEXT NOT NULL UNIQUE,
  tracking_code TEXT NOT NULL UNIQUE,
  customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  deadline TEXT,
  selling_price_cents INTEGER NOT NULL CHECK (selling_price_cents >= 0),
  paid_amount_cents INTEGER NOT NULL CHECK (paid_amount_cents >= 0),
  notes TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS print_jobs (
  id TEXT PRIMARY KEY,
  order_id TEXT REFERENCES orders(id) ON DELETE SET NULL,
  printer_id TEXT REFERENCES printers(id) ON DELETE SET NULL,
  spool_id TEXT REFERENCES spools(id) ON DELETE SET NULL,
  model_id TEXT,
  status TEXT NOT NULL,
  print_minutes INTEGER NOT NULL DEFAULT 0,
  filament_milligrams INTEGER NOT NULL DEFAULT 0,
  scheduled_start TEXT,
  scheduled_end TEXT,
  total_cost_cents INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  completed_at TEXT
);

CREATE TABLE IF NOT EXISTS models (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  original_filename TEXT NOT NULL,
  storage_path TEXT NOT NULL,
  format TEXT NOT NULL,
  file_size_bytes INTEGER NOT NULL CHECK (file_size_bytes >= 0),
  estimated_print_minutes INTEGER,
  estimated_filament_milligrams INTEGER,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_models (
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  model_id TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
  quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
  PRIMARY KEY (order_id, model_id)
);

CREATE TABLE IF NOT EXISTS order_events (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  title TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_models_created ON models(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_print_jobs_created ON print_jobs(created_at DESC);
"#;

pub struct Database {
    connection: Connection,
}

impl Database {
    pub fn open(path: &Path) -> Result<Self, String> {
        let connection = Connection::open(path).map_err(error)?;
        connection.execute_batch(SCHEMA).map_err(error)?;
        let database = Self { connection };
        database.migrate_legacy_schema()?;
        database.seed()?;
        Ok(database)
    }

    fn migrate_legacy_schema(&self) -> Result<(), String> {
        for (table, column, definition) in [
            ("printers", "catalog_key", "TEXT"),
            ("printers", "build_x_milli", "INTEGER"),
            ("printers", "build_y_milli", "INTEGER"),
            ("printers", "build_z_milli", "INTEGER"),
            (
                "printers",
                "purchase_price_cents",
                "INTEGER NOT NULL DEFAULT 0",
            ),
            (
                "printers",
                "depreciation_hours_milli",
                "INTEGER NOT NULL DEFAULT 5000000",
            ),
            (
                "printers",
                "total_hours_milli",
                "INTEGER NOT NULL DEFAULT 0",
            ),
            ("printers", "nozzle_microns", "INTEGER NOT NULL DEFAULT 400"),
            ("printers", "serial_number", "TEXT"),
            ("printers", "location", "TEXT"),
            ("spools", "manufacturer", "TEXT NOT NULL DEFAULT ''"),
            ("spools", "product_name", "TEXT NOT NULL DEFAULT ''"),
            ("spools", "supplier", "TEXT"),
            (
                "print_jobs",
                "printer_id",
                "TEXT REFERENCES printers(id) ON DELETE SET NULL",
            ),
            (
                "print_jobs",
                "spool_id",
                "TEXT REFERENCES spools(id) ON DELETE SET NULL",
            ),
            (
                "print_jobs",
                "model_id",
                "TEXT REFERENCES models(id) ON DELETE SET NULL",
            ),
            ("print_jobs", "print_minutes", "INTEGER NOT NULL DEFAULT 0"),
            (
                "print_jobs",
                "filament_milligrams",
                "INTEGER NOT NULL DEFAULT 0",
            ),
            ("print_jobs", "scheduled_start", "TEXT"),
            ("print_jobs", "scheduled_end", "TEXT"),
            (
                "print_jobs",
                "total_cost_cents",
                "INTEGER NOT NULL DEFAULT 0",
            ),
            ("print_jobs", "completed_at", "TEXT"),
            (
                "print_jobs",
                "electricity_cost_cents",
                "INTEGER NOT NULL DEFAULT 0",
            ),
            ("print_jobs", "energy_microwh", "INTEGER NOT NULL DEFAULT 0"),
            (
                "print_jobs",
                "suggested_price_cents",
                "INTEGER NOT NULL DEFAULT 0",
            ),
            ("models", "preview_path", "TEXT"),
        ] {
            if !self.has_column(table, column)? {
                self.connection
                    .execute_batch(&format!(
                        "ALTER TABLE {table} ADD COLUMN {column} {definition}"
                    ))
                    .map_err(error)?;
            }
        }
        Ok(())
    }

    fn has_column(&self, table: &str, column: &str) -> Result<bool, String> {
        let mut statement = self
            .connection
            .prepare(&format!("PRAGMA table_info({table})"))
            .map_err(error)?;
        let names = statement
            .query_map([], |row| row.get::<_, String>(1))
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)?;
        Ok(names.iter().any(|name| name == column))
    }

    fn seed(&self) -> Result<(), String> {
        self.connection
            .execute(
                "INSERT OR IGNORE INTO settings VALUES (1, ?1, ?2, ?3, ?4, ?5, ?6, ?7)",
                params![
                    "PrintForge Studio",
                    "MDL",
                    2_580_000i64,
                    2_500i64,
                    5_000i64,
                    400_000i64,
                    200_000i64
                ],
            )
            .map_err(error)?;

        let count: i64 = self
            .connection
            .query_row("SELECT COUNT(*) FROM printers", [], |row| row.get(0))
            .map_err(error)?;
        if count == 0 {
            let now = Utc::now().to_rfc3339();
            for (name, manufacturer, model, status, watts) in [
                (
                    "X1 Carbon",
                    "Bambu Lab",
                    "X1 Carbon",
                    "PRINTING",
                    350_000i64,
                ),
                ("P1S", "Bambu Lab", "P1S", "IDLE", 300_000i64),
                ("MK4", "Prusa", "MK4", "IDLE", 120_000i64),
            ] {
                self.connection
                    .execute(
                        "INSERT INTO printers (id,name,manufacturer,model,status,power_watts_milli,created_at) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)",
                        params![
                            Uuid::new_v4().to_string(),
                            name,
                            manufacturer,
                            model,
                            status,
                            watts,
                            now
                        ],
                    )
                    .map_err(error)?;
            }
        }

        let customer_count: i64 = self
            .connection
            .query_row("SELECT COUNT(*) FROM customers", [], |row| row.get(0))
            .map_err(error)?;
        if customer_count == 0 {
            let now = Utc::now().to_rfc3339();
            self.connection
                .execute(
                    "INSERT INTO customers VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
                    params![
                        Uuid::new_v4().to_string(),
                        "Demo Customer",
                        "Local Workshop",
                        "+373 00 000 000",
                        "demo@example.com",
                        now
                    ],
                )
                .map_err(error)?;
        }

        let spool_count: i64 = self
            .connection
            .query_row("SELECT COUNT(*) FROM spools", [], |row| row.get(0))
            .map_err(error)?;
        if spool_count == 0 {
            self.connection
                .execute(
                    "INSERT INTO spools (id,code,material,color_name,color_hex,initial_milligrams,remaining_milligrams,purchase_price_cents,created_at,manufacturer,product_name) VALUES (?1, 'SP-0001', 'PLA', 'Graphite', '#232825', 1000000, 740000, 45000, ?2, 'Bambu Lab', 'PLA Basic')",
                    params![Uuid::new_v4().to_string(), Utc::now().to_rfc3339()],
                )
                .map_err(error)?;
        }
        Ok(())
    }

    pub fn dashboard(&self) -> Result<Dashboard, String> {
        let settings = self.settings()?;
        let active_orders = self
            .scalar("SELECT COUNT(*) FROM orders WHERE status NOT IN ('COMPLETED', 'CANCELLED')")?;
        let queued_jobs = self.scalar(
            "SELECT COUNT(*) FROM print_jobs WHERE status IN ('QUEUED', 'READY', 'PRINTING')",
        )?;
        let available_printers =
            self.scalar("SELECT COUNT(*) FROM printers WHERE status = 'IDLE'")?;
        let printing_printers =
            self.scalar("SELECT COUNT(*) FROM printers WHERE status = 'PRINTING'")?;
        let maintenance_printers =
            self.scalar("SELECT COUNT(*) FROM printers WHERE status = 'MAINTENANCE'")?;
        let low_stock_spools = self
            .connection
            .query_row(
                "SELECT COUNT(*) FROM spools WHERE remaining_milligrams <= ?1",
                params![(settings.low_stock_threshold_grams * 1000.0).round() as i64],
                |row| row.get(0),
            )
            .map_err(error)?;
        let (revenue_cents, paid_cents): (i64, i64) = self.connection.query_row(
            "SELECT COALESCE(SUM(selling_price_cents), 0), COALESCE(SUM(paid_amount_cents), 0) FROM orders WHERE status != 'CANCELLED'",
            [],
            |row| Ok((row.get(0)?, row.get(1)?)),
        ).map_err(error)?;
        let (spool_count, filament_milligrams, stock_value_cents): (i64, i64, i64) = self.connection.query_row(
            "SELECT COUNT(*), COALESCE(SUM(remaining_milligrams),0), COALESCE(SUM(remaining_milligrams * purchase_price_cents / initial_milligrams),0) FROM spools",
            [], |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?)),
        ).map_err(error)?;
        let (production_cost_cents, electricity_cost_cents): (i64, i64) = self.connection.query_row(
            "SELECT COALESCE(SUM(total_cost_cents),0), COALESCE(SUM(electricity_cost_cents),0) FROM print_jobs WHERE status='COMPLETED'",
            [], |row| Ok((row.get(0)?, row.get(1)?)),
        ).map_err(error)?;
        let mut printer_statement = self.connection.prepare(
            "SELECT p.id,p.name,p.manufacturer,p.status,o.number,m.name FROM printers p LEFT JOIN print_jobs j ON j.printer_id=p.id AND j.status='PRINTING' LEFT JOIN orders o ON o.id=j.order_id LEFT JOIN models m ON m.id=j.model_id ORDER BY p.name"
        ).map_err(error)?;
        let printers = printer_statement
            .query_map([], |row| {
                Ok(DashboardPrinter {
                    id: row.get(0)?,
                    name: row.get(1)?,
                    manufacturer: row.get(2)?,
                    status: row.get(3)?,
                    order_number: row.get(4)?,
                    model_name: row.get(5)?,
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)?;

        Ok(Dashboard {
            active_orders,
            queued_jobs,
            available_printers,
            printing_printers,
            maintenance_printers,
            low_stock_spools,
            spool_count,
            filament_grams: filament_milligrams as f64 / 1000.0,
            stock_value: cents(stock_value_cents),
            revenue: cents(revenue_cents),
            outstanding: cents((revenue_cents - paid_cents).max(0)),
            production_cost: cents(production_cost_cents),
            profit: cents(revenue_cents - production_cost_cents),
            electricity_cost: cents(electricity_cost_cents),
            currency: settings.currency,
            printers,
        })
    }

    fn scalar(&self, sql: &str) -> Result<i64, String> {
        self.connection
            .query_row(sql, [], |row| row.get(0))
            .map_err(error)
    }

    pub fn customers(&self) -> Result<Vec<Customer>, String> {
        let mut statement = self
            .connection
            .prepare("SELECT c.id,c.name,c.company,c.phone,c.email,(SELECT COUNT(*) FROM orders o WHERE o.customer_id=c.id),(SELECT COALESCE(SUM(o.selling_price_cents),0) FROM orders o WHERE o.customer_id=c.id AND o.status!='CANCELLED'),(SELECT COUNT(*) FROM models m WHERE m.customer_id=c.id) FROM customers c ORDER BY c.name")
            .map_err(error)?;
        statement
            .query_map([], |row| {
                Ok(Customer {
                    id: row.get(0)?,
                    name: row.get(1)?,
                    company: row.get(2)?,
                    phone: row.get(3)?,
                    email: row.get(4)?,
                    order_count: row.get(5)?,
                    total_amount: cents(row.get(6)?),
                    model_count: row.get(7)?,
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)
    }

    pub fn printers(&self) -> Result<Vec<Printer>, String> {
        let mut statement = self.connection.prepare("SELECT id,name,manufacturer,model,status,power_watts_milli,catalog_key,build_x_milli,build_y_milli,build_z_milli,purchase_price_cents,depreciation_hours_milli,total_hours_milli,nozzle_microns,serial_number,location FROM printers ORDER BY name").map_err(error)?;
        statement
            .query_map([], |row| {
                Ok(Printer {
                    id: row.get(0)?,
                    name: row.get(1)?,
                    manufacturer: row.get(2)?,
                    model: row.get(3)?,
                    status: row.get(4)?,
                    power_watts: row.get::<_, i64>(5)? as f64 / 1000.0,
                    catalog_key: row.get(6)?,
                    build_x_mm: row
                        .get::<_, Option<i64>>(7)?
                        .map(|value| value as f64 / 1000.0),
                    build_y_mm: row
                        .get::<_, Option<i64>>(8)?
                        .map(|value| value as f64 / 1000.0),
                    build_z_mm: row
                        .get::<_, Option<i64>>(9)?
                        .map(|value| value as f64 / 1000.0),
                    purchase_price: cents(row.get(10)?),
                    depreciation_hours: row.get::<_, i64>(11)? as f64 / 1000.0,
                    total_hours: row.get::<_, i64>(12)? as f64 / 1000.0,
                    nozzle_mm: row.get::<_, i64>(13)? as f64 / 1000.0,
                    serial_number: row.get(14)?,
                    location: row.get(15)?,
                    image_url: None,
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)
    }

    pub fn create_printer(&self, input: NewPrinter) -> Result<Printer, String> {
        if input.name.trim().is_empty()
            || input.manufacturer.trim().is_empty()
            || input.model.trim().is_empty()
        {
            return Err("printer name, manufacturer and model are required".into());
        }
        if !input.power_watts.is_finite() || input.power_watts < 0.0 {
            return Err("printer power must be finite and non-negative".into());
        }
        let id = Uuid::new_v4().to_string();
        self.connection.execute(
            "INSERT INTO printers (id,name,manufacturer,model,status,power_watts_milli,created_at,catalog_key,build_x_milli,build_y_milli,build_z_milli,purchase_price_cents,depreciation_hours_milli,nozzle_microns,serial_number,location) VALUES (?1,?2,?3,?4,'IDLE',?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15)",
            params![id, input.name.trim(), input.manufacturer.trim(), input.model.trim(), scaled(input.power_watts, 1000.0)?, Utc::now().to_rfc3339(), input.catalog_key, optional_scaled(input.build_x_mm, 1000.0)?, optional_scaled(input.build_y_mm, 1000.0)?, optional_scaled(input.build_z_mm, 1000.0)?, to_cents(input.purchase_price)?, scaled(input.depreciation_hours.unwrap_or(5000.0),1000.0)?, scaled(input.nozzle_mm.unwrap_or(0.4),1000.0)?, input.serial_number, input.location],
        ).map_err(error)?;
        self.printers()?
            .into_iter()
            .find(|printer| printer.id == id)
            .ok_or_else(|| "printer not found after insert".into())
    }

    pub fn spools(&self) -> Result<Vec<Spool>, String> {
        let mut statement = self.connection.prepare("SELECT id,code,material,color_name,color_hex,initial_milligrams,remaining_milligrams,purchase_price_cents,manufacturer,product_name,supplier FROM spools ORDER BY code").map_err(error)?;
        statement
            .query_map([], |row| {
                let initial: i64 = row.get(5)?;
                let remaining: i64 = row.get(6)?;
                let purchase: i64 = row.get(7)?;
                Ok(Spool {
                    id: row.get(0)?,
                    code: row.get(1)?,
                    material: row.get(2)?,
                    color_name: row.get(3)?,
                    color_hex: row.get(4)?,
                    remaining_grams: remaining as f64 / 1000.0,
                    initial_grams: initial as f64 / 1000.0,
                    purchase_price: cents(purchase),
                    price_per_gram: if initial == 0 {
                        0.0
                    } else {
                        purchase as f64 / (initial as f64 / 1000.0) / 100.0
                    },
                    stock_value: cents((remaining * purchase) / initial.max(1)),
                    manufacturer: row.get(8)?,
                    product_name: row.get(9)?,
                    supplier: row.get(10)?,
                    status: if remaining == 0 {
                        "EMPTY".into()
                    } else {
                        "ACTIVE".into()
                    },
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)
    }

    pub fn create_spool(&self, input: NewSpool) -> Result<Spool, String> {
        if input.code.trim().is_empty()
            || input.material.trim().is_empty()
            || input.color_name.trim().is_empty()
        {
            return Err("spool code, material and color are required".into());
        }
        if !input.initial_grams.is_finite() || input.initial_grams <= 0.0 {
            return Err("initial spool weight must be positive".into());
        }
        let id = Uuid::new_v4().to_string();
        let milligrams = scaled(input.initial_grams, 1000.0)?;
        self.connection.execute(
            "INSERT INTO spools (id,code,material,color_name,color_hex,initial_milligrams,remaining_milligrams,purchase_price_cents,created_at,manufacturer,product_name,supplier) VALUES (?1,?2,?3,?4,?5,?6,?6,?7,?8,?9,?10,?11)",
            params![id, input.code.trim(), input.material.trim(), input.color_name.trim(), input.color_hex.trim(), milligrams, to_cents(input.purchase_price)?, Utc::now().to_rfc3339(), input.manufacturer.trim(), input.product_name.trim(), input.supplier],
        ).map_err(error)?;
        self.spools()?
            .into_iter()
            .find(|spool| spool.id == id)
            .ok_or_else(|| "spool not found after insert".into())
    }

    pub fn create_customer(&self, input: NewCustomer) -> Result<Customer, String> {
        if input.name.trim().is_empty() {
            return Err("customer name is required".into());
        }
        let id = Uuid::new_v4().to_string();
        self.connection.execute(
            "INSERT INTO customers (id,name,company,phone,email,created_at) VALUES (?1,?2,?3,?4,?5,?6)",
            params![id, input.name.trim(), input.company, input.phone, input.email, Utc::now().to_rfc3339()],
        ).map_err(error)?;
        self.customers()?
            .into_iter()
            .find(|customer| customer.id == id)
            .ok_or_else(|| "customer not found after insert".into())
    }

    pub fn models(&self) -> Result<Vec<ModelAsset>, String> {
        let mut statement = self.connection.prepare(
            "SELECT m.id,m.customer_id,COALESCE(c.name,'Без клиента'),m.name,m.original_filename,m.format,m.file_size_bytes,m.estimated_print_minutes,m.estimated_filament_milligrams,m.created_at,m.preview_path FROM models m LEFT JOIN customers c ON c.id=m.customer_id ORDER BY m.created_at DESC"
        ).map_err(error)?;
        statement
            .query_map([], |row| {
                Ok(ModelAsset {
                    id: row.get(0)?,
                    customer_id: row.get(1)?,
                    customer_name: row.get(2)?,
                    name: row.get(3)?,
                    original_filename: row.get(4)?,
                    format: row.get(5)?,
                    file_size_bytes: row.get(6)?,
                    estimated_print_minutes: row.get(7)?,
                    estimated_filament_grams: row
                        .get::<_, Option<i64>>(8)?
                        .map(|value| value as f64 / 1000.0),
                    created_at: row.get(9)?,
                    preview_path: row.get(10)?,
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)
    }

    pub fn import_model(
        &self,
        input: ImportModel,
        models_directory: &Path,
    ) -> Result<ModelAsset, String> {
        let source = Path::new(&input.source_path);
        if !source.is_file() {
            return Err("selected model file does not exist".into());
        }
        let extension = source
            .extension()
            .and_then(|value| value.to_str())
            .unwrap_or("")
            .to_ascii_lowercase();
        if !["stl", "obj", "3mf", "gcode", "gco", "gc"].contains(&extension.as_str()) {
            return Err("supported formats: STL, OBJ, 3MF and G-code".into());
        }
        if let Some(customer_id) = &input.customer_id
            && !self.customer_exists(customer_id)?
        {
            return Err("customer not found".into());
        }
        std::fs::create_dir_all(models_directory).map_err(|error| error.to_string())?;
        let id = Uuid::new_v4().to_string();
        let original = source
            .file_name()
            .and_then(|value| value.to_str())
            .ok_or_else(|| "invalid filename".to_string())?;
        let destination = models_directory.join(format!("{id}.{extension}"));
        std::fs::copy(source, &destination)
            .map_err(|error| format!("could not copy model: {error}"))?;
        let metadata = std::fs::metadata(&destination).map_err(|error| error.to_string())?;
        let display_name = input
            .name
            .filter(|value| !value.trim().is_empty())
            .unwrap_or_else(|| {
                source
                    .file_stem()
                    .and_then(|value| value.to_str())
                    .unwrap_or(original)
                    .to_string()
            });
        let (minutes, filament_milligrams) = if ["gcode", "gco", "gc"].contains(&extension.as_str())
        {
            parse_gcode_metadata(&destination)
        } else {
            (None, None)
        };
        self.connection.execute(
            "INSERT INTO models (id,customer_id,name,original_filename,storage_path,format,file_size_bytes,estimated_print_minutes,estimated_filament_milligrams,created_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10)",
            params![id,input.customer_id,display_name,original,destination.to_string_lossy(),extension.to_uppercase(),metadata.len() as i64,minutes,filament_milligrams,Utc::now().to_rfc3339()],
        ).map_err(error)?;
        self.models()?
            .into_iter()
            .find(|model| model.id == id)
            .ok_or_else(|| "model not found after import".into())
    }

    fn customer_exists(&self, id: &str) -> Result<bool, String> {
        Ok(self
            .connection
            .query_row("SELECT 1 FROM customers WHERE id=?1", [id], |_| Ok(()))
            .optional()
            .map_err(error)?
            .is_some())
    }

    pub fn model_storage_path(&self, id: &str) -> Result<String, String> {
        self.connection
            .query_row("SELECT storage_path FROM models WHERE id=?1", [id], |row| {
                row.get(0)
            })
            .map_err(error)
    }

    pub fn print_jobs(&self) -> Result<Vec<PrintJob>, String> {
        let mut statement = self.connection.prepare(
            "SELECT j.id,j.order_id,o.number,j.printer_id,p.name,j.spool_id,s.code,j.model_id,m.name,j.status,j.print_minutes,j.filament_milligrams,j.scheduled_start,j.scheduled_end,j.total_cost_cents,j.electricity_cost_cents,j.energy_microwh,j.suggested_price_cents,j.created_at FROM print_jobs j LEFT JOIN orders o ON o.id=j.order_id LEFT JOIN printers p ON p.id=j.printer_id LEFT JOIN spools s ON s.id=j.spool_id LEFT JOIN models m ON m.id=j.model_id ORDER BY j.created_at DESC"
        ).map_err(error)?;
        statement
            .query_map([], |row| {
                Ok(PrintJob {
                    id: row.get(0)?,
                    order_id: row.get(1)?,
                    order_number: row.get(2)?,
                    printer_id: row.get(3)?,
                    printer_name: row.get(4)?,
                    spool_id: row.get(5)?,
                    spool_code: row.get(6)?,
                    model_id: row.get(7)?,
                    model_name: row.get(8)?,
                    status: row.get(9)?,
                    print_minutes: row.get(10)?,
                    filament_grams: row.get::<_, i64>(11)? as f64 / 1000.0,
                    scheduled_start: row.get(12)?,
                    scheduled_end: row.get(13)?,
                    total_cost: cents(row.get(14)?),
                    electricity_cost: cents(row.get(15)?),
                    energy_kwh: row.get::<_, i64>(16)? as f64 / 1_000_000.0,
                    suggested_price: cents(row.get(17)?),
                    created_at: row.get(18)?,
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)
    }

    pub fn create_print_job(&self, input: NewPrintJob) -> Result<PrintJob, String> {
        if input.print_minutes < 0
            || !input.filament_grams.is_finite()
            || input.filament_grams < 0.0
        {
            return Err("print time and filament must be non-negative".into());
        }
        let id = Uuid::new_v4().to_string();
        self.connection.execute(
            "INSERT INTO print_jobs (id,order_id,printer_id,spool_id,model_id,status,print_minutes,filament_milligrams,scheduled_start,scheduled_end,total_cost_cents,electricity_cost_cents,energy_microwh,suggested_price_cents,created_at) VALUES (?1,?2,?3,?4,?5,'QUEUED',?6,?7,?8,?9,?10,?11,?12,?13,?14)",
            params![id,input.order_id,input.printer_id,input.spool_id,input.model_id,input.print_minutes,scaled(input.filament_grams,1000.0)?,input.scheduled_start,input.scheduled_end,to_cents(input.total_cost)?,to_cents(input.electricity_cost.unwrap_or(0.0))?,scaled(input.energy_kwh.unwrap_or(0.0),1_000_000.0)?,to_cents(input.suggested_price.unwrap_or(input.total_cost))?,Utc::now().to_rfc3339()],
        ).map_err(error)?;
        self.print_jobs()?
            .into_iter()
            .find(|job| job.id == id)
            .ok_or_else(|| "print job not found after insert".into())
    }

    pub fn complete_print_job(&mut self, id: &str) -> Result<PrintJob, String> {
        let transaction = self.connection.transaction().map_err(error)?;
        let (spool_id, filament): (Option<String>, i64) = transaction
            .query_row(
                "SELECT spool_id,filament_milligrams FROM print_jobs WHERE id=?1",
                [id],
                |row| Ok((row.get(0)?, row.get(1)?)),
            )
            .map_err(error)?;
        if let Some(spool_id) = spool_id {
            let changed = transaction.execute("UPDATE spools SET remaining_milligrams=remaining_milligrams-?1 WHERE id=?2 AND remaining_milligrams>=?1", params![filament,spool_id]).map_err(error)?;
            if changed == 0 {
                return Err("not enough filament on the selected spool".into());
            }
        }
        transaction
            .execute(
                "UPDATE print_jobs SET status='COMPLETED',completed_at=?1 WHERE id=?2",
                params![Utc::now().to_rfc3339(), id],
            )
            .map_err(error)?;
        transaction.commit().map_err(error)?;
        self.print_jobs()?
            .into_iter()
            .find(|job| job.id == id)
            .ok_or_else(|| "print job not found".into())
    }

    pub fn start_print_job(&self, id: &str) -> Result<PrintJob, String> {
        let changed = self.connection.execute("UPDATE print_jobs SET status='PRINTING' WHERE id=?1 AND status IN ('QUEUED','READY')", [id]).map_err(error)?;
        if changed == 0 {
            return Err("print job is not ready to start".into());
        }
        self.print_jobs()?
            .into_iter()
            .find(|job| job.id == id)
            .ok_or_else(|| "print job not found".into())
    }

    pub fn orders(&self) -> Result<Vec<Order>, String> {
        let mut statement = self.connection.prepare(
            "SELECT o.id, o.number, o.tracking_code, o.customer_id, COALESCE(c.name, 'Без клиента'), o.title, o.status, o.deadline, o.selling_price_cents, o.paid_amount_cents, o.created_at FROM orders o LEFT JOIN customers c ON c.id = o.customer_id ORDER BY o.created_at DESC"
        ).map_err(error)?;
        let mut orders = statement
            .query_map([], map_order)
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)?;
        for order in &mut orders {
            order.models = self.order_models(&order.id)?;
        }
        Ok(orders)
    }

    pub fn create_order(&self, input: NewOrder) -> Result<Order, String> {
        let title = input.title.trim();
        let model_ids = input.model_ids.clone();
        if title.is_empty() {
            return Err("order title is required".into());
        }
        if input.selling_price < 0.0
            || input.paid_amount < 0.0
            || input.paid_amount > input.selling_price
        {
            return Err("payment values are invalid".into());
        }
        if let Some(customer_id) = &input.customer_id {
            let exists: Option<String> = self
                .connection
                .query_row(
                    "SELECT id FROM customers WHERE id = ?1",
                    [customer_id],
                    |row| row.get(0),
                )
                .optional()
                .map_err(error)?;
            if exists.is_none() {
                return Err("customer not found".into());
            }
        }

        let next: i64 = self.scalar("SELECT COUNT(*) + 1 FROM orders")?;
        let number = format!("ORD-{}-{next:05}", Utc::now().format("%Y"));
        let id = Uuid::new_v4().to_string();
        let tracking_code = tracking_code();
        let now = Utc::now().to_rfc3339();
        self.connection.execute(
            "INSERT INTO orders (id, number, tracking_code, customer_id, title, status, deadline, selling_price_cents, paid_amount_cents, notes, created_at, updated_at) VALUES (?1, ?2, ?3, ?4, ?5, 'NEW', ?6, ?7, ?8, ?9, ?10, ?10)",
            params![id, number, tracking_code, input.customer_id, title, input.deadline, to_cents(input.selling_price)?, to_cents(input.paid_amount)?, input.notes, now],
        ).map_err(error)?;
        for model_id in model_ids {
            self.connection.execute("INSERT OR IGNORE INTO order_models (order_id,model_id,quantity) VALUES (?1,?2,1)", params![id,model_id]).map_err(error)?;
        }
        self.connection.execute(
            "INSERT INTO order_events (id,order_id,event_type,title,message,created_at) VALUES (?1,?2,'STATUS','Заказ принят','',?3)",
            params![Uuid::new_v4().to_string(),id,now],
        ).map_err(error)?;
        self.order_by_id(&id)
    }

    pub fn update_order_status(&self, id: &str, status: &str) -> Result<Order, String> {
        const ALLOWED: &[&str] = &[
            "NEW",
            "CONFIRMED",
            "WAITING",
            "READY_TO_PRINT",
            "PRINTING",
            "POST_PROCESSING",
            "READY",
            "COMPLETED",
            "CANCELLED",
        ];
        if !ALLOWED.contains(&status) {
            return Err("unsupported order status".into());
        }
        let changed = self
            .connection
            .execute(
                "UPDATE orders SET status = ?1, updated_at = ?2 WHERE id = ?3",
                params![status, Utc::now().to_rfc3339(), id],
            )
            .map_err(error)?;
        if changed == 0 {
            return Err("order not found".into());
        }
        self.connection.execute(
            "INSERT INTO order_events (id,order_id,event_type,title,message,created_at) VALUES (?1,?2,'STATUS',?3,'',?4)",
            params![Uuid::new_v4().to_string(),id,status,Utc::now().to_rfc3339()],
        ).map_err(error)?;
        self.order_by_id(id)
    }

    fn order_by_id(&self, id: &str) -> Result<Order, String> {
        let mut order = self.connection.query_row(
            "SELECT o.id, o.number, o.tracking_code, o.customer_id, COALESCE(c.name, 'Без клиента'), o.title, o.status, o.deadline, o.selling_price_cents, o.paid_amount_cents, o.created_at FROM orders o LEFT JOIN customers c ON c.id = o.customer_id WHERE o.id = ?1",
            [id], map_order,
        ).map_err(error)?;
        order.models = self.order_models(id)?;
        Ok(order)
    }

    fn order_models(&self, order_id: &str) -> Result<Vec<OrderModel>, String> {
        let mut statement = self.connection.prepare("SELECT m.id,m.name,m.original_filename,m.format FROM order_models om JOIN models m ON m.id=om.model_id WHERE om.order_id=?1 ORDER BY m.name").map_err(error)?;
        statement
            .query_map([order_id], |row| {
                Ok(OrderModel {
                    id: row.get(0)?,
                    name: row.get(1)?,
                    original_filename: row.get(2)?,
                    format: row.get(3)?,
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)
    }

    pub fn order_events(&self, order_id: &str) -> Result<Vec<OrderEvent>, String> {
        let mut statement = self.connection.prepare("SELECT id,order_id,event_type,title,message,created_at FROM order_events WHERE order_id=?1 ORDER BY created_at").map_err(error)?;
        statement
            .query_map([order_id], |row| {
                Ok(OrderEvent {
                    id: row.get(0)?,
                    order_id: row.get(1)?,
                    event_type: row.get(2)?,
                    title: row.get(3)?,
                    message: row.get(4)?,
                    created_at: row.get(5)?,
                })
            })
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)
    }

    pub fn add_order_event(
        &self,
        order_id: &str,
        input: NewOrderEvent,
    ) -> Result<OrderEvent, String> {
        if input.title.trim().is_empty() {
            return Err("event title is required".into());
        }
        let id = Uuid::new_v4().to_string();
        let now = Utc::now().to_rfc3339();
        self.connection.execute("INSERT INTO order_events (id,order_id,event_type,title,message,created_at) VALUES (?1,?2,'NOTE',?3,?4,?5)",params![id,order_id,input.title.trim(),input.message.unwrap_or_default(),now]).map_err(error)?;
        self.order_events(order_id)?
            .into_iter()
            .find(|event| event.id == id)
            .ok_or_else(|| "event not found".into())
    }

    pub fn receipt_order(&self, id: &str) -> Result<Order, String> {
        self.order_by_id(id)
    }

    pub fn settings(&self) -> Result<Settings, String> {
        self.connection.query_row("SELECT company_name, currency, electricity_price_micros, machine_rate_cents, labour_rate_cents, default_markup_basis_points, low_stock_threshold_milligrams FROM settings WHERE id = 1", [], |row| Ok(Settings {
            company_name: row.get(0)?, currency: row.get(1)?, electricity_price_per_kwh: row.get::<_, i64>(2)? as f64 / 1_000_000.0, machine_rate_per_hour: cents(row.get(3)?), labour_rate_per_hour: cents(row.get(4)?), default_markup_percent: row.get::<_, i64>(5)? as f64 / 10_000.0, low_stock_threshold_grams: row.get::<_, i64>(6)? as f64 / 1000.0,
        })).map_err(error)
    }

    pub fn save_settings(&self, settings: Settings) -> Result<Settings, String> {
        if settings.company_name.trim().is_empty() || settings.currency.trim().len() != 3 {
            return Err("company name and 3-letter currency are required".into());
        }
        let non_negative = [
            settings.electricity_price_per_kwh,
            settings.machine_rate_per_hour,
            settings.labour_rate_per_hour,
            settings.default_markup_percent,
            settings.low_stock_threshold_grams,
        ];
        if non_negative
            .iter()
            .any(|value| !value.is_finite() || *value < 0.0)
        {
            return Err("settings values must be finite and non-negative".into());
        }
        self.connection.execute("UPDATE settings SET company_name=?1, currency=?2, electricity_price_micros=?3, machine_rate_cents=?4, labour_rate_cents=?5, default_markup_basis_points=?6, low_stock_threshold_milligrams=?7 WHERE id=1", params![settings.company_name.trim(), settings.currency.trim().to_uppercase(), (settings.electricity_price_per_kwh * 1_000_000.0).round() as i64, to_cents(settings.machine_rate_per_hour)?, to_cents(settings.labour_rate_per_hour)?, (settings.default_markup_percent * 10_000.0).round() as i64, (settings.low_stock_threshold_grams * 1000.0).round() as i64]).map_err(error)?;
        self.settings()
    }
}

fn map_order(row: &rusqlite::Row<'_>) -> rusqlite::Result<Order> {
    Ok(Order {
        id: row.get(0)?,
        number: row.get(1)?,
        tracking_code: row.get(2)?,
        customer_id: row.get(3)?,
        customer_name: row.get(4)?,
        title: row.get(5)?,
        status: row.get(6)?,
        deadline: row.get(7)?,
        selling_price: cents(row.get(8)?),
        paid_amount: cents(row.get(9)?),
        created_at: row.get(10)?,
        models: Vec::new(),
    })
}

fn tracking_code() -> String {
    const ALPHABET: &[u8] = b"23456789ABCDEFGHJKLMNPQRSTUVWXYZ";
    let mut rng = rand::rng();
    (0..10)
        .map(|_| ALPHABET[rng.random_range(0..ALPHABET.len())] as char)
        .collect()
}

fn to_cents(value: f64) -> Result<i64, String> {
    if !value.is_finite() || value < 0.0 {
        return Err("money must be finite and non-negative".into());
    }
    Ok((value * 100.0).round() as i64)
}

fn scaled(value: f64, factor: f64) -> Result<i64, String> {
    if !value.is_finite() || value < 0.0 {
        return Err("number must be finite and non-negative".into());
    }
    Ok((value * factor).round() as i64)
}

fn optional_scaled(value: Option<f64>, factor: f64) -> Result<Option<i64>, String> {
    value.map(|number| scaled(number, factor)).transpose()
}

fn cents(value: i64) -> f64 {
    value as f64 / 100.0
}
fn error(error: rusqlite::Error) -> String {
    error.to_string()
}

fn parse_gcode_metadata(path: &Path) -> (Option<i64>, Option<i64>) {
    let Ok(content) = std::fs::read_to_string(path) else {
        return (None, None);
    };
    let mut minutes = None;
    let mut filament_milligrams = None;
    for line in content.lines().take(20_000) {
        let normalized = line.to_ascii_lowercase();
        if minutes.is_none()
            && (normalized.contains("estimated printing time")
                || normalized.contains("estimated_print_time"))
        {
            minutes = parse_duration_minutes(&normalized);
        }
        if filament_milligrams.is_none()
            && normalized.contains("filament used [g]")
            && let Some((_, value)) = normalized.split_once('=')
            && let Ok(grams) = value.trim().parse::<f64>()
        {
            filament_milligrams = scaled(grams, 1000.0).ok();
        }
        if minutes.is_some() && filament_milligrams.is_some() {
            break;
        }
    }
    (minutes, filament_milligrams)
}

fn parse_duration_minutes(value: &str) -> Option<i64> {
    let duration = value.split_once('=').map(|(_, tail)| tail).unwrap_or(value);
    let mut minutes = 0.0;
    for token in duration.split_whitespace() {
        let cleaned = token
            .trim_matches(|character: char| !character.is_ascii_alphanumeric() && character != '.');
        if let Some(hours) = cleaned
            .strip_suffix('h')
            .and_then(|number| number.parse::<f64>().ok())
        {
            minutes += hours * 60.0;
        } else if let Some(value) = cleaned
            .strip_suffix('m')
            .and_then(|number| number.parse::<f64>().ok())
        {
            minutes += value;
        } else if let Some(seconds) = cleaned
            .strip_suffix('s')
            .and_then(|number| number.parse::<f64>().ok())
        {
            minutes += seconds / 60.0;
        }
    }
    (minutes > 0.0).then(|| minutes.round() as i64)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn database() -> Database {
        Database::open(Path::new(":memory:")).expect("in-memory database")
    }

    #[test]
    fn seeds_local_workspace_and_persists_settings() {
        let database = database();

        assert_eq!(database.printers().expect("printers").len(), 3);
        assert_eq!(database.customers().expect("customers").len(), 1);
        assert_eq!(database.spools().expect("spools").len(), 1);

        let saved = database
            .save_settings(Settings {
                company_name: "Makerspace Test".into(),
                currency: "eur".into(),
                electricity_price_per_kwh: 3.123_456,
                machine_rate_per_hour: 42.25,
                labour_rate_per_hour: 80.50,
                default_markup_percent: 37.5,
                low_stock_threshold_grams: 150.0,
            })
            .expect("save settings");

        assert_eq!(saved.company_name, "Makerspace Test");
        assert_eq!(saved.currency, "EUR");
        assert_eq!(saved.electricity_price_per_kwh, 3.123_456);
        assert_eq!(saved.machine_rate_per_hour, 42.25);
        assert_eq!(saved.default_markup_percent, 37.5);
    }

    #[test]
    fn creates_and_tracks_order_lifecycle_in_dashboard() {
        let database = database();
        let customer_id = database.customers().expect("customers")[0].id.clone();

        let order = database
            .create_order(NewOrder {
                customer_id: Some(customer_id),
                title: "Functional enclosure".into(),
                deadline: Some("2026-09-01T12:00".into()),
                selling_price: 350.75,
                paid_amount: 100.25,
                notes: Some("PLA graphite".into()),
                model_ids: Vec::new(),
            })
            .expect("create order");

        assert_eq!(order.status, "NEW");
        assert_eq!(order.tracking_code.len(), 10);
        assert!(order.number.starts_with("ORD-"));
        assert_eq!(order.selling_price, 350.75);

        let dashboard = database.dashboard().expect("dashboard");
        assert_eq!(dashboard.active_orders, 1);
        assert_eq!(dashboard.revenue, 350.75);
        assert_eq!(dashboard.outstanding, 250.50);

        let completed = database
            .update_order_status(&order.id, "COMPLETED")
            .expect("complete order");
        assert_eq!(completed.status, "COMPLETED");
        assert_eq!(database.dashboard().expect("dashboard").active_orders, 0);
    }

    #[test]
    fn rejects_invalid_order_payment_and_status() {
        let database = database();
        let invalid_payment = database.create_order(NewOrder {
            customer_id: None,
            title: "Invalid payment".into(),
            deadline: None,
            selling_price: 10.0,
            paid_amount: 11.0,
            notes: None,
            model_ids: Vec::new(),
        });
        assert!(invalid_payment.is_err());

        let order = database
            .create_order(NewOrder {
                customer_id: None,
                title: "Valid order".into(),
                deadline: None,
                selling_price: 10.0,
                paid_amount: 0.0,
                notes: None,
                model_ids: Vec::new(),
            })
            .expect("create order");
        assert!(database.update_order_status(&order.id, "HACKED").is_err());
    }

    #[test]
    fn runs_complete_admin_workflow_with_model_history_and_filament_deduction() {
        let mut database = database();
        let customer_id = database.customers().expect("customers")[0].id.clone();
        let printer_id = database.printers().expect("printers")[0].id.clone();
        let spool = database.spools().expect("spools")[0].clone();
        let directory = std::env::temp_dir().join(format!("printforge-flow-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&directory).expect("temporary model directory");
        let source = directory.join("customer-part.gcode");
        std::fs::write(
            &source,
            "; estimated printing time = 1h 25m\n; filament used [g] = 12.5\nG28\n",
        )
        .expect("test gcode");

        let model = database
            .import_model(
                ImportModel {
                    source_path: source.to_string_lossy().into_owned(),
                    customer_id: Some(customer_id.clone()),
                    name: Some("Customer part".into()),
                },
                &directory.join("library"),
            )
            .expect("import model");
        assert_eq!(model.estimated_print_minutes, Some(85));
        assert_eq!(model.estimated_filament_grams, Some(12.5));

        let order = database
            .create_order(NewOrder {
                customer_id: Some(customer_id),
                title: "Customer part production".into(),
                deadline: Some("2026-09-05T12:00:00Z".into()),
                selling_price: 175.0,
                paid_amount: 75.0,
                notes: Some("Quality check required".into()),
                model_ids: vec![model.id.clone()],
            })
            .expect("create order with model");
        assert_eq!(order.models.len(), 1);
        assert_eq!(order.models[0].id, model.id);

        database
            .add_order_event(
                &order.id,
                NewOrderEvent {
                    title: "Model checked".into(),
                    message: Some("Geometry is printable".into()),
                },
            )
            .expect("add order history event");
        assert_eq!(database.order_events(&order.id).expect("events").len(), 2);

        let job = database
            .create_print_job(NewPrintJob {
                order_id: Some(order.id),
                printer_id: Some(printer_id),
                spool_id: Some(spool.id.clone()),
                model_id: Some(model.id),
                print_minutes: 85,
                filament_grams: 12.5,
                scheduled_start: Some("2026-09-01T10:00:00Z".into()),
                scheduled_end: Some("2026-09-01T11:25:00Z".into()),
                total_cost: 44.25,
                electricity_cost: Some(1.5),
                energy_kwh: Some(0.58),
                suggested_price: Some(61.95),
            })
            .expect("create print job");
        assert_eq!(job.model_name.as_deref(), Some("Customer part"));
        assert_eq!(
            database.start_print_job(&job.id).expect("start").status,
            "PRINTING"
        );
        assert_eq!(
            database
                .complete_print_job(&job.id)
                .expect("complete")
                .status,
            "COMPLETED"
        );
        let remaining = database
            .spools()
            .expect("spools")
            .into_iter()
            .find(|item| item.id == spool.id)
            .expect("used spool")
            .remaining_grams;
        assert!((remaining - (spool.remaining_grams - 12.5)).abs() < 0.001);
        std::fs::remove_dir_all(directory).expect("remove temporary model directory");
    }

    #[test]
    fn prevents_completing_job_when_spool_has_insufficient_filament() {
        let mut database = database();
        let spool = database.spools().expect("spools")[0].clone();
        let job = database
            .create_print_job(NewPrintJob {
                order_id: None,
                printer_id: None,
                spool_id: Some(spool.id.clone()),
                model_id: None,
                print_minutes: 60,
                filament_grams: spool.remaining_grams + 1.0,
                scheduled_start: None,
                scheduled_end: None,
                total_cost: 10.0,
                electricity_cost: Some(1.0),
                energy_kwh: Some(0.2),
                suggested_price: Some(14.0),
            })
            .expect("create oversized job");
        database.start_print_job(&job.id).expect("start job");
        assert!(database.complete_print_job(&job.id).is_err());
        assert_eq!(
            database
                .spools()
                .expect("spools")
                .into_iter()
                .find(|item| item.id == spool.id)
                .expect("spool")
                .remaining_grams,
            spool.remaining_grams
        );
    }
}
