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

  onMount(async () => {
    initTheme();
    try {
      const res = await fetch("/version");
      const data = await res.json();
      version = data.version;
    } catch {
      version = "";
    }
  });

  function handleConversionComplete(newHistory) {
    conversions = newHistory;
    lastConversion = conversions[0];
  }
</script>

<div class="app-bg min-h-screen flex flex-col">
  <nav class="glass sticky top-0 z-20">
    <div class="max-w-6xl 2xl:max-w-7xl mx-auto flex items-center justify-between px-5 h-14">
      <h1 class="font-semibold tracking-tight">Actual Helper</h1>
      <div class="flex items-center gap-3">
        {#if version}
          <span class="text-xs copy-dim">{version}</span>
        {/if}
        <button
          class="theme-toggle h-11 w-11 px-0 btn-ghost-apple"
          aria-label="Theme: {theme}"
          onclick={() => {
            const order = ["light", "dark", "amoled"];
            theme = setTheme(order[(order.indexOf(theme) + 1) % order.length]);
          }}
        >
          {#if theme === "light"}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-7 w-7" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2" /><path d="M12 20v2" />
              <path d="m4.93 4.93 1.41 1.41" /><path d="m17.66 17.66 1.41 1.41" />
              <path d="M2 12h2" /><path d="M20 12h2" />
              <path d="m6.34 17.66-1.41 1.41" /><path d="m19.07 4.93-1.41 1.41" />
            </svg>
          {:else if theme === "dark"}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-7 w-7" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401" />
            </svg>
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-7 w-7" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M18 5h4" /><path d="M20 3v4" />
              <path d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401" />
            </svg>
          {/if}
        </button>
      </div>
    </div>
  </nav>

  <main class="max-w-6xl 2xl:max-w-7xl mx-auto w-full px-5 py-8 flex-1" in:fade={{ duration: 300 }}>
    <div class="grid gap-5 md:grid-cols-5 md:items-start">
      <div class="flex flex-col gap-5 min-w-0 md:col-span-2">
        <ConfigPanel />
        <UploadForm onConversionComplete={handleConversionComplete} />
      </div>

      <div class="flex flex-col gap-5 min-w-0 md:col-span-3">
        {#if lastConversion}
          <ResultPanel
            filename={lastConversion.filename}
            provider={lastConversion.provider}
          />
        {/if}

        {#if conversions.length > 0}
          <HistoryDashboard bind:conversions />
        {:else}
          <section class="sheet rounded-[var(--radius-apple-lg)] p-6">
            <h2 class="text-[15px] font-semibold tracking-tight">No conversions yet</h2>
            <ol class="mt-3 flex flex-col gap-2 text-sm copy list-decimal ps-5 marker:text-ink-3">
              <li>Pick your bank or e-wallet.</li>
              <li>Drop the CSV or PDF statement.</li>
              <li>Get an Actual Budget CSV, ready to import.</li>
            </ol>
          </section>
        {/if}
      </div>
    </div>
  </main>
  <footer class="text-center text-sm copy py-6">
    Actual Helper — open source tool for Actual Budget Malaysian users
  </footer>
</div>
