use rust_decimal::prelude::{FromPrimitive, ToPrimitive};
use rust_decimal::{Decimal, RoundingStrategy};

use crate::models::{CostBreakdown, CostInput};

fn decimal(value: f64, field: &str) -> Result<Decimal, String> {
    if !value.is_finite() || value < 0.0 {
        return Err(format!("{field} must be a finite non-negative number"));
    }
    Decimal::from_f64(value).ok_or_else(|| format!("invalid {field}"))
}

fn money(value: Decimal) -> Decimal {
    value.round_dp_with_strategy(2, RoundingStrategy::MidpointAwayFromZero)
}

fn output(value: Decimal) -> f64 {
    value.to_f64().unwrap_or(0.0)
}

pub fn calculate(input: CostInput) -> Result<CostBreakdown, String> {
    let minutes = Decimal::from(input.print_minutes);
    let hours = minutes / Decimal::from(60u32);
    let filament_grams = decimal(input.filament_grams, "filamentGrams")?;
    let filament_price = decimal(input.filament_price_per_gram, "filamentPricePerGram")?;
    let power_watts = decimal(input.power_watts, "powerWatts")?;
    let electricity_price = decimal(input.electricity_price_per_kwh, "electricityPricePerKwh")?;
    let machine_rate = decimal(input.machine_rate_per_hour, "machineRatePerHour")?;
    let depreciation_rate = decimal(input.depreciation_per_hour, "depreciationPerHour")?;
    let operator_hours = decimal(input.operator_hours, "operatorHours")?;
    let labour_rate = decimal(input.labour_rate_per_hour, "labourRatePerHour")?;
    let post_processing = decimal(input.post_processing_cost, "postProcessingCost")?;
    let packaging = decimal(input.packaging_cost, "packagingCost")?;
    let other = decimal(input.other_cost, "otherCost")?;
    let markup_percent = decimal(input.markup_percent, "markupPercent")?;

    let material_cost = money(filament_grams * filament_price);
    let energy_kwh = (power_watts / Decimal::from(1000u32) * hours).round_dp(5);
    let electricity_cost = money(energy_kwh * electricity_price);
    let machine_cost = money(hours * machine_rate);
    let depreciation_cost = money(hours * depreciation_rate);
    let labour_cost = money(operator_hours * labour_rate);
    let total_cost = money(
        material_cost
            + electricity_cost
            + machine_cost
            + depreciation_cost
            + labour_cost
            + post_processing
            + packaging
            + other,
    );
    let markup_amount = money(total_cost * markup_percent / Decimal::from(100u32));
    let suggested_price = money(total_cost + markup_amount);

    Ok(CostBreakdown {
        material_cost: output(material_cost),
        energy_kwh: output(energy_kwh),
        electricity_cost: output(electricity_cost),
        machine_cost: output(machine_cost),
        depreciation_cost: output(depreciation_cost),
        labour_cost: output(labour_cost),
        post_processing_cost: output(money(post_processing)),
        packaging_cost: output(money(packaging)),
        other_cost: output(money(other)),
        total_cost: output(total_cost),
        markup_amount: output(markup_amount),
        suggested_price: output(suggested_price),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn calculates_electricity_and_full_price_with_decimal_rounding() {
        let result = calculate(CostInput {
            print_minutes: 270,
            filament_grams: 184.0,
            filament_price_per_gram: 0.45,
            power_watts: 120.0,
            electricity_price_per_kwh: 2.58,
            machine_rate_per_hour: 25.0,
            depreciation_per_hour: 5.0,
            operator_hours: 0.5,
            labour_rate_per_hour: 50.0,
            post_processing_cost: 0.0,
            packaging_cost: 10.0,
            other_cost: 0.0,
            markup_percent: 40.0,
        })
        .expect("cost calculation");

        assert_eq!(result.energy_kwh, 0.54);
        assert_eq!(result.electricity_cost, 1.39);
        assert_eq!(result.material_cost, 82.8);
        assert_eq!(result.total_cost, 254.19);
        assert_eq!(result.suggested_price, 355.87);
    }

    #[test]
    fn rejects_negative_input() {
        let result = calculate(CostInput {
            print_minutes: 1,
            filament_grams: -1.0,
            filament_price_per_gram: 1.0,
            power_watts: 1.0,
            electricity_price_per_kwh: 1.0,
            machine_rate_per_hour: 1.0,
            depreciation_per_hour: 1.0,
            operator_hours: 1.0,
            labour_rate_per_hour: 1.0,
            post_processing_cost: 0.0,
            packaging_cost: 0.0,
            other_cost: 0.0,
            markup_percent: 0.0,
        });
        assert!(result.is_err());
    }
}
