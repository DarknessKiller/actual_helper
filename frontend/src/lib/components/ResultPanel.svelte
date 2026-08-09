<script>
  import { fade, fly } from "svelte/transition";
  let { filename = "", provider = "", csv = "" } = $props();
  let visible = $state(true);
</script>

{#if visible}
  <div class="card bg-success/10 border border-success/30 mt-4" in:fly={{ y: 20, duration: 400 }} out:fade={{ duration: 200 }}>
    <div class="card-body py-4">
      <div class="text-4xl mb-1 text-center">✅</div>
      <h3 class="card-title text-success text-base justify-center">Conversion Complete!</h3>
      <p class="text-sm text-base-content/60 text-center">{filename} ({provider.toUpperCase()}) converted locally. Output is shown in this session and is not downloadable.</p>
      <textarea class="textarea textarea-bordered w-full mt-2 font-mono text-xs" rows="8" readonly value={csv}></textarea>
      <button class="btn btn-outline btn-success btn-sm mt-2" onclick={() => (visible = false)}>Dismiss</button>
    </div>
  </div>
{/if}

<style>
  textarea { resize: vertical; }
</style>
