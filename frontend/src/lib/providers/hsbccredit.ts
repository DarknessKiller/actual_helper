import type { ActualBudgetReport, ProviderConfig } from '../types'
import { createEngine } from '../rules'
import { normalizeWhitespace, extractAfterMarker } from '../cardutil'
import { formatDate } from '../dateutil'

export function parseHSBCCredit(text: string, config: ProviderConfig): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []

  const account = extractAfterMarker(text, 'Card Number', 'HSBC Credit Card')

  const dateMatch = text.match(/Statement Date\s+(\d{2}\s+\w{3}\s+\d{4})/)
  if (!dateMatch) return reports
  const stmtDate = new Date(dateMatch[1])
  if (isNaN(stmtDate.getTime())) return reports

  // Strip pipes, brackets
  const cleaned = text.replace(/[|\[\]]/g, '')

  const txRe = /^(\d{2}\s+\w{3})\s+(\d{2}\s+\w{3})\s+(.+?)\s+([\d,.]+)(CR)?\s*$/gm
  let match: RegExpExecArray | null
  let started = false

  while ((match = txRe.exec(cleaned)) !== null) {
    const postDate = formatDate(match[1], stmtDate)
    const transDate = formatDate(match[2], stmtDate)
    let description = normalizeWhitespace(match[3])
    const amount = parseFloat(match[4].replace(/,/g, ''))
    const isCredit = !!match[5]

    if (!postDate || !transDate || isNaN(amount)) continue

    // Skip summary lines
    if (/Your Previous Statement Balance|Credit limit used|Your Credit Limit|Your charge\(s\)|Total credit limit used|Your statement balance/i.test(description)) continue

    // Skip until we find a real transaction
    if (!started && amount === 0) continue
    started = true

    if (engine.shouldSkip(description)) continue

    const [group, category] = engine.matchCategory(description)
    const sign = isCredit ? amount : -amount

    reports.push({
      Account: account,
      Date: transDate,
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
