import { describe, it, expect } from 'vitest'
import { parseGXBank } from '../providers/gxbank'
import type { ProviderConfig } from '../types'

const emptyConfig: ProviderConfig = {
  exclude_keywords: [],
  include_keywords: [],
  categories: [],
  account_mappings: {}
}

describe('GXBank provider', () => {
  it('returns empty when no marker found', () => {
    const reports = parseGXBank('random text without marker', emptyConfig)
    expect(reports).toHaveLength(0)
  })

  it('parses single interest earned transaction as credit', () => {
    const text = `May 2026
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

    const reports = parseGXBank(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Date).toBe('2026-06-01')
    expect(reports[0].Notes).toContain('Interest earned')
    expect(reports[0].Amount).toBe('0.55')
  })

  it('joins multi-line description', () => {
    const text = `May 2026
Closing balance (RM)
Baki penutup
21 May 2026
12:00 AM
Opening balance
0.00
21 May
12:09 AM
Pocket
Withdraw from Pocket
+10,097.90
10,097.90`

    const reports = parseGXBank(text, emptyConfig)
    expect(reports).toHaveLength(1)
    expect(reports[0].Notes).toContain('Pocket')
    expect(reports[0].Notes).toContain('Withdraw from Pocket')
  })

  it('skips opening balance and parses multiple transactions', () => {
    const text = `May 2026
Closing balance (RM)
Baki penutup
1 Jun 2026
12:00 AM
Opening balance
100.00
5 Jun
3:00 PM
Lunch
-25.00
75.00
10 Jun
9:00 AM
Salary
+5000.00
5075.00`

    const reports = parseGXBank(text, emptyConfig)
    expect(reports).toHaveLength(2)
    expect(reports[0].Notes).toContain('Lunch')
    expect(reports[0].Amount).toBe('-25.00')
    expect(reports[1].Notes).toContain('Salary')
    expect(reports[1].Amount).toBe('5000.00')
  })
})
