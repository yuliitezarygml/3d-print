use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Dashboard {
    pub active_orders: i64,
    pub queued_jobs: i64,
    pub available_printers: i64,
    pub low_stock_spools: i64,
    pub revenue: f64,
    pub outstanding: f64,
    pub currency: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Customer {
    pub id: String,
    pub name: String,
    pub company: Option<String>,
    pub phone: Option<String>,
    pub email: Option<String>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Printer {
    pub id: String,
    pub name: String,
    pub manufacturer: String,
    pub model: String,
    pub status: String,
    pub power_watts: f64,
    pub catalog_key: Option<String>,
    pub build_x_mm: Option<f64>,
    pub build_y_mm: Option<f64>,
    pub build_z_mm: Option<f64>,
    pub purchase_price: f64,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct NewPrinter {
    pub catalog_key: Option<String>,
    pub name: String,
    pub manufacturer: String,
    pub model: String,
    pub power_watts: f64,
    pub build_x_mm: Option<f64>,
    pub build_y_mm: Option<f64>,
    pub build_z_mm: Option<f64>,
    pub purchase_price: f64,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Spool {
    pub id: String,
    pub code: String,
    pub material: String,
    pub color_name: String,
    pub color_hex: String,
    pub remaining_grams: f64,
    pub price_per_gram: f64,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct NewSpool {
    pub code: String,
    pub manufacturer: String,
    pub product_name: String,
    pub material: String,
    pub color_name: String,
    pub color_hex: String,
    pub initial_grams: f64,
    pub purchase_price: f64,
    pub supplier: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct NewCustomer {
    pub name: String,
    pub company: Option<String>,
    pub phone: Option<String>,
    pub email: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PrinterCatalogModel {
    pub key: String,
    pub manufacturer: String,
    pub model: String,
    pub full_name: String,
    pub technology: String,
    pub nozzle_diameters: Vec<f64>,
    pub build_x_mm: Option<f64>,
    pub build_y_mm: Option<f64>,
    pub build_z_mm: Option<f64>,
    pub image_url: Option<String>,
    pub default_materials: Vec<String>,
    pub profile_url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ModelAsset {
    pub id: String,
    pub customer_id: Option<String>,
    pub customer_name: String,
    pub name: String,
    pub original_filename: String,
    pub format: String,
    pub file_size_bytes: i64,
    pub estimated_print_minutes: Option<i64>,
    pub estimated_filament_grams: Option<f64>,
    pub created_at: String,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ImportModel {
    pub source_path: String,
    pub customer_id: Option<String>,
    pub name: Option<String>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PrintJob {
    pub id: String,
    pub order_id: Option<String>,
    pub order_number: Option<String>,
    pub printer_id: Option<String>,
    pub printer_name: Option<String>,
    pub spool_id: Option<String>,
    pub spool_code: Option<String>,
    pub status: String,
    pub print_minutes: i64,
    pub filament_grams: f64,
    pub scheduled_start: Option<String>,
    pub scheduled_end: Option<String>,
    pub total_cost: f64,
    pub created_at: String,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct NewPrintJob {
    pub order_id: Option<String>,
    pub printer_id: Option<String>,
    pub spool_id: Option<String>,
    pub print_minutes: i64,
    pub filament_grams: f64,
    pub scheduled_start: Option<String>,
    pub scheduled_end: Option<String>,
    pub total_cost: f64,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReceiptResult {
    pub path: String,
    pub filename: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AppInfo {
    pub data_directory: String,
    pub database_path: String,
    pub catalog_models: usize,
    pub app_version: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Order {
    pub id: String,
    pub number: String,
    pub tracking_code: String,
    pub customer_id: Option<String>,
    pub customer_name: String,
    pub title: String,
    pub status: String,
    pub deadline: Option<String>,
    pub selling_price: f64,
    pub paid_amount: f64,
    pub created_at: String,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct NewOrder {
    pub customer_id: Option<String>,
    pub title: String,
    pub deadline: Option<String>,
    pub selling_price: f64,
    pub paid_amount: f64,
    pub notes: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Settings {
    pub company_name: String,
    pub currency: String,
    pub electricity_price_per_kwh: f64,
    pub machine_rate_per_hour: f64,
    pub labour_rate_per_hour: f64,
    pub default_markup_percent: f64,
    pub low_stock_threshold_grams: f64,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CostInput {
    pub print_minutes: u32,
    pub filament_grams: f64,
    pub filament_price_per_gram: f64,
    pub power_watts: f64,
    pub electricity_price_per_kwh: f64,
    pub machine_rate_per_hour: f64,
    pub depreciation_per_hour: f64,
    pub operator_hours: f64,
    pub labour_rate_per_hour: f64,
    pub post_processing_cost: f64,
    pub packaging_cost: f64,
    pub other_cost: f64,
    pub markup_percent: f64,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CostBreakdown {
    pub material_cost: f64,
    pub energy_kwh: f64,
    pub electricity_cost: f64,
    pub machine_cost: f64,
    pub depreciation_cost: f64,
    pub labour_cost: f64,
    pub post_processing_cost: f64,
    pub packaging_cost: f64,
    pub other_cost: f64,
    pub total_cost: f64,
    pub markup_amount: f64,
    pub suggested_price: f64,
}
