# PrintForge user guide

[Русский](USER_GUIDE_RU.md) · [English](USER_GUIDE_EN.md)

This guide describes the everyday workshop workflow, from the first login to sharing a PDF receipt and tracking code with a customer.

## 1. Login and dashboard

Open `http://localhost` and sign in as an administrator. The dashboard shows active orders, the print queue, active printers, remaining filament, revenue, cost, electricity, and profit.

![Workshop dashboard](images/dashboard.jpg)

Check the current electricity tariff in the lower-left corner before starting. It is included in every new print calculation.

## 2. Workshop settings

Open **Settings**:

1. enter the workshop name shown to customers;
2. choose the currency;
3. set the price per kWh;
4. enter the public URL for customer links and QR codes;
5. save the changes.

![Settings and Telegram](images/settings.jpg)

Tariff changes affect new jobs. Existing jobs keep their historical tariff.

## 3. Add a printer

Open **Printers** and select **Add from catalog**.

![Printer fleet](images/printers.jpg)

The catalog contains 387 profiles and supports manufacturer/model search.

![Select a printer profile](images/printer-catalog.jpg)

1. Find the printer.
2. Select its card to fill the image and build volume.
3. Enter average printing power in watts.
4. Enter purchase price and depreciation parameters.
5. Save.

Use realistic printing consumption, not only the power-supply maximum. A plug-in power meter gives the best result.

## 4. Add a spool

Open **Inventory** and create a spool with its internal code, manufacturer, product, material, color, initial/current weight, purchase price, and supplier.

PrintForge calculates the price per gram. Completing a successful job deducts actual material and creates an inventory movement.

## 5. Create a customer

Open **Customers**, add a record, and enter name, company, phone, and email. One customer can own multiple models and orders.

## 6. Customer model library

Open **3D models** and upload STL, OBJ, or 3MF.

![Model library](images/models.jpg)

1. Select the owner.
2. Enter a clear model name.
3. Select the source file.
4. Optionally add an image.
5. Save.

For 3MF, the system attempts to extract the embedded preview. Upload a separate image for STL and OBJ. The source remains downloadable by the administrator and through authorized linked orders.

## 7. Create an order

Open **Orders** and select **New order**.

![Order list](images/orders.jpg)

1. Select a customer.
2. Attach one or more models from that customer's library.
3. Enter quantity, deadline, note, total price, and amount paid.
4. Create the order.

PrintForge creates an internal number such as `ORD-2026-00001`, a random ten-character code without ambiguous `0/O` and `1/I`, and a public `/track/CODE` URL.

Share the tracking code, link, or PDF. The public page accepts the tracking code, not the internal order number.

## 8. Calculate a print job

Open **Print queue** and select **Calculate job**.

![Job cost calculation](images/cost-calculator.jpg)

Enter the order and model, printer, spool, print minutes, filament weight, operator time, post-processing, packaging, other expenses, and margin. Review material, electricity, machine, depreciation, total cost, and suggested price before adding the job to the queue.

## 9. Production and actual values

Update the job status during production. On completion, enter actual print minutes, grams, and kWh when a meter reading is available.

If kWh is empty, PrintForge derives it from power and time. A `SUCCESS` update transactionally records actuals, deducts filament, records inventory movement, increments printer runtime, and recalculates cost.

## 10. Customer view

The customer opens a link or sends the code to Telegram. The public page shows the current stage, completion percentage, models, cost, payment, and balance.

![Customer tracking](images/tracking.jpg)

Customers can download allowed source files and the PDF receipt, but cannot access the admin panel or other orders.

## 11. PDF receipt

Select **Download PDF** in the order card. Customers have the same action on the public page.

![PDF example](images/pdf-receipt.png)

The document contains workshop branding, order number and code, QR link, customer, date, deadline, status, models, total, paid amount, and balance. It confirms the estimate and payment state; it is not a fiscal cash-register receipt.

## 12. Telegram bot

After the bot is configured, a customer:

1. opens the workshop bot;
2. sends the tracking code;
3. receives status, progress, and payment information;
4. can receive a photo and model file.

Every order has its own code. A customer can check multiple orders by sending different codes.

## 13. Daily administrator routine

At the start of the day:

- review the queue, available printers, and low spools;
- verify the electricity tariff;
- review unfinished orders and deadlines.

After every print:

- record actual weight and time;
- update the order status;
- verify the spool balance;
- send an updated receipt when needed.

At the end of the day, review revenue, cost, and profit. Create a backup before major changes or upgrades.

## 14. Troubleshooting

```bash
docker compose ps
docker compose logs --tail=200
curl -fsS http://localhost/health
```

Port configuration, recovery, updates, and detailed troubleshooting are documented in [SETUP_EN.md](SETUP_EN.md).
