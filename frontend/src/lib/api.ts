type ConversionResult = { csv: string }

// Render a row of pdf text items/words as pdftotext -layout would: place each
// token at its horizontal column so multi-space column gaps are preserved. The
// Go parsers' regexes rely on `\s{2,}` column separators, which a naive
// single-space join destroys. `xOf` reads the token's left edge and `wOf` its
// width (pdf.js transform vs OCR bbox).
function renderLayout(
  tokens: any[],
  xOf: (t: any) => number,
  wOf: (t: any) => number,
): string {
  tokens.sort((a, b) => xOf(a) - xOf(b))
  let charW = 0
  let n = 0
  for (const t of tokens) {
    const s = (t.str ?? t.text ?? '').trim()
    const w = wOf(t)
    if (s && w > 0) {
      charW += w / (s.length || 1)
      n++
    }
  }
  charW = n ? charW / n : 0
  if (charW <= 0) return tokens.map((t) => (t.str ?? t.text ?? '').trim()).filter(Boolean).join(' ')

  let out = ''
  let first = true
  let prevRight = 0
  for (const t of tokens) {
    const s = (t.str ?? t.text ?? '').trim()
    if (!s) continue
    const x = xOf(t)
    if (first) {
      out += ' '.repeat(Math.max(0, Math.round(x / charW)))
      first = false
    } else {
      out += ' '.repeat(Math.max(1, Math.round((x - prevRight) / charW)))
    }
    out += s
    prevRight = x + (wOf(t) || s.length * charW)
  }
  return out
}

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
        .map((line) =>
          renderLayout(
            line,
            (it) => it.transform?.[4] ?? 0,
            (it) => it.width ?? 0,
          ),
        )
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
        const { data } = await worker.recognize(canvas, {}, { blocks: true })
        // Rebuild layout (multi-space columns) from Tesseract word boxes, the
        // same way the digital path rebuilds pdf.js layout. The `blocks` output
        // is required or data.blocks is null and reconstruction yields nothing.
        ocrTexts.push(
          (data.blocks ?? [])
            .flatMap((b) => b.paragraphs ?? [])
            .flatMap((p) => p.lines ?? [])
            .map((line) =>
              renderLayout(
                line.words ?? [],
                (w) => w.bbox?.x0 ?? 0,
                (w) => (w.bbox?.x1 ?? 0) - (w.bbox?.x0 ?? 0),
              ),
            )
            .join('\n')
        )
      }
    } finally {
      await worker.terminate()
    }
    const ocrText = ocrTexts.join('\n') + '\n'
    if (!ocrText.trim()) throw digitalError
    return { csv: wasmResult(parsePDF(provider, ocrText)) }
  }
}
