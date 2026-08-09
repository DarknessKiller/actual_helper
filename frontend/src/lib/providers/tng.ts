import type { ActualBudgetReport, ProviderConfig } from '../types'
import { createEngine } from '../rules'
import { normalizeWhitespace } from '../cardutil'
import { parseTNGDate, truncate, toLocalDate } from '../dateutil'

const creditTypes = new Set([
  'Reload', 'Receive from Wallet', 'DUITNOW_RECEIVEFROM', 'Refund',
  'GO+ Daily Earnings', 'GO+ Cash In', 'Interest Payment'
])

const referencePrefixes = ['TNGD', 'TNGQR', 'TNGOW', 'CC-', 'DD-']

function isReferenceToken(tok: string): boolean {
  if (/^\d{10,}$/.test(tok)) return true
  if (/^\d{8,}[A-Za-z]+$/.test(tok)) return true
  if (/^[A-Za-z]+\d{3,}$/.test(tok)) return true
  for (const p of referencePrefixes) {
    if (tok.startsWith(p)) return true
  }
  return false
}

function trimAtReference(desc: string): string {
  const toks = desc.split(/\s+/)
  for (let i = 0; i < toks.length; i++) {
    if (isReferenceToken(toks[i])) {
      return toks.slice(0, i).join(' ')
    }
  }
  return desc
}

export function parseTNG(text: string, config: ProviderConfig): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []

  const marker = 'TNG WALLET TRANSACTION'
  const markerIdx = text.lastIndexOf(marker)
  if (markerIdx === -1) return reports
  const content = text.slice(markerIdx)

  // Split into lines and find date lines (D/M/YYYY format)
  const lines = content.split('\n')
  const dateLineRe = /^\d{1,2}\/\d{1,2}\/\d{4}$/

  // Find all date line indices
  const dateIndices: number[] = []
  for (let i = 0; i < lines.length; i++) {
    if (dateLineRe.test(lines[i].trim())) {
      dateIndices.push(i)
    }
  }

  // Process each block between date lines
  for (let b = 0; b < dateIndices.length; b++) {
    const start = dateIndices[b]
    const end = b + 1 < dateIndices.length ? dateIndices[b + 1] : lines.length
    const block = lines.slice(start, end).map(l => l.trim()).filter(l => l)

    if (block.length < 4) continue

    // block[0] = date, block[1] = status, block[2] = txType, block[3] = reference
    const dateRaw = block[0]
    const status = block[1]
    const txType = block[2] || ''

    if (status !== 'Success') continue

    const date = parseTNGDate(dateRaw)
    if (!date) continue

    // Find amount line (first RM line, not wallet balance)
    let amount = 0
    let amountIdx = -1
    for (let j = 4; j < block.length; j++) {
      const m = block[j].match(/^RM([\d,.]+)$/)
      if (m) {
        amount = parseFloat(m[1].replace(/,/g, ''))
        amountIdx = j
        break
      }
    }
    if (amountIdx === -1) continue

    // Description: everything between reference and amount
    const descLines = block.slice(4, amountIdx)
    const description = trimAtReference(normalizeWhitespace(descLines.join(' ')))

    const isCredit = creditTypes.has(txType)

    if (engine.shouldSkip(description)) continue

    const [group, category] = engine.matchCategory(description)
    const sign = isCredit ? amount : -amount

    reports.push({
      Account: 'TNG Digital',
      Date: toLocalDate(date),
      Payee: '',
      Notes: truncate(description, 100),
      Category_Group: group,
      Category: category,
      Amount: sign.toFixed(2),
      Split_Amount: '',
      Cleared: 'Cleared'
    })
  }

  return reports
}
