package models_test

import (
	"testing"

	"actual_helper/internal/models"
)

func TestMaskAccountNumber(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"card number", "1234 5678 9012 3456", "**** **** **** 3456"},
		{"account number", "12345678901", "*******8901"},
		{"minimum maskable", "12345", "*2345"},
		{"four digits are not enough to mask", "1234", "1234"},
		{"name unchanged", "HLB Debit Account", "HLB Debit Account"},
		{"empty unchanged", "", ""},
		{"mixed text and digits", "A/C 12345678", "A/C ****5678"},
		{"already masked is idempotent", "**** **** **** 3456", "**** **** **** 3456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := models.MaskAccountNumber(tt.in); got != tt.want {
				t.Errorf("MaskAccountNumber(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
