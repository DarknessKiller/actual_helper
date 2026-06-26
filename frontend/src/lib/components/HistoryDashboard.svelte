<script>
  import { clearHistory } from "$lib/stores/history.js";
  import { fade, slide } from "svelte/transition";

  let { conversions = $bindable([]) } = $props();

  let providerFilter = $state("");

  let stats = $derived({
    total: conversions.length,
    byProvider: conversions.reduce((acc, c) => {
      acc[c.provider] = (acc[c.provider] || 0) + 1;
      return acc;
    }, {}),
    today: conversions.filter((c) => {
      const d = new Date(c.timestamp);
      const now = new Date();
      return d.toDateString() === now.toDateString();
    }).length,
  });

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
  <div class="card bg-base-100 shadow-md mt-6" in:fade={{ duration: 400 }}>
    <div class="card-body">
      <div class="flex items-center justify-between mb-4">
        <h2 class="card-title text-lg">Conversion History</h2>
        <button class="btn btn-ghost btn-sm text-error" onclick={handleClear}>
          Clear All
        </button>
      </div>

      <!-- Stats -->
      <div
        class="stats stats-vertical sm:stats-horizontal shadow-sm mb-4 w-full"
      >
        <div class="stat">
          <div class="stat-title">Total Files</div>
          <div class="stat-value text-primary text-2xl">{stats.total}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Today</div>
          <div class="stat-value text-secondary text-2xl">{stats.today}</div>
        </div>
        {#each Object.entries(stats.byProvider) as [p, count]}
          <div class="stat">
            <div class="stat-title">{p.toUpperCase()}</div>
            <div class="stat-value text-accent text-2xl">{count}</div>
          </div>
        {/each}
      </div>

      <!-- Filter -->
      <div class="flex gap-2 mb-3">
        <button
          class="btn btn-xs {!providerFilter ? 'btn-primary' : 'btn-ghost'}"
          onclick={() => (providerFilter = "")}>All</button
        >
        {#each Object.keys(stats.byProvider) as p}
          <button
            class="btn btn-xs {providerFilter === p
              ? 'btn-primary'
              : 'btn-ghost'}"
            onclick={() => (providerFilter = p)}>{p.toUpperCase()}</button
          >
        {/each}
      </div>

      <!-- History list -->
      <div class="flex flex-col gap-2">
        {#each filtered as conversion (conversion.id)}
          <div
            class="flex items-center justify-between p-3 rounded-lg bg-base-200/50 hover:bg-base-200 transition-colors"
            in:slide={{ duration: 300 }}
          >
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <span class="badge badge-outline badge-sm"
                >{conversion.provider.toUpperCase()}</span
              >
              <div class="min-w-0 flex-1">
                <p class="text-sm font-medium truncate">
                  {conversion.filename}
                </p>
                <p class="text-xs text-base-content/40">
                  {formatDate(conversion.timestamp)}
                </p>
              </div>
            </div>
            <div class="badge badge-success badge-sm gap-1 shrink-0">
              ✓ Done
            </div>
          </div>
        {/each}
      </div>
    </div>
  </div>
{:else}
  <div class="card bg-base-100 shadow-md mt-6" in:fade={{ duration: 400 }}>
    <div class="card-body items-center text-center py-8">
      <div class="text-4xl mb-3 opacity-30">📋</div>
      <h3 class="card-title text-base-content/50">No conversions yet</h3>
      <p class="text-sm text-base-content/40">
        Upload a file above to get started
      </p>
    </div>
  </div>
{/if}
