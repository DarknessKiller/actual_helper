import type { ActualBudgetReport } from './types'

const HEADER = ['Account', 'Date', 'Payee', 'Notes', 'Category_Group', 'Category', 'Amount', 'Split_Amount', 'Cleared']

function csvEscape(field: string): string {
  if (field === '') return ''
  if (field.includes(',') || field.includes('"') || field.includes('\n') || field.includes('\r')) {
    return '"' + field.replace(/"/g, '""') + '"'
  }
  return field
}

function reportToRow(r: ActualBudgetReport): string[] {
  return [r.Account, r.Date, r.Payee, r.Notes, r.Category_Group, r.Category, r.Amount, r.Split_Amount, r.Cleared]
}

export function toActualCSV(reports: ActualBudgetReport[]): string {
  const rows = [HEADER.map(csvEscape).join(',')]
  for (const r of reports) {
    rows.push(reportToRow(r).map(csvEscape).join(','))
  }
  return rows.join('\n') + '\n'
}

export function toActualCSVBlob(reports: ActualBudgetReport[]): Blob {
  return new Blob([toActualCSV(reports)], { type: 'text/csv;charset=utf-8' })
}
