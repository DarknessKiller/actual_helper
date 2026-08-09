import { describe, it, expect } from 'vitest'
import { parseTNG } from '../providers/tng'
import type { ProviderConfig } from '../types'

const emptyConfig: ProviderConfig = {
  exclude_keywords: [],
  include_keywords: [],
  categories: [],
  account_mappings: {}
}

describe('TNG provider', () => {
  it('parses a payment as debit (negative amount)', () => {
    const text = `TNG WALLET TRANSACTION
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

    const reports = parseTNG(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Payee).toBe('')
    expect(reports[0].Amount).toBe('-34.00')
  })

  it('parses a reload as credit (positive amount)', () => {
    const text = `TNG WALLET TRANSACTION
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
Reload
111111
Top Up from Bank

RM100.00
RM150.00`

    const reports = parseTNG(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Payee).toBe('')
    expect(reports[0].Amount).toBe('100.00')
  })

  it('parses DUITNOW_RECEIVEFROM as credit', () => {
    const text = `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
3/5/2026

Success
DUITNOW_RECEIVEFROM
111111
Bob

RM100.00
RM105.10`

    const reports = parseTNG(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Amount).toBe('100.00')
  })

  it('parses multiple transactions', () => {
    const text = `TNG WALLET TRANSACTION
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

RM34.00
RM50.00
2/5/2026
Success
Reload
222222
Top Up

RM100.00
RM150.00`

    const reports = parseTNG(text, emptyConfig)
    expect(reports).toHaveLength(2)
    expect(reports[0].Amount).toBe('-34.00')
    expect(reports[1].Amount).toBe('100.00')
  })

  it('returns empty for text without transaction section', () => {
    const reports = parseTNG('random text', emptyConfig)
    expect(reports).toHaveLength(0)
  })

  it('handles date with single-digit day', () => {
    const text = `TNG WALLET TRANSACTION
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
Test Merchant

RM25.50
RM100.00`

    const reports = parseTNG(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Date).toBe('2026-05-01')
  })

  it('handles date with double-digit day and month', () => {
    const text = `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
01/12/2026
Success
Reload
111111
Salary

RM1000.00
RM2000.00`

    const reports = parseTNG(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Date).toBe('2026-12-01')
  })
})
