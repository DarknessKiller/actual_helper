export type ConversionResult = { csv: string }

function wasmResult(result: unknown): string {
  if (typeof result !== 'string') return (result as any)?.csv ?? ''
  let parsed: any
  try {
    parsed = JSON.parse(result)
  } catch {
    return result
  }
  if (parsed?.error) throw new Error(parsed.error)
  return parsed?.csv ?? result
}

export async function convertFile(provider: string, file: File, password = ''): Promise<ConversionResult> {
  const convert = (globalThis as any).actualHelperConvert
  if (!convert) throw new Error('Browser converter is not loaded')
  if (file.name.toLowerCase().endsWith('.csv') || file.type === 'text/csv') {
    return { csv: wasmResult(convert(provider, await file.text())) }
  }

  const pdfjs = await import('pdfjs-dist')
  const pdfDocument = await pdfjs.getDocument({ data: new Uint8Array(await file.arrayBuffer()), password }).promise
  let text = ''
  for (let pageNumber = 1; pageNumber <= pdfDocument.numPages; pageNumber++) {
    const page = await pdfDocument.getPage(pageNumber)
    const content = await page.getTextContent()
    text += content.items.map((item: any) => item.str).join(' ') + '\n'
  }
  if (!text.trim()) {
    const { createWorker } = await import('tesseract.js')
    const worker = await createWorker('eng')
    try {
      for (let pageNumber = 1; pageNumber <= pdfDocument.numPages; pageNumber++) {
        const page = await pdfDocument.getPage(pageNumber)
        const viewport = page.getViewport({ scale: 2 })
        const canvas = window.document.createElement('canvas')
        canvas.width = viewport.width
        canvas.height = viewport.height
        await page.render({ canvas, canvasContext: canvas.getContext('2d')!, viewport }).promise
        text += (await worker.recognize(canvas)).data.text + '\n'
      }
    } finally {
      await worker.terminate()
    }
  }
  if (!text.trim()) throw new Error('No text found in PDF')
  const parsePDF = (globalThis as any).actualHelperParsePDFText
  if (!parsePDF) throw new Error('Browser PDF parser is not loaded')
  return { csv: wasmResult(parsePDF(provider, text)) }
}
