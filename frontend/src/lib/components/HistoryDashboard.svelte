<script>
  import { clearHistory } from "$lib/stores/history.js";
  import { slide, fade } from "svelte/transition";

  let { conversions = $bindable([]) } = $props();

  let providerFilter = $state("");

  let byProvider = $derived(
    conversions.reduce((acc, c) => {
      acc[c.provider] = (acc[c.provider] || 0) + 1;
      return acc;
    }, {}),
  );

  let filtered = $derived(
    providerFilter
      ? conversions.filter((c) => c.provider === providerFilter)
      : conversions,
  );

  function handleClear() {
    clearHistory();
    conversions = [];
  }

  function formatDate(iso) {
    const d = new Date(iso);
    return d.toLocaleDateString("en-MY", {
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }
</script>

{#if conversions.length > 0}
  <section class="sheet rounded-[var(--radius-apple-lg)] p-5 mt-5" in:fade={{ duration: 300 }}>
    <div class="flex items-center justify-between mb-3">
      <h2 class="text-[15px] font-semibold tracking-tight">History</h2>
      <button class="btn-ghost-apple h-8 px-2 text-sm" data-danger onclick={handleClear}>Clear</button>
    </div>

    {#if Object.keys(byProvider).length > 1}
      <div class="flex flex-wrap gap-2 mb-3">
        <button class="chip" data-selected={!providerFilter || undefined} onclick={() => (providerFilter = "")}>All</button>
        {#each Object.keys(byProvider) as p}
          <button class="chip" data-selected={providerFilter === p || undefined} onclick={() => (providerFilter = p)}>{p.toUpperCase()}</button>
        {/each}
      </div>
    {/if}

    <ul class="flex flex-col divide-y divide-[var(--ah-divider)]">
      {#each filtered as conversion (conversion.id)}
        <li class="py-3 flex items-center gap-3 min-w-0" in:slide={{ duration: 200 }}>
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium truncate tracking-tight">{conversion.filename}</p>
            <p class="text-xs copy-dim">{formatDate(conversion.timestamp)}</p>
          </div>
          <span class="chip shrink-0 text-xs"><span class="text-success">✓</span>{conversion.provider.toUpperCase()}</span>
        </li>
      {/each}
    </ul>
  </section>
{/if}
