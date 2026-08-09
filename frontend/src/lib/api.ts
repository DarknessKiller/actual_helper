import type { ActualBudgetReport, ProviderConfig } from './types'
import { extractPDFText } from './pdf-worker'
import { ocrPDFPages } from './ocr-worker'
import { parseTNG } from './providers/tng'
import { parseRYT } from './providers/ryt'
import { parseHLB } from './providers/hlb'
import { parseHSBCCredit } from './providers/hsbccredit'
import { parseUOBCredit } from './providers/uobcredit'
import { parseGXBank } from './providers/gxbank'
import { toActualCSVBlob } from './csv'

export type { ActualBudgetReport }

const OCR_PROVIDERS = new Set(['hsbccredit'])

const parsers: Record<string, (text: string, config: ProviderConfig) => ActualBudgetReport[]> = {
  tng: parseTNG,
  ryt: parseRYT,
  hlb: parseHLB,
  hsbccredit: parseHSBCCredit,
  uobcredit: parseUOBCredit,
  gxbank: parseGXBank,
}

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

  const parser = parsers[provider]
  if (!parser) throw new Error(`Unknown provider: ${provider}`)

  const reports = parser(text, config)
  onProgress?.({ stage: 'parse', percent: 100, message: `${reports.length} transactions found` })

  onProgress?.({ stage: 'filter', percent: 100 })
  onProgress?.({ stage: 'generate', percent: 0, message: 'Generating CSV...' })

  const blob = toActualCSVBlob(reports)
  onProgress?.({ stage: 'generate', percent: 100 })

  return blob
}
