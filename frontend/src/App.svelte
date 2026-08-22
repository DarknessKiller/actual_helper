<script>
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import UploadForm from "$lib/components/UploadForm.svelte";
  import ResultPanel from "$lib/components/ResultPanel.svelte";
  import HistoryDashboard from "$lib/components/HistoryDashboard.svelte";
  import ConfigPanel from "$lib/components/ConfigPanel.svelte";
  import { loadHistory } from "$lib/stores/history.js";
  import { getTheme, setTheme } from "$lib/stores/theme.js";

  let conversions = $state(loadHistory());
  let lastConversion = $state(null);
  let version = $state("");
  let theme = $state(getTheme());

  onMount(async () => {
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

<div class="min-h-screen bg-base-200">
  <div class="navbar bg-base-100 border-b border-base-200">
    <div class="flex-1">
      <span class="text-xl font-semibold px-4">Actual Helper</span>
    </div>
    <div class="flex-none pr-4 flex items-center gap-3">
      <button
        class="btn btn-ghost btn-circle btn-sm"
        aria-label="Theme: {theme}"
        onclick={() => {
          const order = ["light", "dark", "amoled"];
          theme = setTheme(order[(order.indexOf(theme) + 1) % order.length]);
        }}
      >
        {#if theme === "light"}
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-5 w-5" fill="none"><path stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" d="M12 3v2m0 14v2m9-9h-2M5 12H3m13.36 6.36l-1.42-1.42M7.06 7.06L5.64 5.64m12.72 0l-1.42 1.42M7.06 16.94l-1.42 1.42M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/></svg>
        {:else if theme === "dark"}
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-5 w-5" fill="none"><path stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" d="M20.354 15.354A9 9 0 0 1 8.646 3.646 9.003 9.003 0 0 0 12 21a9.003 9.003 0 0 0 8.354-5.646z"/></svg>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" class="h-5 w-5" fill="none"><path stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" d="M17 9V7a2 2 0 0 0-2-2H9a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2v-2m-5-1.2l-1.6-2.8"/></svg>
        {/if}
      </button>
      {#if version}
        <span class="badge badge-soft badge-accent">v{version}</span>
      {/if}
    </div>
  </div>

  <main
    class="max-w-2xl mx-auto px-4 py-6 sm:py-10"
    in:fade={{ duration: 300 }}
  >
    <ConfigPanel />
    <UploadForm onConversionComplete={handleConversionComplete} />

    {#if lastConversion}
      <ResultPanel
        filename={lastConversion.filename}
        provider={lastConversion.provider}
      />
    {/if}

    <HistoryDashboard bind:conversions />
  </main>

  <footer class="text-center text-xs text-base-content/30 py-6">
    Actual Helper — A Open source tool for Actual Budget Malaysian users
  </footer>
</div>
