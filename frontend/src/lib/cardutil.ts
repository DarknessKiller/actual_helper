export const whitespaceRe = /\s+/g
export const cardNumberRe = /(\d{4}[\s-]*\d{4}[\s-]*\d{4}[\s-]*\d{4})/

export function normalizeWhitespace(s: string): string {
  return s.replace(whitespaceRe, ' ').trim()
}

export function extractAfterMarker(text: string, marker: string, fallback: string): string {
  const idx = text.indexOf(marker)
  if (idx === -1) return fallback

  const after = text.slice(idx + marker.length).replace(/\n/g, ' ').replace(/-/g, ' ')
  const m = after.match(cardNumberRe)
  return m ? m[1] : fallback
}

export function extractNearCardType(text: string, cardTypes: string[], fallback: string): string {
  for (const ct of cardTypes) {
    const idx = text.indexOf(ct)
    if (idx === -1) continue
    const start = Math.max(0, idx - 50)
    const end = Math.min(text.length, idx + 200)
    const area = text.slice(start, end)
    const m = area.match(cardNumberRe)
    if (m) return m[1].replace(/-/g, ' ')
  }
  return fallback
}

export function applyMapping(mapping: Record<string, string> | undefined, name: string): string {
  if (!mapping) return name
  return mapping[name] ?? name
}
