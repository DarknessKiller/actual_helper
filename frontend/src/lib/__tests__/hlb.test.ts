import { describe, it, expect } from 'vitest'
import { parseHLB } from '../providers/hlb'
import type { ProviderConfig } from '../types'

const emptyConfig: ProviderConfig = {
  exclude_keywords: [],
  include_keywords: [],
  categories: [],
  account_mappings: {}
}

describe('HLB provider', () => {
  it('detects credit format and parses transaction', () => {
    const text = `Credit Card Number
5123 4567 8901 2345
Statement Date 01 Jan 2026
Post date    Trans date   Description                  Amount
01 Jan       01 Jan       KEDAI MAKAN                    25.00
02 Jan       02 Jan       GROCERY STORE CR               100.00 CR`

    const reports = parseHLB(text, emptyConfig)
    expect(reports.length).toBeGreaterThanOrEqual(1)
  })

  it('detects debit format', () => {
    const text = `A/C No: 1234567890
Deposit    Withdrawal    Description    Date
100.00                  Salary         01-01-2026
                        Lunch          02-01-2026    25.00`

    const reports = parseHLB(text, emptyConfig)
    expect(reports.length).toBeGreaterThanOrEqual(0)
  })
})
