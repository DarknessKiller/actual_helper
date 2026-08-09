import { describe, it, expect } from 'vitest'
import { formatDate, parseTNGDate, parseDayMonthYear, parseStatementDate } from '../dateutil'

describe('dateutil', () => {
  describe('formatDate', () => {
    it('converts DD MMM to YYYY-MM-DD', () => {
      const stmtDate = new Date(2026, 0, 15) // Jan 15, 2026
      expect(formatDate('1 Jan', stmtDate)).toBe('2026-01-01')
      expect(formatDate('15 Dec', stmtDate)).toBe('2025-12-15')
    })
  })

  describe('parseTNGDate', () => {
    it('parses D/M/YYYY', () => {
      const d = parseTNGDate('1/5/2026')
      expect(d).not.toBeNull()
      expect(d!.getFullYear()).toBe(2026)
      expect(d!.getMonth()).toBe(4) // May = 4 (D/M format: 1 May 2026)
      expect(d!.getDate()).toBe(1)
    })

    it('parses DD/MM/YYYY', () => {
      const d = parseTNGDate('01/12/2026')
      expect(d).not.toBeNull()
      expect(d!.getMonth()).toBe(11) // Dec = 11
      expect(d!.getDate()).toBe(1)
    })
  })

  describe('parseDayMonthYear', () => {
    it('parses D January 2006', () => {
      const d = parseDayMonthYear('1 January 2026')
      expect(d).not.toBeNull()
      expect(d!.getFullYear()).toBe(2026)
      expect(d!.getMonth()).toBe(0)
      expect(d!.getDate()).toBe(1)
    })

    it('parses D Jan 2006', () => {
      const d = parseDayMonthYear('1 Jan 2026')
      expect(d).not.toBeNull()
      expect(d!.getFullYear()).toBe(2026)
    })
  })

  describe('parseStatementDate', () => {
    it('parses 02 Jan 2006', () => {
      const d = parseStatementDate('02 Jan 2026')
      expect(d).not.toBeNull()
      expect(d!.getFullYear()).toBe(2026)
      expect(d!.getMonth()).toBe(0)
      expect(d!.getDate()).toBe(2)
    })

    it('parses 02 Jan 06', () => {
      const d = parseStatementDate('02 Jan 06')
      expect(d).not.toBeNull()
      expect(d!.getFullYear()).toBe(2006)
    })
  })
})
