package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"actual_helper/internal/providers/gxbank"
	"actual_helper/internal/providers/tng"
)

func TestTNGWASM(t *testing.T) {
	text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
Payment
111111
Merchant A
222222
RM34.00
RM5.10`

	prov := tng.New(nil, nil, nil, nil)
	reports, err := prov.ParsePDFText(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].Amount != "-34.00" {
		t.Errorf("expected -34.00, got %s", reports[0].Amount)
	}
	fmt.Printf("TNG OK: %s %s %s\n", reports[0].Date, reports[0].Notes, reports[0].Amount)
}

func TestGXWASM(t *testing.T) {
	text := `May 2026
Closing balance (RM)
Baki penutup
1 Jun 2026
12:00 AM
Opening balance
10,006.05
1 Jun
11:59 PM
Interest earned
+0.55
10,006.60`

	prov := gxbank.New(nil, nil, nil, nil)
	reports, err := prov.ParsePDFText(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	b, _ := json.MarshalIndent(reports[0], "", "  ")
	fmt.Printf("GXBank OK: %s\n", string(b))
}
