export function convertFile(provider: string, file: File, password?: string): Promise<Response> {
  const formData = new FormData()
  formData.append('file', file)
  if (password) {
    formData.append('password', password)
  }

  return fetch(`/convert/${provider}`, {
    method: 'POST',
    body: formData,
  })
}

// --- Provider config lifecycle ---

export async function downloadConfig(): Promise<Blob> {
  const res = await fetch('/config', { method: 'GET' })
  if (!res.ok) {
    throw new Error(await res.text().catch(() => 'Sample config unavailable'))
  }
  return res.blob()
}

export async function uploadConfig(file: File): Promise<string[]> {
  const res = await fetch('/config', {
    method: 'POST',
    body: file,
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok) {
    throw new Error(await res.text().catch(() => 'Invalid config'))
  }
  const data = await res.json()
  return (data.applied ?? []) as string[]
}

export async function unloadConfig(): Promise<boolean> {
  const res = await fetch('/config', { method: 'DELETE' })
  if (!res.ok) {
    throw new Error(await res.text().catch(() => 'Unload failed'))
  }
  const data = await res.json()
  return Boolean(data.cleared)
}
