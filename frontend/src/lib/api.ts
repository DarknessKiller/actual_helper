type ConversionResult = { csv: string }

function wasmResult(result: unknown): string {
  if (typeof result !== 'string') return (result as any)?.csv ?? ''
  if (result.startsWith('{"error')) {
    try {
      const parsed = JSON.parse(result)
      if (parsed?.error) throw new Error(parsed.error)
    } catch (e) {
      if (e instanceof Error && e.message !== result) throw e
    }
  }
  return result
}

export async function convertFile(provider: string, file: File, password = ''): Promise<ConversionResult> {
  const convert = (globalThis as any).actualHelperConvert
  if (!convert) throw new Error('Browser converter is not loaded')
  if (file.name.toLowerCase().endsWith('.csv') || file.type === 'text/csv') {
    return { csv: wasmResult(convert(provider, await file.text())) }
  }

  const pdfjs = await import('pdfjs-dist')
  pdfjs.GlobalWorkerOptions.workerSrc = new URL('pdfjs-dist/build/pdf.worker.min.mjs', import.meta.url).toString()
  const pdfDocument = await pdfjs.getDocument({ data: new Uint8Array(await file.arrayBuffer()), password }).promise
  const pageTexts: string[] = []
  for (let pageNumber = 1; pageNumber <= pdfDocument.numPages; pageNumber++) {
    const page = await pdfDocument.getPage(pageNumber)
    const content = await page.getTextContent()
    const items = content.items as any[]
    const lines: any[][] = []
    for (const item of items) {
      const y = item.transform?.[5] ?? 0
      let line = lines.find((candidate) => Math.abs((candidate[0].transform?.[5] ?? 0) - y) < 3)
      if (!line) lines.push(line = [])
      line.push(item)
    }
    pageTexts.push(
      lines
        .sort((a, b) => (b[0].transform?.[5] ?? 0) - (a[0].transform?.[5] ?? 0))
        .map((line) => line.sort((a, b) => (a.transform?.[4] ?? 0) - (b.transform?.[4] ?? 0)).map((item) => item.str).join(' '))
        .join('\n')
    )
  }
  const text = pageTexts.join('\n') + '\n'
  const parsePDF = (globalThis as any).actualHelperParsePDFText
  if (!parsePDF) throw new Error('Browser PDF parser is not loaded')
  try {
    return { csv: wasmResult(parsePDF(provider, text)) }
  } catch (digitalError) {
    const { createWorker } = await import('tesseract.js')
    const worker = await createWorker('eng+msa')
    const ocrTexts: string[] = []
    try {
      for (let pageNumber = 1; pageNumber <= pdfDocument.numPages; pageNumber++) {
        const page = await pdfDocument.getPage(pageNumber)
        const viewport = page.getViewport({ scale: 2 })
        const canvas = window.document.createElement('canvas')
        canvas.width = viewport.width
        canvas.height = viewport.height
        await page.render({ canvas, canvasContext: canvas.getContext('2d')!, viewport }).promise
        ocrTexts.push((await worker.recognize(canvas)).data.text)
      }
    } finally {
      await worker.terminate()
    }
    const ocrText = ocrTexts.join('\n') + '\n'
    if (!ocrText.trim()) throw digitalError
    return { csv: wasmResult(parsePDF(provider, ocrText)) }
  }
}
