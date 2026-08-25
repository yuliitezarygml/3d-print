mod catalog;
mod commands;
mod cost;
mod db;
mod models;
mod receipt;

use std::fs;
use std::sync::Mutex;

use commands::AppState;
use db::Database;
use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            let data_directory = app.path().app_data_dir()?;
            fs::create_dir_all(&data_directory)?;
            let database_path = data_directory.join("printforge.sqlite");
            let database = Database::open(&database_path).map_err(std::io::Error::other)?;
            app.manage(AppState {
                database: Mutex::new(database),
                data_directory,
                database_path,
            });
            Ok(())
        })
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![
            commands::get_dashboard,
            commands::list_orders,
            commands::create_order,
            commands::update_order_status,
            commands::list_customers,
            commands::create_customer,
            commands::list_printers,
            commands::create_printer,
            commands::list_printer_catalog,
            commands::list_spools,
            commands::create_spool,
            commands::list_models,
            commands::import_model,
            commands::reveal_model,
            commands::list_print_jobs,
            commands::create_print_job,
            commands::complete_print_job,
            commands::calculate_cost,
            commands::get_settings,
            commands::save_settings,
            commands::app_info,
            commands::open_data_directory,
            commands::generate_receipt,
        ])
        .run(tauri::generate_context!())
        .expect("error while running PrintForge Desktop");
}
