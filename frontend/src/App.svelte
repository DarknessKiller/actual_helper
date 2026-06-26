<script>
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import UploadForm from "$lib/components/UploadForm.svelte";
  import ResultPanel from "$lib/components/ResultPanel.svelte";
  import HistoryDashboard from "$lib/components/HistoryDashboard.svelte";
  import { loadHistory } from "$lib/stores/history.js";

  let conversions = $state(loadHistory());
  let lastConversion = $state(null);
  let version = $state("");

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
    <div class="flex-none pr-4">
      {#if version}
        <span class="badge badge-soft badge-accent">v{version}</span>
      {/if}
    </div>
  </div>

  <main
    class="max-w-2xl mx-auto px-4 py-6 sm:py-10"
    in:fade={{ duration: 300 }}
  >
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
