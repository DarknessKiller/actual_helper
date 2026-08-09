import { describe, it, expect } from 'vitest'
import { parseUOBCredit } from '../providers/uobcredit'
import type { ProviderConfig } from '../types'

const emptyConfig: ProviderConfig = {
  exclude_keywords: [],
  include_keywords: [],
  categories: [],
  account_mappings: {}
}

describe('UOB Credit provider', () => {
  it('parses credit card statement', () => {
    const text = `WORLD MASTERCARD
Statement Date 01 Jan 2026
01 Jan    SHOPPING MALL              150.00
02 Jan    PAYMENT RECEIVED           500.00 CR
** end of statement**`

    const reports = parseUOBCredit(text, emptyConfig)
    expect(reports.length).toBeGreaterThanOrEqual(1)
  })
})
