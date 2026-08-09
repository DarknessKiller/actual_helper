import { describe, it, expect } from 'vitest'
import { toActualCSV, toActualCSVBlob } from '../csv'
import type { ActualBudgetReport } from '../types'

describe('csv', () => {
  const report: ActualBudgetReport = {
    Account: 'Test Account',
    Date: '2026-01-15',
    Payee: '',
    Notes: 'Test transaction',
    Category_Group: 'Shopping',
    Category: 'Online',
    Amount: '-25.00',
    Split_Amount: '',
    Cleared: 'Cleared'
  }

  it('generates correct CSV header', () => {
    const csv = toActualCSV([])
    const header = csv.split('\n')[0]
    expect(header).toBe('Account,Date,Payee,Notes,Category_Group,Category,Amount,Split_Amount,Cleared')
  })

  it('generates correct CSV row', () => {
    const csv = toActualCSV([report])
    const lines = csv.trim().split('\n')
    expect(lines).toHaveLength(2)
    expect(lines[1]).toContain('Test Account')
    expect(lines[1]).toContain('2026-01-15')
    expect(lines[1]).toContain('-25.00')
  })

  it('escapes fields with commas', () => {
    const r = { ...report, Notes: 'Hello, World' }
    const csv = toActualCSV([r])
    expect(csv).toContain('"Hello, World"')
  })

  it('creates blob with correct type', () => {
    const blob = toActualCSVBlob([report])
    expect(blob.type).toBe('text/csv;charset=utf-8')
  })
})
