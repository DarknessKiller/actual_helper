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
