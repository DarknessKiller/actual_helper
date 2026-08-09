import { createWorker, type Worker } from 'tesseract.js'
import * as pdfjsLib from 'pdfjs-dist'

let cachedWorker: Worker | null = null

export async function initOCR(
  langs = 'eng+msa',
  onProgress?: (pct: number, stage: string) => void
): Promise<void> {
  if (cachedWorker) return
  onProgress?.(0, 'Initializing OCR engine...')
  cachedWorker = await createWorker(langs, 1, {
    logger: (m: any) => {
      if (m.status === 'recognizing text' && typeof m.progress === 'number') {
        onProgress?.(Math.round(m.progress * 100), 'Recognizing text...')
      }
    }
  })
  onProgress?.(100, 'OCR ready')
}

export async function ocrImage(
  imageData: ImageData | HTMLCanvasElement | string,
  onProgress?: (pct: number) => void
): Promise<string> {
  await initOCR()
  if (!cachedWorker) throw new Error('OCR not initialized')
  const result = await cachedWorker.recognize(imageData as any)
  onProgress?.(100)
  return result.data.text
}

export async function ocrPDFPages(
  file: File,
  _password?: string,
  onProgress?: (pct: number, page: number, total: number) => void
): Promise<string> {
  await initOCR()
  if (!cachedWorker) throw new Error('OCR not initialized')

  const data = await file.arrayBuffer()
  const pdf = await pdfjsLib.getDocument({ data }).promise
  const totalPages = pdf.numPages
  const texts: string[] = []

  for (let i = 1; i <= totalPages; i++) {
    const page = await pdf.getPage(i)
    const viewport = page.getViewport({ scale: 2 })
    const canvas = document.createElement('canvas')
    canvas.width = viewport.width
    canvas.height = viewport.height
    const ctx = canvas.getContext('2d')!
    await page.render({ canvasContext: ctx, viewport }).promise

    const result = await cachedWorker.recognize(canvas)
    texts.push(result.data.text)
    onProgress?.(Math.round((i / totalPages) * 100), i, totalPages)
  }

  return texts.join('\n\n')
}

export async function terminateOCR(): Promise<void> {
  if (cachedWorker) {
    await cachedWorker.terminate()
    cachedWorker = null
  }
}
