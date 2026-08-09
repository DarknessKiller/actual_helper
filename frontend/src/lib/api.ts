import { initWASM, goParse } from './wasm'
import { extractPDFText } from './pdf-worker'
import { ocrPDFPages } from './ocr-worker'
import { toActualCSVBlob } from './csv'
import type { ProviderConfig } from './types'

export type { ActualBudgetReport } from './types'

const OCR_PROVIDERS = new Set(['hsbccredit'])

export type ConversionProgress = {
  stage: 'extract' | 'parse' | 'filter' | 'generate'
  percent: number
  message?: string
}

export async function convertLocally(
  provider: string,
  file: File,
  password: string | undefined,
  config: ProviderConfig,
  onProgress?: (p: ConversionProgress) => void
): Promise<Blob> {
  onProgress?.({ stage: 'extract', percent: 0, message: 'Reading file...' })

  let text: string

  if (file.name.toLowerCase().endsWith('.csv')) {
    text = await file.text()
    onProgress?.({ stage: 'extract', percent: 100 })
  } else if (OCR_PROVIDERS.has(provider)) {
    onProgress?.({ stage: 'extract', percent: 0, message: 'OCR processing...' })
    text = await ocrPDFPages(file, undefined, (pct, _page, total) => {
      onProgress?.({ stage: 'extract', percent: pct, message: `OCR page ${total}...` })
    })
  } else {
    text = await extractPDFText(file, password, (pct) => {
      onProgress?.({ stage: 'extract', percent: pct, message: 'Extracting text...' })
    })
  }

  onProgress?.({ stage: 'parse', percent: 0, message: 'Parsing transactions...' })

  // Ensure WASM is loaded
  await initWASM()

  // Call Go WASM
  const configJSON = JSON.stringify(config)
  const result = goParse(provider, text, configJSON)

  if (!result.ok) {
    throw new Error(result.error || 'Parse failed')
  }

  onProgress?.({ stage: 'parse', percent: 100, message: `${result.count} transactions found` })

  onProgress?.({ stage: 'filter', percent: 100 })
  onProgress?.({ stage: 'generate', percent: 0, message: 'Generating CSV...' })

  // Convert WASM result to ActualBudgetReport[]
  const reports = result.reports.map((r: any) => ({
    Account: r.Account || '',
    Date: r.Date || '',
    Payee: r.Payee || '',
    Notes: r.Notes || '',
    Category_Group: r.Category_Group || '',
    Category: r.Category || '',
    Amount: r.Amount || '0.00',
    Split_Amount: r.Split_Amount || '',
    Cleared: r.Cleared || 'Cleared',
  }))

  const blob = toActualCSVBlob(reports)
  onProgress?.({ stage: 'generate', percent: 100 })

  return blob
}
