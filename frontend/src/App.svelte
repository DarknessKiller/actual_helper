<script>
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import UploadForm from "$lib/components/UploadForm.svelte";
  import ResultPanel from "$lib/components/ResultPanel.svelte";
  import HistoryDashboard from "$lib/components/HistoryDashboard.svelte";
  import ConfigPanel from "$lib/components/ConfigPanel.svelte";
  import { loadHistory } from "$lib/stores/history.js";
  import { getTheme, setTheme, initTheme } from "$lib/stores/theme.js";

  let conversions = $state(loadHistory());
  let theme = $state(getTheme());
  let lastConversion = $state(null);
  let version = $state("");
  let wasmReady = $state(false);
  let wasmError = $state("");
  let mobileHistoryOpen = $state(false);

  onMount(async () => {
    initTheme();
    try {
      const verRes = await fetch("/version");
      const verData = await verRes.json();
      version = verData.version;

      const wasm = new Go();
      const { instance } = await WebAssembly.instantiateStreaming(fetch(`/actual-helper.wasm?v=${version}`), wasm.importObject);
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
    } catch (error) {
      wasmError = error.message || "Browser converter failed to load";
    }
  });

  function handleConversionComplete(result) {
    conversions = result.history;
    lastConversion = { ...conversions[0], csv: result.csv };
  }
</script>

<div class="app-bg min-h-screen flex flex-col">
  <nav class="glass sticky top-0 z-20">
    <div class="max-w-6xl mx-auto flex items-center justify-between px-5 h-14">
      <span class="font-semibold tracking-tight">Actual Helper</span>
      <div class="flex items-center gap-3">
        {#if conversions.length > 0}
          <button class="chip lg:hidden" data-selected onclick={() => (mobileHistoryOpen = !mobileHistoryOpen)}>
            History ({conversions.length})
          </button>
        {/if}
        {#if version}
          <span class="text-xs copy-dim">{version}</span>
        {/if}
        <button class="btn-ghost-apple theme-toggle h-11 w-11 px-0" aria-label="Theme: {theme}" onclick={() => {
          const order = ["light", "dark", "amoled"];
          theme = setTheme(order[(order.indexOf(theme) + 1) % order.length]);
        }}>
          {#if theme === "light"}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-8 w-8" fill="none"><path stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" d="M12 3v2m0 14v2m9-9h-2M5 12H3m13.36 6.36l-1.42-1.42M7.06 7.06L5.64 5.64m12.72 0l-1.42 1.42M7.06 16.94l-1.42 1.42M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/></svg>
          {:else if theme === "dark"}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-8 w-8" fill="none"><path stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" d="M20.354 15.354A9 9 0 0 1 8.646 3.646 9.003 9.003 0 0 0 12 21a9.003 9.003 0 0 0 8.354-5.646z"/></svg>
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-8 w-8" fill="none"><path stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" d="M17 9V7a2 2 0 0 0-2-2H9a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2v-2m-5-1.2l-1.6-2.8"/></svg>
          {/if}
        </button>
      </div>
    </div>
  </nav>

  <main class="max-w-6xl mx-auto w-full px-5 py-8 flex-1">
    {#if wasmError}
      <div class="glass rounded-2xl px-4 py-3 mb-5 flex items-center gap-3 text-danger" role="alert" in:fade={{ duration: 300 }}>
        <span class="flex-1 text-sm">{wasmError}. Refresh to retry.</span>
      </div>
    {:else if !wasmReady}
      <div class="flex items-center gap-3 py-4 copy text-sm" role="status" in:fade={{ duration: 300 }}>
        <span class="progress-apple w-40"><div style="width: 60%"></div></span>
        Loading browser converter…
      </div>
    {/if}

    <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_380px] lg:items-start">
      <!-- Primary flow: config first, then convert -->
      <div class="space-y-5 min-w-0">
        <ConfigPanel onConfigChange={() => {}} />
        <UploadForm onConversionComplete={handleConversionComplete} disabled={!wasmReady} />
        {#if lastConversion}
          <ResultPanel filename={lastConversion.filename} provider={lastConversion.provider} csv={lastConversion.csv} />
        {/if}
      </div>

      <!-- Desktop rail: history uses the extra real estate off the main path -->
      {#if conversions.length > 0}
        <aside class="hidden lg:block sticky top-20">
          <HistoryDashboard bind:conversions />
        </aside>
      {/if}
    </div>

    {#if mobileHistoryOpen}
      <div class="mt-5 lg:hidden"><HistoryDashboard bind:conversions /></div>
    {/if}
  </main>

  <footer class="text-center text-sm copy py-6">
    Actual Helper — open source tool for Actual Budget Malaysian users
  </footer>
</div>
