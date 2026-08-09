import type { ActualBudgetReport, ProviderConfig } from '../types'
import { createEngine } from '../rules'
import { normalizeWhitespace, extractAfterMarker } from '../cardutil'
import { formatDate } from '../dateutil'

function detectFormat(text: string): 'credit' | 'debit' {
  if (/Credit Card Number|HLB Credit Card|Tarikh Penyata/i.test(text)) return 'credit'
  return 'debit'
}

function isSkipLine(line: string): boolean {
  const s = line.toLowerCase()
  return s.includes('previous balance from last statement') ||
    s.includes('new transaction / charges') ||
    s.includes('sub total') ||
    s.includes('total balance') ||
    s.includes('payment received') ||
    s.includes('current outstanding balance')
}

function parseCredit(text: string, config: ProviderConfig): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []

  const dateMatch = text.match(/(?:Tarikh Penyata|Statement Date)\s+(\d{2}\s+\w{3}\s+\d{4})/)
  if (!dateMatch) return reports
  const stmtDate = new Date(dateMatch[1])
  if (isNaN(stmtDate.getTime())) return reports

  const account = extractAfterMarker(text, 'Credit Card Number', 'HLB Credit Card')

  const txRe = /^\s*(\d{2}\s+\w{3})\s+(\d{2}\s+\w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?$/gm
  let match: RegExpExecArray | null

  while ((match = txRe.exec(text)) !== null) {
    const postDate = formatDate(match[1], stmtDate)
    const transDate = formatDate(match[2], stmtDate)
    const description = normalizeWhitespace(match[3])
    const amount = parseFloat(match[4].replace(/,/g, ''))
    const isCredit = match[5] === 'CR'

    if (!postDate || !transDate || isNaN(amount)) continue
    if (isSkipLine(description)) continue
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

function extractLayoutAmounts(line: string, depositCol: number, withdrawalCol: number): { deposit: string; withdrawal: string } {
  const deposit = depositCol >= 0 ? line.slice(depositCol).split(/\s{2,}/)[0]?.trim() || '' : ''
  const withdrawal = withdrawalCol >= 0 ? line.slice(withdrawalCol).split(/\s{2,}/)[0]?.trim() || '' : ''
  return { deposit, withdrawal }
}

function parseDebitLayout(text: string, config: ProviderConfig, account: string): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []

  const lines = text.split('\n')
  let headerIdx = -1
  let depositCol = -1
  let withdrawalCol = -1

  for (let i = 0; i < lines.length; i++) {
    if (/Deposit/.test(lines[i]) && /Withdrawal/.test(lines[i])) {
      headerIdx = i
      depositCol = lines[i].indexOf('Deposit')
      withdrawalCol = lines[i].indexOf('Withdrawal')
      break
    }
  }
  if (headerIdx === -1) return reports

  let curDate = ''
  let curDesc: string[] = []
  let curDeposit = ''
  let curWithdrawal = ''

  const flush = () => {
    if (!curDate || (!curDeposit && !curWithdrawal)) return
    const isCredit = curDeposit !== ''
    const amountStr = isCredit ? curDeposit : curWithdrawal
    const amount = parseFloat(amountStr.replace(/,/g, ''))
    if (isNaN(amount)) return

    const description = normalizeWhitespace(curDesc.join(' '))
    if (/Balance from previous statement/i.test(description)) return
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

  for (let i = headerIdx + 1; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()

    if (/Total Withdrawals|Closing Balance|Balance brought forward|Opening Balance/i.test(trimmed)) {
      flush()
      break
    }

    const dateMatch = trimmed.match(/^(\d{2}-\d{2}-\d{4})/)
    if (dateMatch) {
      flush()
      const parts = dateMatch[1].split('-')
      curDate = `${parts[2]}-${parts[1]}-${parts[0]}`
      curDesc = []
      const amounts = extractLayoutAmounts(line, depositCol, withdrawalCol)
      curDeposit = amounts.deposit
      curWithdrawal = amounts.withdrawal
      // Rest of the line after date is description
      const afterDate = trimmed.slice(dateMatch[0].length).trim()
      if (afterDate) curDesc.push(afterDate)
      continue
    }

    if (!trimmed) {
      // Count blank lines, flush after 3
      continue
    }

    // Non-date, non-blank line = description continuation or amount
    const amounts = extractLayoutAmounts(line, depositCol, withdrawalCol)
    if (amounts.deposit || amounts.withdrawal) {
      if (amounts.deposit) curDeposit = amounts.deposit
      if (amounts.withdrawal) curWithdrawal = amounts.withdrawal
    } else {
      curDesc.push(trimmed)
    }
  }

  flush()
  return reports
}

function parseDebitColumnar(text: string, config: ProviderConfig, account: string): ActualBudgetReport[] {
  const engine = createEngine(config)
  const reports: ActualBudgetReport[] = []
  const lines = text.split('\n')

  let curDate = ''
  let descriptions: string[] = []
  let amounts: string[] = []

  const flush = () => {
    if (!curDate || amounts.length === 0) return
    for (let k = 0; k < amounts.length; k++) {
      const desc = normalizeWhitespace(descriptions[k] || '')
      const amt = parseFloat(amounts[k].replace(/,/g, ''))
      if (isNaN(amt)) continue
      if (/Balance from previous statement|opening balance/i.test(desc)) continue
      if (engine.shouldSkip(desc)) continue

      const isCredit = desc.toLowerCase() === 'deposit'
      const [group, category] = engine.matchCategory(desc)
      const sign = isCredit ? amt : -amt

      reports.push({
        Account: account,
        Date: curDate,
        Payee: '',
        Notes: desc,
        Category_Group: group,
        Category: category,
        Amount: sign.toFixed(2),
        Split_Amount: '',
        Cleared: 'Cleared'
      })
    }
  }

  const dateRe = /^(\d{2}-\d{2}-\d{4})$/

  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) continue
    if (/^Total|Date.*Tarikh|Deposit.*Withdrawal/i.test(trimmed)) continue
    if (/Balance brought forward|Opening Balance|Closing Balance/i.test(trimmed)) {
      flush()
      curDate = ''
      descriptions = []
      amounts = []
      continue
    }

    const dm = trimmed.match(dateRe)
    if (dm) {
      flush()
      const parts = dm[1].split('-')
      curDate = `${parts[2]}-${parts[1]}-${parts[0]}`
      descriptions = []
      amounts = []
      continue
    }

    const amtMatch = trimmed.match(/^([\d,.]+)$/)
    if (amtMatch) {
      amounts.push(amtMatch[1])
    } else {
      const desc = trimmed.replace(/\s{2,}/g, ' ')
      if (desc.toLowerCase() === 'deposit' || descriptions.length > amounts.length) {
        descriptions.push(desc)
      } else {
        descriptions.push(desc)
      }
    }
  }
  flush()

  return reports
}

function parseDebit(text: string, config: ProviderConfig): ActualBudgetReport[] {
  const acctMatch = text.match(/(?:A\/C No|No Akaun)[\s:]+([^\n]+)/i)
  const account = acctMatch ? acctMatch[1].trim() : 'HLB Debit Account'

  if (/Deposit/.test(text) && /Withdrawal/.test(text)) {
    return parseDebitLayout(text, config, account)
  }
  return parseDebitColumnar(text, config, account)
}

export function parseHLB(text: string, config: ProviderConfig): ActualBudgetReport[] {
  return detectFormat(text) === 'credit'
    ? parseCredit(text, config)
    : parseDebit(text, config)
}
