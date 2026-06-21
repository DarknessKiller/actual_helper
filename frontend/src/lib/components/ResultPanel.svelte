<script>
  import { fade, fly } from 'svelte/transition'

  let { filename = '', provider = '' } = $props()
  let visible = $state(true)
</script>

{#if visible}
  <div
    class="card bg-success/10 border border-success/30 mt-4"
    in:fly={{ y: 20, duration: 400 }}
    out:fade={{ duration: 200 }}
  >
    <div class="card-body items-center text-center py-4">
      <div class="text-4xl mb-1">✅</div>
      <h3 class="card-title text-success text-base">Conversion Complete!</h3>
      <p class="text-sm text-base-content/60">
        {filename} ({provider.toUpperCase()}) converted and downloaded.
      </p>
      <button
        class="btn btn-outline btn-success btn-sm mt-2"
        onclick={() => visible = false}
      >
        Dismiss
      </button>
    </div>
  </div>
{/if}
