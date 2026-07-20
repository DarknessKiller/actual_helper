package cardutil_test

import (
	"testing"

	"actual_helper/internal/providers/cardutil"

	"github.com/stretchr/testify/require"
)

func TestExtractAfterMarker(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		marker   string
		fallback string
		want     string
	}{
		{
			name:     "finds card number after marker",
			text:     "Card Number\n1234-5678-9012-3456",
			marker:   "Card Number",
			fallback: "Fallback",
			want:     "1234 5678 9012 3456",
		},
		{
			name:     "returns fallback when marker not found",
			text:     "No card info here",
			marker:   "Card Number",
			fallback: "Fallback",
			want:     "Fallback",
		},
		{
			name:     "returns fallback when no card number after marker",
			text:     "Card Number\nno digits here",
			marker:   "Card Number",
			fallback: "Fallback",
			want:     "Fallback",
		},
		{
			name:     "handles card number with spaces",
			text:     "Credit Card Number 1234 5678 9012 3456",
			marker:   "Credit Card Number",
			fallback: "Fallback",
			want:     "1234 5678 9012 3456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardutil.ExtractAfterMarker(tt.text, tt.marker, tt.fallback)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExtractNearCardType(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		cardTypes []string
		fallback  string
		want      string
	}{
		{
			name:      "finds card number near WORLD MASTERCARD",
			text:      "some text WORLD MASTERCARD 1234-5678-9012-3456 more text",
			cardTypes: []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"},
			fallback:  "Fallback",
			want:      "1234-5678-9012-3456",
		},
		{
			name:      "finds card number near VISA with spaces",
			text:      "VISA ending 1234 5678 9012 3456",
			cardTypes: []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"},
			fallback:  "Fallback",
			want:      "1234 5678 9012 3456",
		},
		{
			name:      "returns fallback when no card type found",
			text:      "no card info here",
			cardTypes: []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"},
			fallback:  "Fallback",
			want:      "Fallback",
		},
		{
			name:      "returns fallback when card type found but no card number nearby",
			text:      "WORLD MASTERCARD no digits",
			cardTypes: []string{"WORLD MASTERCARD"},
			fallback:  "Fallback",
			want:      "Fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardutil.ExtractNearCardType(tt.text, tt.cardTypes, tt.fallback)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestApplyMapping(t *testing.T) {
	tests := []struct {
		name    string
		mapping map[string]string
		input   string
		want    string
	}{
		{
			name:    "maps existing key",
			mapping: map[string]string{"1234-5678-9012-3456": "My Card"},
			input:   "1234-5678-9012-3456",
			want:    "My Card",
		},
		{
			name:    "returns original when key not found",
			mapping: map[string]string{"1234-5678-9012-3456": "My Card"},
			input:   "9999-8888-7777-6666",
			want:    "9999-8888-7777-6666",
		},
		{
			name:    "returns original when mapping is nil",
			mapping: nil,
			input:   "1234-5678-9012-3456",
			want:    "1234-5678-9012-3456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardutil.ApplyMapping(tt.mapping, tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
