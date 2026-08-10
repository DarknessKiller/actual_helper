<script>
  let { onConfigChange } = $props();
  let configError = $state("");
  let input = $state(null);

  async function importConfig(event) {
    const file = event.target?.files?.[0];
    if (!file) return;
    configError = "";
    try {
      const result = JSON.parse(actualHelperSetConfig(await file.text()));
      if (result.error) throw new Error(result.error);
      onConfigChange?.();
    } catch (error) {
      configError = error.message || "Invalid provider config";
    } finally {
      if (input) input.value = "";
    }
  }

  function resetConfig() {
    const result = JSON.parse(actualHelperResetConfig());
    configError = result.error || "";
    if (!configError) onConfigChange?.();
  }
</script>

<div class="card bg-base-100 shadow-sm mb-4">
  <div class="card-body py-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h2 class="font-semibold">Provider configuration</h2>
        <p class="text-xs text-base-content/60">Loaded locally in this browser session; never uploaded.</p>
      </div>
      <div class="flex gap-2">
        <label class="btn btn-sm btn-outline">
          Load JSON
          <input bind:this={input} class="hidden" type="file" accept="application/json,.json" onchange={importConfig} />
        </label>
        <button class="btn btn-sm btn-ghost" type="button" onclick={resetConfig}>Reset</button>
      </div>
    </div>
    {#if configError}<p class="text-sm text-error mt-2" role="alert">{configError}</p>{/if}
  </div>
</div>
