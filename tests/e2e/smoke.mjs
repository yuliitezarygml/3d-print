import assert from "node:assert/strict";

const baseUrl = process.env.PRINTFORGE_URL ?? "http://localhost";
const stamp = Date.now();

async function request(path, options = {}, expected = 200) {
  const response = await fetch(`${baseUrl}${path}`, options);
  const text = await response.text();
  let body;
  try { body = text ? JSON.parse(text) : undefined; } catch { body = text; }
  assert.equal(response.status, expected, `${options.method ?? "GET"} ${path}: ${text}`);
  return body;
}

console.log(`Testing PrintForge at ${baseUrl}`);

await request("/health");
const frontend = await fetch(`${baseUrl}/login`);
assert.equal(frontend.status, 200);
assert.match(await frontend.text(), /PrintForge/);

await request("/api/auth/login", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ email: "admin@printforge.local", password: "wrong-password" }),
}, 401);

const login = await request("/api/auth/login", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ email: "admin@printforge.local", password: "admin12345" }),
});
assert.equal(login.user.role, "ADMIN");
const headers = { authorization: `Bearer ${login.accessToken}`, "content-type": "application/json" };

const settings = await request("/api/settings", { headers });
assert.ok(settings.electricityPricePerKwh > 0);

const printers = await request("/api/printers", { headers });
assert.ok(printers.length >= 1);
assert.ok(printers.some((item) => item.imageUrl));
const printer = printers.find((item) => item.status !== "MAINTENANCE") ?? printers[0];
const catalog = await request("/api/printer-catalog", { headers });
assert.ok(catalog.total >= 300);
assert.equal(catalog.models.filter((item) => item.imageUrl).length, catalog.total);

const customer = await request("/api/customers", {
  method: "POST", headers,
  body: JSON.stringify({ name: `E2E Customer ${stamp}`, email: `e2e-${stamp}@example.test` }),
}, 201);
assert.match(customer.name, /E2E Customer/);

const spool = await request("/api/spools", {
  method: "POST", headers,
  body: JSON.stringify({ code: `E2E-${stamp}`, manufacturer: "Test Lab", productName: "QA PLA", material: "PLA", colorName: "Signal Violet", colorHex: "#7557FF", initialWeightGrams: 100, purchasePrice: 50, supplier: "QA" }),
}, 201);
assert.equal(spool.remainingWeightGrams, 100);
assert.equal(spool.pricePerGram, 0.5);

const stl = `solid test\nfacet normal 0 0 1\n outer loop\n  vertex 0 0 0\n  vertex 10 0 0\n  vertex 0 10 0\n endloop\nendfacet\nendsolid test\n`;
const uploadBody = new FormData();
uploadBody.set("name", `E2E triangle ${stamp}`);
uploadBody.set("customerId", customer.id);
uploadBody.set("file", new Blob([stl], { type: "model/stl" }), `e2e-${stamp}.stl`);
const model = await request("/api/models/upload", {
  method: "POST",
  headers: { authorization: `Bearer ${login.accessToken}` },
  body: uploadBody,
}, 201);
assert.equal(model.format, "STL");
assert.equal(model.triangleCount, 1);
const modelFile = await fetch(`${baseUrl}/api/models/${model.id}/file`, { headers });
assert.equal(modelFile.status, 200);

const order = await request("/api/orders", {
  method: "POST", headers,
  body: JSON.stringify({ customerId: customer.id, modelIds: [model.id], sellingPrice: 250, paidAmount: 100, notes: "Automated professional user journey" }),
}, 201);
assert.match(order.number, /^ORD-\d{4}-\d{5}$/);
assert.match(order.trackingCode, /^[23456789A-HJ-NP-Z]{10}$/);
const publicOrder = await request(`/api/public/track/${order.trackingCode}`);
assert.equal(publicOrder.models.length, 1);
assert.equal(publicOrder.models[0].id, model.id);
const publicReceipt = await fetch(`${baseUrl}/api/public/track/${order.trackingCode}/receipt.pdf`);
assert.equal(publicReceipt.status, 200);
assert.equal(publicReceipt.headers.get("content-type"), "application/pdf");
assert.match(publicReceipt.headers.get("content-disposition") ?? "", new RegExp(`receipt-${order.number}\\.pdf`));
const publicReceiptBytes = new Uint8Array(await publicReceipt.arrayBuffer());
assert.ok(publicReceiptBytes.length > 10_000);
assert.equal(new TextDecoder().decode(publicReceiptBytes.slice(0, 5)), "%PDF-");
const privateReceipt = await fetch(`${baseUrl}/api/orders/${order.id}/receipt.pdf`, { headers });
assert.equal(privateReceipt.status, 200);
assert.equal(privateReceipt.headers.get("content-type"), "application/pdf");

const job = await request("/api/print-jobs", {
  method: "POST", headers,
  body: JSON.stringify({ orderId: order.id, modelId: model.id, printerId: printer.id, spoolId: spool.id, quantity: 1, estimatedMinutes: 60, estimatedFilamentGrams: 10, labourHours: 0.25, postProcessingCost: 3, packagingCost: 5, otherCost: 2, markupPercent: 40 }),
}, 201);
const expectedEstimatedEnergy = printer.powerWatts / 1000;
assert.ok(Math.abs(job.costs.energyKwh - expectedEstimatedEnergy) < 0.00001);
assert.ok(Math.abs(job.costs.electricityCost - expectedEstimatedEnergy * settings.electricityPricePerKwh) < 0.00001);

await request(`/api/print-jobs/${job.id}/status`, {
  method: "PATCH", headers, body: JSON.stringify({ status: "PRINTING" }),
});
const printingOrder = await request(`/api/public/track/${order.trackingCode}`);
assert.equal(printingOrder.status, "PRINTING");

const actualEnergy = 0.222;
const completed = await request(`/api/print-jobs/${job.id}/status`, {
  method: "PATCH", headers,
  body: JSON.stringify({ status: "SUCCESS", actualMinutes: 65, actualFilamentGrams: 11, actualEnergyKwh: actualEnergy }),
});
assert.equal(completed.status, "SUCCESS");
assert.equal(completed.remainingSpoolGrams, 89);
assert.ok(Math.abs(completed.costs.electricityCost - actualEnergy * settings.electricityPricePerKwh) < 0.00001);
const trackedCompleted = await request(`/api/public/track/${order.trackingCode}`);
assert.equal(trackedCompleted.status, "POST_PROCESSING");
const publicModelFile = await fetch(`${baseUrl}${trackedCompleted.models[0].downloadUrl}`);
assert.equal(publicModelFile.status, 200);

const spoolsAfter = await request("/api/spools", { headers });
assert.equal(spoolsAfter.find((item) => item.id === spool.id).remainingWeightGrams, 89);
const transactions = await request("/api/inventory/transactions", { headers });
assert.ok(transactions.some((item) => item.spoolCode === spool.code && item.type === "PRINT_USAGE" && item.quantityGrams === -11));

const dashboard = await request("/api/dashboard", { headers });
assert.ok(dashboard.spoolCount >= 1);

console.log(JSON.stringify({
  result: "PASS",
  order: order.number,
  trackingCode: order.trackingCode,
  publicStatus: trackedCompleted.status,
  receiptPdfBytes: publicReceiptBytes.length,
  printerCatalogModels: catalog.total,
  model: model.originalFilename,
  job: job.id,
  estimatedEnergyKwh: job.costs.energyKwh,
  actualEnergyKwh: actualEnergy,
  electricityCost: completed.costs.electricityCost,
  filamentWrittenOffGrams: 11,
  remainingSpoolGrams: completed.remainingSpoolGrams,
}, null, 2));
