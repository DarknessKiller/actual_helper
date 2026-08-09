import type { CategoryRule, ProviderConfig } from './types'

export function createEngine(config: ProviderConfig) {
  const excludeKeywords = (config.exclude_keywords || []).map(k => k.toLowerCase())
  const includeKeywords = (config.include_keywords || []).map(k => k.toLowerCase())
  const categories = config.categories || []

  return {
    shouldSkip(description: string): boolean {
      const lower = description.toLowerCase()

      if (includeKeywords.length > 0) {
        for (const kw of includeKeywords) {
          if (lower.includes(kw)) return false
        }
        return true
      }

      for (const kw of excludeKeywords) {
        if (lower.includes(kw)) return true
      }
      return false
    },

    matchCategory(description: string): [string, string] {
      const lower = description.toLowerCase()
      for (const r of categories) {
        if (lower.includes(r.keyword.toLowerCase())) {
          return [r.group, r.category]
        }
      }
      return ['', '']
    }
  }
}

export function mergeConfigs(global: ProviderConfig, provider: ProviderConfig): ProviderConfig {
  return {
    exclude_keywords: [...(global.exclude_keywords || []), ...(provider.exclude_keywords || [])],
    include_keywords: [...(global.include_keywords || []), ...(provider.include_keywords || [])],
    categories: [...(global.categories || []), ...(provider.categories || [])],
    account_mappings: { ...(global.account_mappings || {}), ...(provider.account_mappings || {}) }
  }
}
