// Mirrors internal/models/actual_report.go
export interface ActualBudgetReport {
  Account: string
  Date: string
  Payee: string
  Notes: string
  Category_Group: string
  Category: string
  Amount: string
  Split_Amount: string
  Cleared: string
}

// Mirrors internal/models/rule.go
export interface CategoryRule {
  keyword: string
  group: string
  category: string
}

// Mirrors internal/config/config.go ProviderConfig
export interface ProviderConfig {
  exclude_keywords: string[]
  include_keywords: string[]
  categories: CategoryRule[]
  account_mappings: Record<string, string>
}

export interface AppConfig {
  global: ProviderConfig
  providers: Record<string, ProviderConfig>
}

// Parse result from any provider
export interface ParseResult {
  reports: ActualBudgetReport[]
  errors: string[]
}

// Progress callback for workers
export type ProgressCallback = (progress: { stage: string; percent: number; message?: string }) => void

// Extraction method for PDFs
export type ExtractionMethod = 'digital' | 'ocr'
