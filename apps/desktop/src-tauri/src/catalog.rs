use std::sync::OnceLock;

use serde::Deserialize;

use crate::models::PrinterCatalogModel;

#[derive(Clone, Deserialize)]
struct Catalog {
    models: Vec<PrinterCatalogModel>,
}

static CATALOG: OnceLock<Result<Catalog, String>> = OnceLock::new();

fn catalog() -> Result<&'static Catalog, String> {
    CATALOG
        .get_or_init(|| {
            serde_json::from_str(include_str!(
                "../../../backend/internal/http/printer_catalog.json"
            ))
            .map_err(|error| format!("could not load bundled printer catalog: {error}"))
        })
        .as_ref()
        .map_err(Clone::clone)
}

pub fn models() -> Result<Vec<PrinterCatalogModel>, String> {
    Ok(catalog()?.models.clone())
}

pub fn count() -> usize {
    catalog()
        .map(|value| value.models.len())
        .unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bundles_current_major_manufacturers() {
        let models = models().expect("catalog");
        assert_eq!(models.len(), 387);
        assert!(models.iter().any(|item| item.manufacturer == "Bambu Lab"));
        assert!(models.iter().any(|item| item.manufacturer == "Creality"));
        assert!(models.iter().any(|item| item.manufacturer == "Anycubic"));
    }
}
