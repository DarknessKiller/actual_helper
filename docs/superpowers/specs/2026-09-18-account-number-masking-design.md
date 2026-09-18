# Account Number Masking in Logs

## Problem

Several providers use the statement's raw account/card number as the account name:

| Provider | Source | Example value |
|---|---|---|
| `hsbccredit` | `cardutil.ExtractAfterMarker(text, "Card Number", …)` | `1234 5678 9012 3456` |
| `uobcredit` | `cardutil.ExtractNearCardType(…, "VISA"/"MASTERCARD", …)` | `1234 5678 9012 3456` |
| `hlb` credit | `cardutil.ExtractAfterMarker(text, "Credit Card Number", …)` | `1234 5678 9012 3456` |
| `hlb` debit | `extractAccountFromMarkers(text, ["A/C No", "No Akaun"], …)` | `12345678901` |

Every provider logs it (`logger.InfoContext(…, "account", accountName)`), so the full account number ends up in server logs. The CSV `Account` column is **not** in scope: the raw value is what the user maps to their Actual Budget accounts, so output behavior stays unchanged.

## Design

### Masking rule

`models.MaskAccountNumber(s string) string`: keep the last 4 digits, replace every earlier digit with `*`, preserve separators and non-digit characters. Fewer than 5 digits → unchanged, so names and fallbacks are unaffected. Idempotent.

| Input | Output |
|---|---|
| `1234 5678 9012 3456` | `**** **** **** 3456` |
| `12345678901` | `*******8901` |
| `HLB Debit Account` | unchanged |
| `GX Bank` | unchanged |
| `""` | unchanged |

### Where it is applied

The six `"account", accountName` log fields in provider services log `models.MaskAccountNumber(accountName)` instead of the raw value. Debugging signal (which account was detected) is kept, the number is not.

### Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/models/mask.go` | Create | `MaskAccountNumber` |
| `internal/providers/{hsbccredit,ryt,gxbank,uobcredit,hlb}/service.go` | Modify | Log masked account |
| `internal/models/mask_test.go` | Create | Unit tests for the mask rule |
| `internal/services/convert_test.go` | Modify | Regression test: `ToActualCSV` keeps accounts unmasked |

### Tests

| Test | Description |
|------|-------------|
| Masks full card number | `1234 5678 9012 3456` → `**** **** **** 3456` |
| Masks long account number | `12345678901` → `*******8901` |
| Leaves short/name values alone | `1234`, `HLB Debit Account`, `""` unchanged |
| Idempotent | masked value stays masked |
| CSV output keeps the account | `ToActualCSV` row contains the unmasked account |

## Non-goals

- **CSV output**: the `Account` column keeps the raw value; users rely on it for `account_mappings` and account recognition.
- Transaction descriptions (`Notes`) may still mention other accounts (e.g. transfer references); masking those would destroy the information the user imports into Actual Budget.

## No other changes

Extraction, mapping, filtering, categorization, and CSV output are untouched.
