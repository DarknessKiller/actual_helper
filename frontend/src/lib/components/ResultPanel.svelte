<script>
  import { fly, fade } from "svelte/transition";

  let { filename = "", provider = "" } = $props();
  let visible = $state(true);
</script>

{#if visible}
  <div
    class="sheet rounded-[var(--radius-apple-lg)] p-6 mt-5 flex items-center gap-4"
    in:fly={{ y: 8, duration: 300 }}
    out:fade={{ duration: 200 }}
    role="status"
  >
    <div class="flex items-center justify-center h-11 w-11 rounded-full bg-success/15 text-success shrink-0">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
      </svg>
    </div>
    <div class="flex-1 min-w-0">
      <p class="font-semibold tracking-tight">Conversion complete</p>
      <p class="text-sm copy-dim truncate">{filename} ({provider.toUpperCase()}) converted and downloaded.</p>
    </div>
    <button class="btn-ghost-apple h-8 px-3 text-sm shrink-0" onclick={() => (visible = false)}>Dismiss</button>
  </div>
{/if}
