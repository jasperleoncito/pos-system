package service

import "testing"

func TestInclusiveTaxPortion(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		rate   float64
		want   int64
	}{
		// ₱160.00 at 12% inclusive → tax portion 16000*12/112 = 1714.28… → 1714
		{"php160 at 12pct", 16000, 12, 1714},
		{"php100 at 12pct", 10000, 12, 1071},
		{"zero rate", 10000, 0, 0},
		{"zero amount", 0, 12, 0},
		{"rounds half up", 112, 12, 12}, // 112*12/112 = 12
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InclusiveTaxPortion(tt.amount, tt.rate); got != tt.want {
				t.Errorf("InclusiveTaxPortion(%d, %v) = %d, want %d", tt.amount, tt.rate, got, tt.want)
			}
		})
	}
}

func TestExclusiveTaxAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		rate   float64
		want   int64
	}{
		{"php100 at 12pct", 10000, 12, 1200},
		{"php1.55 at 10pct rounds", 155, 10, 16}, // 15.5 → 16 half-up
		{"zero rate", 10000, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExclusiveTaxAmount(tt.amount, tt.rate); got != tt.want {
				t.Errorf("ExclusiveTaxAmount(%d, %v) = %d, want %d", tt.amount, tt.rate, got, tt.want)
			}
		})
	}
}
