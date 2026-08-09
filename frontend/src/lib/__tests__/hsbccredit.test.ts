import { describe, it, expect } from 'vitest'
import { parseHSBCCredit } from '../providers/hsbccredit'
import type { ProviderConfig } from '../types'

const emptyConfig: ProviderConfig = {
  exclude_keywords: [],
  include_keywords: [],
  categories: [],
  account_mappings: {}
}

describe('HSBC Credit provider', () => {
  it('parses credit card statement', () => {
    const text = `Card Number: 4321 8765 4321 8765
Statement Date 15 Jan 2026
Post date    Trans date   Description                  Amount
16 Jan       15 Jan       SHOPPING MALL                  150.00
17 Jan       16 Jan       PAYMENT RECEIVED - THANK YOU   500.00 CR`

    const reports = parseHSBCCredit(text, emptyConfig)
    expect(reports.length).toBeGreaterThanOrEqual(1)
  })
})
