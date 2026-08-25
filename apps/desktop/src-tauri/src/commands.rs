use std::path::PathBuf;
use std::process::Command;
use std::sync::Mutex;

use tauri::State;

use crate::catalog;
use crate::cost;
use crate::db::Database;
use crate::models::{
    AppInfo, CostBreakdown, CostInput, Customer, Dashboard, ImportModel, ModelAsset, NewCustomer,
    NewOrder, NewPrintJob, NewPrinter, NewSpool, Order, PrintJob, Printer, PrinterCatalogModel,
    ReceiptResult, Settings, Spool,
};
use crate::receipt;

pub struct AppState {
    pub database: Mutex<Database>,
    pub data_directory: PathBuf,
    pub database_path: PathBuf,
}

fn with_database<T>(
    state: State<'_, AppState>,
    operation: impl FnOnce(&mut Database) -> Result<T, String>,
) -> Result<T, String> {
    let mut database = state
        .database
        .lock()
        .map_err(|_| "database lock is poisoned".to_string())?;
    operation(&mut database)
}

#[tauri::command]
pub fn get_dashboard(state: State<'_, AppState>) -> Result<Dashboard, String> {
    with_database(state, |database| database.dashboard())
}

#[tauri::command]
pub fn list_orders(state: State<'_, AppState>) -> Result<Vec<Order>, String> {
    with_database(state, |database| database.orders())
}

#[tauri::command]
pub fn create_order(state: State<'_, AppState>, input: NewOrder) -> Result<Order, String> {
    with_database(state, |database| database.create_order(input))
}

#[tauri::command]
pub fn update_order_status(
    state: State<'_, AppState>,
    id: String,
    status: String,
) -> Result<Order, String> {
    with_database(state, |database| database.update_order_status(&id, &status))
}

#[tauri::command]
pub fn list_customers(state: State<'_, AppState>) -> Result<Vec<Customer>, String> {
    with_database(state, |database| database.customers())
}

#[tauri::command]
pub fn create_customer(state: State<'_, AppState>, input: NewCustomer) -> Result<Customer, String> {
    with_database(state, |database| database.create_customer(input))
}

#[tauri::command]
pub fn list_printers(state: State<'_, AppState>) -> Result<Vec<Printer>, String> {
    with_database(state, |database| database.printers())
}

#[tauri::command]
pub fn create_printer(state: State<'_, AppState>, input: NewPrinter) -> Result<Printer, String> {
    with_database(state, |database| database.create_printer(input))
}

#[tauri::command]
pub fn list_printer_catalog() -> Result<Vec<PrinterCatalogModel>, String> {
    catalog::models()
}

#[tauri::command]
pub fn list_spools(state: State<'_, AppState>) -> Result<Vec<Spool>, String> {
    with_database(state, |database| database.spools())
}

#[tauri::command]
pub fn create_spool(state: State<'_, AppState>, input: NewSpool) -> Result<Spool, String> {
    with_database(state, |database| database.create_spool(input))
}

#[tauri::command]
pub fn list_models(state: State<'_, AppState>) -> Result<Vec<ModelAsset>, String> {
    with_database(state, |database| database.models())
}

#[tauri::command]
pub fn import_model(state: State<'_, AppState>, input: ImportModel) -> Result<ModelAsset, String> {
    let models_directory = state.data_directory.join("models");
    with_database(state, |database| {
        database.import_model(input, &models_directory)
    })
}

#[tauri::command]
pub fn reveal_model(state: State<'_, AppState>, id: String) -> Result<(), String> {
    let path = with_database(state, |database| database.model_storage_path(&id))?;
    reveal_path(&path)
}

#[tauri::command]
pub fn list_print_jobs(state: State<'_, AppState>) -> Result<Vec<PrintJob>, String> {
    with_database(state, |database| database.print_jobs())
}

#[tauri::command]
pub fn create_print_job(
    state: State<'_, AppState>,
    input: NewPrintJob,
) -> Result<PrintJob, String> {
    with_database(state, |database| database.create_print_job(input))
}

#[tauri::command]
pub fn complete_print_job(state: State<'_, AppState>, id: String) -> Result<PrintJob, String> {
    with_database(state, |database| database.complete_print_job(&id))
}

#[tauri::command]
pub fn calculate_cost(input: CostInput) -> Result<CostBreakdown, String> {
    cost::calculate(input)
}

#[tauri::command]
pub fn get_settings(state: State<'_, AppState>) -> Result<Settings, String> {
    with_database(state, |database| database.settings())
}

#[tauri::command]
pub fn save_settings(state: State<'_, AppState>, settings: Settings) -> Result<Settings, String> {
    with_database(state, |database| database.save_settings(settings))
}

#[tauri::command]
pub fn app_info(state: State<'_, AppState>) -> AppInfo {
    AppInfo {
        data_directory: state.data_directory.to_string_lossy().into_owned(),
        database_path: state.database_path.to_string_lossy().into_owned(),
        catalog_models: catalog::count(),
        app_version: env!("CARGO_PKG_VERSION").into(),
    }
}

#[tauri::command]
pub fn open_data_directory(state: State<'_, AppState>) -> Result<(), String> {
    reveal_path(&state.data_directory.to_string_lossy())
}

#[tauri::command]
pub fn generate_receipt(
    state: State<'_, AppState>,
    order_id: String,
) -> Result<ReceiptResult, String> {
    let (order, settings) = with_database(state.clone(), |database| {
        Ok((database.receipt_order(&order_id)?, database.settings()?))
    })?;
    let result = receipt::generate(&order, &settings, &state.data_directory.join("receipts"))?;
    reveal_path(&result.path)?;
    Ok(result)
}

fn reveal_path(path: &str) -> Result<(), String> {
    #[cfg(target_os = "macos")]
    let status = Command::new("open").args(["-R", path]).status();
    #[cfg(target_os = "windows")]
    let status = Command::new("explorer")
        .arg(format!("/select,{path}"))
        .status();
    #[cfg(target_os = "linux")]
    let status = Command::new("xdg-open")
        .arg(
            PathBuf::from(path)
                .parent()
                .unwrap_or_else(|| std::path::Path::new(path)),
        )
        .status();
    status
        .map_err(|error| error.to_string())
        .and_then(|result| {
            if result.success() {
                Ok(())
            } else {
                Err("could not open file location".into())
            }
        })
}
