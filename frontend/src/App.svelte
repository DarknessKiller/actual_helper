<script>
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import UploadForm from "$lib/components/UploadForm.svelte";
  import ResultPanel from "$lib/components/ResultPanel.svelte";
  import HistoryDashboard from "$lib/components/HistoryDashboard.svelte";
  import ConfigPanel from "$lib/components/ConfigPanel.svelte";
  import { loadHistory } from "$lib/stores/history.js";

  let conversions = $state(loadHistory());
  let lastConversion = $state(null);
  let version = $state("");
  let wasmReady = $state(false);
  let wasmError = $state("");

  onMount(async () => {
    try {
      const wasm = new Go();
      const { instance } = await WebAssembly.instantiateStreaming(fetch("/actual-helper.wasm"), wasm.importObject);
      wasm.run(instance);
      await new Promise((resolve, reject) => {
        const started = Date.now();
        const wait = () => {
          if (globalThis.actualHelperConvert) return resolve();
          if (Date.now() - started > 10000) return reject(new Error("Browser converter failed to start"));
          setTimeout(wait, 25);
        };
        wait();
      });
      wasmReady = true;
      const res = await fetch("/version");
      const data = await res.json();
      version = data.version;
    } catch (error) {
      wasmError = error.message || "Browser converter failed to load";
    }
  });

  function handleConversionComplete(result) {
    conversions = result.history;
    lastConversion = { ...conversions[0], csv: result.csv };
  }
</script>

<div class="min-h-screen bg-base-200">
  <div class="navbar bg-base-100 border-b border-base-200">
    <div class="flex-1">
      <span class="text-xl font-semibold px-4">Actual Helper</span>
    </div>
    <div class="flex-none pr-4">
      {#if version}
        <span class="badge badge-soft badge-accent">{version}</span>
      {/if}
    </div>
  </div>

  <main class="max-w-2xl mx-auto px-4 py-6 sm:py-10" in:fade={{ duration: 300 }}>
    {#if wasmError}
      <div role="alert" class="alert alert-error mb-4">{wasmError}. Refresh to retry.</div>
    {:else if !wasmReady}
      <div role="status" class="alert mb-4">Loading browser converter…</div>
    {/if}

    <ConfigPanel />
    <UploadForm onConversionComplete={handleConversionComplete} disabled={!wasmReady} />

    {#if lastConversion}
      <ResultPanel filename={lastConversion.filename} provider={lastConversion.provider} csv={lastConversion.csv} />
    {/if}

    <HistoryDashboard bind:conversions />
  </main>

  <footer class="text-center text-xs text-base-content/30 py-6">
    Actual Helper — A Open source tool for Actual Budget Malaysian users
  </footer>
</div>
