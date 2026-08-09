interface Conversion {
  id: string
  provider: string
  filename: string
  timestamp: string
  success: boolean
}

const STORAGE_KEY = 'actual-helper-conversions'

export function loadHistory(): Conversion[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

export function saveHistory(conversions: Conversion[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(conversions))
  } catch {
  }
}

export function addConversion(conversion: Conversion): Conversion[] {
  const history = loadHistory()
  history.unshift(conversion)
  saveHistory(history)
  return history
}

export function clearHistory(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
  }
}

export type { Conversion }
