import type { ActualBudgetReport, ProviderConfig } from '../types'
import { createEngine } from '../rules'
import { normalizeWhitespace } from '../cardutil'
import { parseDayMonthYear, toLocalDate } from '../dateutil'

export function parseGXBank(text: string, config: ProviderConfig): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []

  // Extract account name
  let account = 'GX Bank Account'
  const soaIdx = text.indexOf('Statements of Accounts')
  if (soaIdx !== -1) {
    const after = text.slice(soaIdx + 'Statements of Accounts'.length)
    const lines = after.split('\n').filter(l => l.trim())
    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed) continue
      if (/^(January|February|March|April|May|June|July|August|September|October|November|December)/i.test(trimmed)) continue
      if (/Account number/i.test(trimmed)) continue
      account = trimmed
      break
    }
  }

  // Statement year
  let stmtYear = new Date().getFullYear()
  const yearMatch = text.match(/^(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})$/m)
  if (yearMatch) stmtYear = parseInt(yearMatch[2], 10)

  // Find data start after "Closing balance (RM)"
  const closingIdx = text.indexOf('Closing balance (RM)')
  if (closingIdx === -1) return reports
  const dataStart = text.indexOf('\n', closingIdx + 1) + 1
  const content = text.slice(dataStart)

  // State machine: date -> time -> desc(s) -> amount -> balance
  const dateRe = /^\d{1,2}\s+(?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|Jul(?:y)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)(?:\s+\d{4})?$/i
  const timeRe = /^\d{1,2}:\d{2}\s*(?:AM|PM)$/i
  const amountRe = /^[+-][\d,.]+$/
  const balanceRe = /^[\d,.]+$/

  let curDate = ''
  let curTime = ''
  let curDescs: string[] = []
  let curAmount = ''

  const flush = () => {
    if (!curDate || !curAmount) return
    if (curDate.toLowerCase().includes('opening balance')) return

    const amount = parseFloat(curAmount.replace(/[+,-]/g, ''))
    if (isNaN(amount)) return
    const isCredit = curAmount.startsWith('+')

    const description = normalizeWhitespace(curDescs.join(' '))
    if (!description) return
    if (/opening balance/i.test(description)) return
    if (engine.shouldSkip(description)) return

    const [group, category] = engine.matchCategory(description)
    const sign = isCredit ? amount : -amount

    reports.push({
      Account: account,
      Date: curDate,
      Payee: '',
      Notes: description,
      Category_Group: group,
      Category: category,
      Amount: sign.toFixed(2),
      Split_Amount: '',
      Cleared: 'Cleared'
    })
  }

  const lines = content.split('\n')
  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) continue

    // Date line
    if (dateRe.test(line)) {
      flush()
      // Append year if missing
      if (/\d{4}$/.test(line)) {
        curDate = line
      } else {
        curDate = line + ' ' + stmtYear
      }
      const d = parseDayMonthYear(curDate)
      curDate = d ? toLocalDate(d) : curDate
      curTime = ''
      curDescs = []
      curAmount = ''
      continue
    }

    // Time line
    if (timeRe.test(line)) {
      curTime = line
      continue
    }

    // Amount line
    if (amountRe.test(line)) {
      curAmount = line
      continue
    }

    // Balance line (skip — not needed)
    if (balanceRe.test(line) && curAmount) {
      // After amount, this is balance — skip
      continue
    }

    // Description line
    if (curDate) {
      curDescs.push(line)
    }
  }

  flush()
  return reports
}
