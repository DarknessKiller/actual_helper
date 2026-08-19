interface Conversion {
  id: string
  provider: string
  filename: string
  timestamp: string
  success: boolean
  csv?: string
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

function saveHistory(conversions: Conversion[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(conversions.map(({ csv, ...conversion }) => conversion)))
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

