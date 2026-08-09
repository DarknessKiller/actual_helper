import { describe, it, expect } from 'vitest'
import { parseRYT } from '../providers/ryt'
import type { ProviderConfig } from '../types'

const emptyConfig: ProviderConfig = {
  exclude_keywords: [],
  include_keywords: [],
  categories: [],
  account_mappings: {}
}

describe('RYT provider', () => {
  it('parses a credit transaction', () => {
    const text = `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 Mar 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: REF20260301ABCDEF1
+123.45
123.45`

    const reports = parseRYT(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Amount).toBe('123.45')
    expect(reports[0].Date).toBe('2026-03-01')
    expect(reports[0].Payee).toBe('')
    expect(reports[0].Notes).toContain('From Alice Tan')
  })

  it('parses a debit transaction', () => {
    const text = `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
2 Mar 2026
To Savings Goal
Money movement
Ref. ID: REF20260302GHIJKL2
-456.78
0.00`

    const reports = parseRYT(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Amount).toBe('-456.78')
    expect(reports[0].Date).toBe('2026-03-02')
  })

  it('skips opening balance row', () => {
    const text = `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 Mar 2026
Opening balance
0.26
1 Mar 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: REF20260301ABCDEF1
+123.45
123.45`

    const reports = parseRYT(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Notes).toContain('From Alice Tan')
  })
})
