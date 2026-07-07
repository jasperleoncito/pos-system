package service

import "math"

// Tax math for centavo amounts.
//
// Inclusive taxes are already inside the price: the tax portion of a
// line is line * rate / (100 + rate). Exclusive taxes are added on top:
// line * rate / 100. Results round half-up to the nearest centavo.

func roundHalfUp(v float64) int64 {
	return int64(math.Floor(v + 0.5))
}

// InclusiveTaxPortion returns the tax already contained in amount.
func InclusiveTaxPortion(amount int64, ratePercent float64) int64 {
	if ratePercent <= 0 {
		return 0
	}
	return roundHalfUp(float64(amount) * ratePercent / (100 + ratePercent))
}

// ExclusiveTaxAmount returns the tax to add on top of amount.
func ExclusiveTaxAmount(amount int64, ratePercent float64) int64 {
	if ratePercent <= 0 {
		return 0
	}
	return roundHalfUp(float64(amount) * ratePercent / 100)
}
