import { invoke } from "@tauri-apps/api/core";

export type Dashboard = { activeOrders:number; queuedJobs:number; availablePrinters:number; lowStockSpools:number; revenue:number; outstanding:number; currency:string };
export type Customer = { id:string; name:string; company?:string; phone?:string; email?:string };
export type Printer = { id:string; name:string; manufacturer:string; model:string; status:string; powerWatts:number; catalogKey?:string; buildXMm?:number; buildYMm?:number; buildZMm?:number; purchasePrice:number };
export type Spool = { id:string; code:string; material:string; colorName:string; colorHex:string; remainingGrams:number; pricePerGram:number };
export type Order = { id:string; number:string; trackingCode:string; customerId?:string; customerName:string; title:string; status:string; deadline?:string; sellingPrice:number; paidAmount:number; createdAt:string };
export type Settings = { companyName:string; currency:string; electricityPricePerKwh:number; machineRatePerHour:number; labourRatePerHour:number; defaultMarkupPercent:number; lowStockThresholdGrams:number };
export type CostInput = { printMinutes:number; filamentGrams:number; filamentPricePerGram:number; powerWatts:number; electricityPricePerKwh:number; machineRatePerHour:number; depreciationPerHour:number; operatorHours:number; labourRatePerHour:number; postProcessingCost:number; packagingCost:number; otherCost:number; markupPercent:number };
export type CostBreakdown = { materialCost:number; energyKwh:number; electricityCost:number; machineCost:number; depreciationCost:number; labourCost:number; postProcessingCost:number; packagingCost:number; otherCost:number; totalCost:number; markupAmount:number; suggestedPrice:number };
export type NewOrder = { customerId?:string; title:string; deadline?:string; sellingPrice:number; paidAmount:number; notes?:string };
export type NewCustomer = { name:string; company?:string; phone?:string; email?:string };
export type NewSpool = { code:string; manufacturer:string; productName:string; material:string; colorName:string; colorHex:string; initialGrams:number; purchasePrice:number; supplier?:string };
export type NewPrinter = { catalogKey?:string; name:string; manufacturer:string; model:string; powerWatts:number; buildXMm?:number; buildYMm?:number; buildZMm?:number; purchasePrice:number };
export type PrinterCatalogModel = { key:string; manufacturer:string; model:string; fullName:string; technology:string; nozzleDiameters:number[]; buildXMm?:number; buildYMm?:number; buildZMm?:number; imageUrl?:string; defaultMaterials:string[]; profileUrl:string };
export type ModelAsset = { id:string; customerId?:string; customerName:string; name:string; originalFilename:string; format:string; fileSizeBytes:number; estimatedPrintMinutes?:number; estimatedFilamentGrams?:number; createdAt:string };
export type PrintJob = { id:string; orderId?:string; orderNumber?:string; printerId?:string; printerName?:string; spoolId?:string; spoolCode?:string; status:string; printMinutes:number; filamentGrams:number; scheduledStart?:string; scheduledEnd?:string; totalCost:number; createdAt:string };
export type NewPrintJob = { orderId?:string; printerId?:string; spoolId?:string; printMinutes:number; filamentGrams:number; scheduledStart?:string; scheduledEnd?:string; totalCost:number };
export type ReceiptResult = { path:string; filename:string };
export type AppInfo = { dataDirectory:string; databasePath:string; catalogModels:number; appVersion:string };

export const desktop = {
  dashboard: () => invoke<Dashboard>("get_dashboard"),
  orders: () => invoke<Order[]>("list_orders"),
  customers: () => invoke<Customer[]>("list_customers"),
  createCustomer: (input:NewCustomer) => invoke<Customer>("create_customer", { input }),
  printers: () => invoke<Printer[]>("list_printers"),
  createPrinter: (input:NewPrinter) => invoke<Printer>("create_printer", { input }),
  printerCatalog: () => invoke<PrinterCatalogModel[]>("list_printer_catalog"),
  spools: () => invoke<Spool[]>("list_spools"),
  createSpool: (input:NewSpool) => invoke<Spool>("create_spool", { input }),
  models: () => invoke<ModelAsset[]>("list_models"),
  importModel: (input:{sourcePath:string;customerId?:string;name?:string}) => invoke<ModelAsset>("import_model", { input }),
  revealModel: (id:string) => invoke<void>("reveal_model", { id }),
  printJobs: () => invoke<PrintJob[]>("list_print_jobs"),
  createPrintJob: (input:NewPrintJob) => invoke<PrintJob>("create_print_job", { input }),
  completePrintJob: (id:string) => invoke<PrintJob>("complete_print_job", { id }),
  settings: () => invoke<Settings>("get_settings"),
  appInfo: () => invoke<AppInfo>("app_info"),
  openDataDirectory: () => invoke<void>("open_data_directory"),
  calculateCost: (input:CostInput) => invoke<CostBreakdown>("calculate_cost", { input }),
  createOrder: (input:NewOrder) => invoke<Order>("create_order", { input }),
  updateOrderStatus: (id:string,status:string) => invoke<Order>("update_order_status", { id, status }),
  saveSettings: (settings:Settings) => invoke<Settings>("save_settings", { settings }),
  generateReceipt: (orderId:string) => invoke<ReceiptResult>("generate_receipt", { orderId }),
};
