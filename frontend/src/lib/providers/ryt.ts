import type { ActualBudgetReport, ProviderConfig } from '../types'
import { createEngine } from '../rules'
import { normalizeWhitespace } from '../cardutil'
import { parseDayMonthYear, toLocalDate } from '../dateutil'

export function parseRYT(text: string, config: ProviderConfig): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []

  // Extract account name
  let account = 'RYT Bank Account'
  const acctIdx = text.indexOf('Account Transactions')
  if (acctIdx !== -1) {
    const after = text.slice(acctIdx + 'Account Transactions'.length)
    const lines = after.split('\n').filter(l => l.trim())
    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed) continue
      if (trimmed.includes('/')) {
        account = trimmed.split('/')[0].trim()
      } else if (!/^(Savings|Current)/.test(trimmed)) {
        account = trimmed
      }
      break
    }
  }

  // Find data start after Baki header
  const bakiIdx = text.indexOf('Baki')
  if (bakiIdx === -1) return reports
  const dataStart = text.indexOf('\n', bakiIdx + 4) + 1
  const content = text.slice(dataStart)

  // Remove page headers
  const cleaned = content.replace(/Savings Account Statement[\s\S]*?Baki\s*\n?/g, '')

  // Split into date blocks
  const dateRe = /^\s*(\d{1,2}\s+[A-Za-z]+\s+\d{4})\b/gm
  const blocks: string[] = []
  let lastIdx = 0
  let match: RegExpExecArray | null
  const indices: number[] = []

  while ((match = dateRe.exec(cleaned)) !== null) {
    indices.push(match.index)
  }

  for (let i = 0; i < indices.length; i++) {
    const start = indices[i]
    const end = indices[i + 1] ?? cleaned.length
    blocks.push(cleaned.slice(start, end))
  }

  for (const block of blocks) {
    const lines = block.split('\n').map(l => l.trim()).filter(l => l)
    if (lines.length < 2) continue

    const firstLine = lines[0]
    const dateMatch = firstLine.match(/^\s*(\d{1,2}\s+[A-Za-z]+\s+\d{4})/)
    if (!dateMatch) continue

    const date = parseDayMonthYear(dateMatch[1])
    if (!date) continue

    // Find amount: last non-empty line with +/- prefix
    let amount = 0
    let amountFound = false
    for (let j = lines.length - 1; j >= 1; j--) {
      const m = lines[j].match(/^([+-][\d,.]+)$/)
      if (m) {
        amount = parseFloat(m[1].replace(/,/g, ''))
        amountFound = true
        break
      }
    }
    if (!amountFound) continue

    // Description: lines between date line and amount, joined with " / "
    const descLines: string[] = []
    for (let j = 1; j < lines.length; j++) {
      if (/^[+-][\d,.]+$/.test(lines[j])) break
      descLines.push(lines[j])
    }
    const description = normalizeWhitespace(descLines.join(' / '))
    if (!description) continue

    if (/opening balance/i.test(description)) continue
    if (engine.shouldSkip(description)) continue

    const [group, category] = engine.matchCategory(description)

    reports.push({
      Account: account,
      Date: toLocalDate(date),
      Payee: '',
      Notes: description,
      Category_Group: group,
      Category: category,
      Amount: amount.toFixed(2),
      Split_Amount: '',
      Cleared: 'Cleared'
    })
  }

  return reports
}
