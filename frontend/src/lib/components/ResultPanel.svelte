<script>
  import { fade, fly } from "svelte/transition";
  let { filename = "", provider = "", csv = "" } = $props();
  let visible = $state(true);

  function downloadCSV() {
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename.replace(/\.[^.]+$/, "") + "_actual.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  $effect(() => { downloadCSV(); });
</script>

{#if visible}
  <div class="card bg-success/10 border border-success/30 mt-4" in:fly={{ y: 20, duration: 400 }} out:fade={{ duration: 200 }}>
    <div class="card-body py-4">
      <div class="text-4xl mb-1 text-center">✅</div>
      <h3 class="card-title text-success text-base justify-center">Conversion Complete!</h3>
      <p class="text-sm text-base-content/60 text-center">{filename} ({provider.toUpperCase()}) converted locally.</p>
    <div class="flex items-center gap-2 mt-2">
      <span class="text-success text-lg">✓</span>
      <p class="text-sm text-base-content/60">Downloaded to your device.</p>
      <button class="btn btn-ghost btn-xs ml-auto" onclick={() => (visible = false)}>Dismiss</button>
    </div>
    </div>
  </div>
{/if}
