import * as pdfjsLib from 'pdfjs-dist'

pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url
).href

export async function extractPDFText(
  file: File,
  password?: string,
  onProgress?: (pct: number) => void
): Promise<string> {
  const data = await file.arrayBuffer()
  const loadingTask = pdfjsLib.getDocument({ data, password: password || undefined })
  const pdf = await loadingTask.promise
  const totalPages = pdf.numPages
  const texts: string[] = []

  for (let i = 1; i <= totalPages; i++) {
    const page = await pdf.getPage(i)
    const content = await page.getTextContent()
    const pageText = content.items
      .map((item: any) => item.str)
      .join('\n')
    texts.push(pageText)
    onProgress?.(Math.round((i / totalPages) * 100))
  }

  return texts.join('\n\n')
}

export async function extractPDFPageCount(
  file: File,
  password?: string
): Promise<number> {
  const data = await file.arrayBuffer()
  const loadingTask = pdfjsLib.getDocument({ data, password: password || undefined })
  const pdf = await loadingTask.promise
  return pdf.numPages
}
