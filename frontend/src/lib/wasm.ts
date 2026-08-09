let wasmReady = false

async function loadWasmExec(): Promise<void> {
  if ((globalThis as any).Go) return
  await new Promise<void>((resolve, reject) => {
    const script = document.createElement('script')
    script.src = './wasm_exec.js'
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('Failed to load wasm_exec.js'))
    document.head.appendChild(script)
  })
}

export async function initWASM(): Promise<void> {
  if (wasmReady) return

  await loadWasmExec()

  const go = new (globalThis as any).Go()
  const result = await WebAssembly.instantiateStreaming(
    fetch('./main.wasm'),
    go.importObject
  )
  go.run(result.instance)
  wasmReady = true
}

export function isWASMReady(): boolean {
  return wasmReady
}

export function goParse(provider: string, text: string, configJSON: string): any {
  if (!wasmReady) throw new Error('WASM not initialized')
  return (globalThis as any).goParse(provider, text, configJSON)
}

export function goProviders(): string[] {
  if (!wasmReady) throw new Error('WASM not initialized')
  return (globalThis as any).goProviders()
}
