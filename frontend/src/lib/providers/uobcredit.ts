import type { ActualBudgetReport, ProviderConfig } from '../types'
import { createEngine } from '../rules'
import { normalizeWhitespace, extractNearCardType } from '../cardutil'
import { formatDate, parseStatementDate } from '../dateutil'

export function parseUOBCredit(text: string, config: ProviderConfig): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []

  const account = extractNearCardType(text, ['WORLD MASTERCARD', 'MASTERCARD', 'VISA'], 'UOB Credit Card')

  const dateMatch = text.match(/Statement Date\s+(\d{2}\s+\w{3}\s+\d{2,4})/)
  if (!dateMatch) return reports
  const stmtDate = parseStatementDate(dateMatch[1])
  if (!stmtDate) return reports

  const txRe = /^\s*(\d{2}\s+\w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?\s*$/gm
  let match: RegExpExecArray | null

  while ((match = txRe.exec(text)) !== null) {
    const dateStr = formatDate(match[1], stmtDate)
    const description = normalizeWhitespace(match[2])
    const amount = parseFloat(match[3].replace(/,/g, ''))
    const isCredit = !!match[4]

    if (!dateStr || isNaN(amount)) continue

    const lower = description.toLowerCase()
    if (lower.includes('sub-total') || lower.includes('minimum payment due') ||
      lower.includes('** end of statement**') || lower.includes('credit limit') ||
      lower.includes('previous bal') || lower.includes('page no')) continue

    if (engine.shouldSkip(description)) continue

    const [group, category] = engine.matchCategory(description)
    const sign = isCredit ? amount : -amount

    reports.push({
      Account: account,
      Date: dateStr,
      Payee: '',
      Notes: description,
      Category_Group: group,
      Category: category,
      Amount: sign.toFixed(2),
      Split_Amount: '',
      Cleared: 'Cleared'
    })
  }

  return reports
}
