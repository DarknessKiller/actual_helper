import { describe, it, expect } from 'vitest'
import { createEngine, mergeConfigs } from '../rules'
import type { ProviderConfig } from '../types'

describe('rule engine', () => {
  const config: ProviderConfig = {
    exclude_keywords: ['noise1', 'noise2'],
    include_keywords: [],
    categories: [
      { keyword: 'shopee', group: 'Shopping', category: 'Online' },
      { keyword: 'grab', group: 'Food & Dining', category: 'Delivery' }
    ],
    account_mappings: {}
  }

  describe('shouldSkip', () => {
    it('skips matching exclude keywords', () => {
      const engine = createEngine(config)
      expect(engine.shouldSkip('noise1 transaction')).toBe(true)
      expect(engine.shouldSkip('something with noise2')).toBe(true)
    })

    it('does not skip non-matching', () => {
      const engine = createEngine(config)
      expect(engine.shouldSkip('normal transaction')).toBe(false)
    })

    it('include keywords act as whitelist', () => {
      const withInclude: ProviderConfig = {
        ...config,
        include_keywords: ['important']
      }
      const engine = createEngine(withInclude)
      expect(engine.shouldSkip('important transaction')).toBe(false)
      expect(engine.shouldSkip('normal transaction')).toBe(true)
    })
  })

  describe('matchCategory', () => {
    it('returns first matching category', () => {
      const engine = createEngine(config)
      const [group, category] = engine.matchCategory('Shopee purchase')
      expect(group).toBe('Shopping')
      expect(category).toBe('Online')
    })

    it('returns empty for no match', () => {
      const engine = createEngine(config)
      const [group, category] = engine.matchCategory('random transaction')
      expect(group).toBe('')
      expect(category).toBe('')
    })

    it('is case-insensitive', () => {
      const engine = createEngine(config)
      const [group, category] = engine.matchCategory('SHOPEE ORDER')
      expect(group).toBe('Shopping')
      expect(category).toBe('Online')
    })
  })
})

describe('mergeConfigs', () => {
  it('concatenates arrays and provider overrides global mappings', () => {
    const global: ProviderConfig = {
      exclude_keywords: ['g1'],
      include_keywords: [],
      categories: [{ keyword: 'g', group: 'G', category: 'G' }],
      account_mappings: { a: 'global' }
    }
    const provider: ProviderConfig = {
      exclude_keywords: ['p1'],
      include_keywords: [],
      categories: [{ keyword: 'p', group: 'P', category: 'P' }],
      account_mappings: { a: 'provider', b: 'provider' }
    }
    const merged = mergeConfigs(global, provider)
    expect(merged.exclude_keywords).toEqual(['g1', 'p1'])
    expect(merged.categories).toHaveLength(2)
    expect(merged.account_mappings.a).toBe('provider')
    expect(merged.account_mappings.b).toBe('provider')
  })
})
