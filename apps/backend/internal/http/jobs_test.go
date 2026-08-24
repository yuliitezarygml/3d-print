package httpapi

import (
	"math"
	"testing"
)

func TestCalculateCostsIncludesElectricityAndDepreciation(t *testing.T) {
	got := calculateCosts(270, 184, 350, 430, 1000, 2.58, 25, 34900, 5000, .5, 50, 10, 8, 2, 40)
	wantEnergy := 1.575
	if math.Abs(got.EnergyKwh-wantEnergy) > .00001 {
		t.Fatalf("energy = %v, want %v", got.EnergyKwh, wantEnergy)
	}
	if math.Abs(got.ElectricityCost-wantEnergy*2.58) > .00001 {
		t.Fatalf("electricity cost = %v", got.ElectricityCost)
	}
	if got.MaterialCost <= 0 || got.MachineCost <= 0 || got.SuggestedPrice <= got.TotalCost {
		t.Fatalf("incomplete cost calculation: %+v", got)
	}
}
