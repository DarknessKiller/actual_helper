package textutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollapse(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello   world", "hello world"},
		{"  leading and trailing  ", "leading and trailing"},
		{"\t\ntabs\nand\nnewlines\t", "tabs and newlines"},
		{"", ""},
		{"single", "single"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			require.Equal(t, tt.want, Collapse(tt.in))
		})
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		in        string
		prefixes  []string
		want      float64
		shouldErr bool
	}{
		{"1,234.56", nil, 1234.56, false},
		{"RM 100.00", []string{"RM"}, 100.00, false},
		{"+100.00", []string{"+"}, 100.00, false},
		{"", nil, 0, true},
		{"abc", nil, 0, true},
		{"  42.5  ", nil, 42.5, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseAmount(tt.in, tt.prefixes...)
			if tt.shouldErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.InDelta(t, tt.want, got, 0.0001)
		})
	}
}

func TestParseDateMulti(t *testing.T) {
	got, err := ParseDateMulti("5 January 2026", "2 January 2006", "2 Jan 2006")
	require.NoError(t, err)
	require.Equal(t, 2026, got.Year())
	require.Equal(t, time.January, got.Month())
	require.Equal(t, 5, got.Day())

	_, err = ParseDateMulti("nope", "2 January 2006")
	require.Error(t, err)
}
