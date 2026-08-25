use std::fs;
use std::path::Path;

use chrono::{DateTime, Local};
use genpdf::elements;
use genpdf::fonts::{FontData, FontFamily};
use genpdf::style::{Color, Style};
use genpdf::{Alignment, Element};

use crate::models::{Order, ReceiptResult, Settings};

const REGULAR_FONT: &[u8] =
    include_bytes!("../../../backend/internal/http/assets/NotoSans-Regular.ttf");
const BOLD_FONT: &[u8] = include_bytes!("../../../backend/internal/http/assets/NotoSans-Bold.ttf");

pub fn generate(
    order: &Order,
    settings: &Settings,
    output_directory: &Path,
) -> Result<ReceiptResult, String> {
    fs::create_dir_all(output_directory).map_err(|error| error.to_string())?;
    let filename = format!("receipt-{}.pdf", order.number);
    let path = output_directory.join(&filename);

    let regular = FontData::new(REGULAR_FONT.to_vec(), None).map_err(pdf_error)?;
    let bold = FontData::new(BOLD_FONT.to_vec(), None).map_err(pdf_error)?;
    let family = FontFamily {
        regular: regular.clone(),
        bold: bold.clone(),
        italic: regular,
        bold_italic: bold,
    };
    let mut document = genpdf::Document::new(family);
    document.set_title(format!("Квитанция {}", order.number));
    document.set_minimal_conformance();
    document.set_line_spacing(1.15);

    let mut decorator = genpdf::SimplePageDecorator::new();
    decorator.set_margins(16);
    decorator.set_header(|page| {
        let text = if page == 1 {
            String::new()
        } else {
            format!("PrintForge · страница {page}")
        };
        elements::Paragraph::new(text)
            .aligned(Alignment::Right)
            .styled(
                Style::new()
                    .with_font_size(8)
                    .with_color(Color::Rgb(110, 118, 113)),
            )
    });
    document.set_page_decorator(decorator);

    let purple = Color::Rgb(120, 91, 242);
    let muted = Color::Rgb(98, 108, 102);
    let green = Color::Rgb(20, 137, 78);

    document.push(
        elements::Paragraph::new(&settings.company_name)
            .styled(Style::new().bold().with_font_size(12).with_color(purple)),
    );
    document.push(elements::Break::new(0.5));
    document.push(
        elements::Paragraph::new("КВИТАНЦИЯ К ЗАКАЗУ")
            .styled(Style::new().bold().with_font_size(25)),
    );
    document.push(
        elements::Paragraph::new(format!("{}  ·  код {}", order.number, order.tracking_code))
            .styled(Style::new().with_font_size(10).with_color(muted)),
    );
    document.push(elements::Break::new(1.2));

    let mut details = elements::TableLayout::new(vec![1, 2]);
    details.set_cell_decorator(elements::FrameCellDecorator::new(true, true, false));
    for (label, value) in [
        ("Клиент", order.customer_name.clone()),
        ("Заказ", order.title.clone()),
        ("Статус", status_label(&order.status).to_string()),
        ("Создан", local_datetime(&order.created_at)),
        (
            "Плановый срок",
            order
                .deadline
                .as_deref()
                .map(local_datetime)
                .unwrap_or_else(|| "Не указан".into()),
        ),
    ] {
        details
            .row()
            .element(
                elements::Paragraph::new(label)
                    .styled(Style::new().bold().with_color(muted))
                    .padded(2),
            )
            .element(elements::Paragraph::new(value).padded(2))
            .push()
            .map_err(pdf_error)?;
    }
    document.push(details);
    document.push(elements::Break::new(1.3));

    document.push(
        elements::Paragraph::new("СОСТАВ ЗАКАЗА")
            .styled(Style::new().bold().with_font_size(10).with_color(purple)),
    );
    document.push(elements::Break::new(0.5));
    let mut items = elements::TableLayout::new(vec![5, 1, 2]);
    items.set_cell_decorator(elements::FrameCellDecorator::new(true, true, false));
    items
        .row()
        .element(
            elements::Paragraph::new("Наименование")
                .styled(Style::new().bold())
                .padded(2),
        )
        .element(
            elements::Paragraph::new("Кол.")
                .aligned(Alignment::Center)
                .styled(Style::new().bold())
                .padded(2),
        )
        .element(
            elements::Paragraph::new("Сумма")
                .aligned(Alignment::Right)
                .styled(Style::new().bold())
                .padded(2),
        )
        .push()
        .map_err(pdf_error)?;
    items
        .row()
        .element(elements::Paragraph::new(&order.title).padded(2))
        .element(
            elements::Paragraph::new("1")
                .aligned(Alignment::Center)
                .padded(2),
        )
        .element(
            elements::Paragraph::new(money(order.selling_price, &settings.currency))
                .aligned(Alignment::Right)
                .padded(2),
        )
        .push()
        .map_err(pdf_error)?;
    document.push(items);
    document.push(elements::Break::new(1.5));

    let balance = (order.selling_price - order.paid_amount).max(0.0);
    let mut totals = elements::TableLayout::new(vec![3, 2]);
    for (label, value, bold_row, color) in [
        ("Стоимость заказа", order.selling_price, false, muted),
        ("Оплачено", order.paid_amount, false, green),
        ("Остаток к оплате", balance, true, purple),
    ] {
        let row_style = if bold_row {
            Style::new().bold().with_font_size(13)
        } else {
            Style::new().with_font_size(10)
        };
        totals
            .row()
            .element(
                elements::Paragraph::new(label)
                    .styled(row_style.with_color(color))
                    .padded(2),
            )
            .element(
                elements::Paragraph::new(money(value, &settings.currency))
                    .aligned(Alignment::Right)
                    .styled(row_style.with_color(color))
                    .padded(2),
            )
            .push()
            .map_err(pdf_error)?;
    }
    document.push(totals.framed().padded(3));
    document.push(elements::Break::new(2));
    document.push(
        elements::Paragraph::new("Спасибо за заказ! Сохраните код для связи с мастерской.")
            .aligned(Alignment::Center)
            .styled(Style::new().with_font_size(9).with_color(muted)),
    );
    document.push(
        elements::Paragraph::new(
            "Документ является расчётной квитанцией и не является фискальным кассовым чеком.",
        )
        .aligned(Alignment::Center)
        .styled(Style::new().with_font_size(7).with_color(muted)),
    );

    document.render_to_file(&path).map_err(pdf_error)?;
    Ok(ReceiptResult {
        path: path.to_string_lossy().into_owned(),
        filename,
    })
}

fn money(value: f64, currency: &str) -> String {
    format!("{value:.2} {currency}")
}

fn local_datetime(value: &str) -> String {
    DateTime::parse_from_rfc3339(value)
        .map(|date| {
            date.with_timezone(&Local)
                .format("%d.%m.%Y %H:%M")
                .to_string()
        })
        .unwrap_or_else(|_| value.replace('T', " "))
}

fn status_label(status: &str) -> &str {
    match status {
        "NEW" => "Новый",
        "CONFIRMED" => "Подтверждён",
        "WAITING" => "Ожидает материал",
        "READY_TO_PRINT" => "Готов к печати",
        "PRINTING" => "Печатается",
        "POST_PROCESSING" => "Постобработка",
        "READY" => "Готов",
        "COMPLETED" => "Выдан",
        "CANCELLED" => "Отменён",
        _ => status,
    }
}

fn pdf_error(error: impl std::fmt::Display) -> String {
    format!("could not create PDF receipt: {error}")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn renders_cyrillic_receipt_as_pdf() {
        let configured =
            std::env::var_os("PRINTFORGE_RECEIPT_PREVIEW").map(std::path::PathBuf::from);
        let directory = configured.clone().unwrap_or_else(|| {
            std::env::temp_dir().join(format!("printforge-receipt-{}", uuid::Uuid::new_v4()))
        });
        let result = generate(
            &Order {
                id: "order-preview".into(),
                number: "ORD-2026-00042".into(),
                tracking_code: "MFHP73PH7K".into(),
                customer_id: None,
                customer_name: "Иван Петров".into(),
                title: "Корпус редуктора - 3D-печать PLA".into(),
                status: "READY".into(),
                deadline: Some("2026-08-28T17:00:00+03:00".into()),
                selling_price: 355.87,
                paid_amount: 150.0,
                created_at: "2026-08-24T14:39:50+03:00".into(),
                models: Vec::new(),
            },
            &Settings {
                company_name: "PrintForge Studio".into(),
                currency: "MDL".into(),
                electricity_price_per_kwh: 2.58,
                machine_rate_per_hour: 25.0,
                labour_rate_per_hour: 50.0,
                default_markup_percent: 40.0,
                low_stock_threshold_grams: 200.0,
            },
            &directory,
        )
        .expect("render receipt");
        let bytes = std::fs::read(&result.path).expect("read receipt");
        assert!(bytes.starts_with(b"%PDF"));
        assert!(bytes.len() > 10_000);
        if configured.is_none() {
            std::fs::remove_dir_all(directory).expect("remove test receipt");
        }
    }
}
