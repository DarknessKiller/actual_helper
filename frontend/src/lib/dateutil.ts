const MONTHS_3: Record<string, number> = {
  JAN: 0, FEB: 1, MAR: 2, APR: 3, MAY: 4, JUN: 5,
  JUL: 6, AUG: 7, SEP: 8, OCT: 9, NOV: 10, DEC: 11
}

const MONTHS_FULL: Record<string, number> = {
  JANUARY: 0, FEBRUARY: 1, MARCH: 2, APRIL: 3, MAY: 4, JUNE: 5,
  JULY: 6, AUGUST: 7, SEPTEMBER: 8, OCTOBER: 9, NOVEMBER: 10, DECEMBER: 11
}

function pad2(n: number): string { return n < 10 ? '0' + n : '' + n }

function fmtDate(y: number, m: number, d: number): string {
  return y + '-' + pad2(m + 1) + '-' + pad2(d)
}

/** "DD MMM" -> "YYYY-MM-DD", inferring year from stmtDate */
export function formatDate(ddmmm: string, stmtDate: Date): string | null {
  const parts = ddmmm.trim().split(/\s+/)
  if (parts.length !== 2) return null
  const day = parseInt(parts[0], 10)
  if (isNaN(day)) return null
  const monthKey = parts[1].toUpperCase()
  const monthNum = MONTHS_3[monthKey]
  if (monthNum === undefined) return null

  let year = stmtDate.getFullYear()
  if (monthNum > stmtDate.getMonth()) year--

  return fmtDate(year, monthNum, day)
}

/** "D/M/YYYY" or "DD/MM/YYYY" (Malaysian format) -> Date */
export function parseTNGDate(raw: string): Date | null {
  const m = raw.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/)
  if (!m) return null
  const day = parseInt(m[1], 10)
  const month = parseInt(m[2], 10) - 1
  const year = parseInt(m[3], 10)
  return new Date(year, month, day)
}

/** "D January 2006" or "D Jan 2006" -> Date */
export function parseDayMonthYear(raw: string): Date | null {
  const parts = raw.trim().split(/\s+/)
  if (parts.length < 3) return null
  const day = parseInt(parts[0], 10)
  if (isNaN(day)) return null
  const monthKey = parts[1].toUpperCase()
  const monthNum = MONTHS_3[monthKey] ?? MONTHS_FULL[monthKey]
  if (monthNum === undefined) return null
  const year = parseInt(parts[2], 10)
  if (isNaN(year)) return null
  return new Date(year, monthNum, day)
}

/** "02 Jan 2006" or "02 Jan 06" -> Date */
export function parseStatementDate(raw: string): Date | null {
  const parts = raw.trim().split(/\s+/)
  if (parts.length !== 3) return null
  const day = parseInt(parts[0], 10)
  if (isNaN(day)) return null
  const monthKey = parts[1].toUpperCase()
  const monthNum = MONTHS_3[monthKey]
  if (monthNum === undefined) return null
  let year = parseInt(parts[2], 10)
  if (isNaN(year)) return null
  if (year < 100) year += 2000
  return new Date(year, monthNum, day)
}

export function truncate(s: string, n: number): string {
  return s.length <= n ? s : s.slice(0, n) + '...'
}

/** Format Date as YYYY-MM-DD in local timezone (not UTC) */
export function toLocalDate(d: Date): string {
  const y = d.getFullYear()
  const m = d.getMonth() + 1
  const day = d.getDate()
  return y + '-' + pad2(m) + '-' + pad2(day)
}
