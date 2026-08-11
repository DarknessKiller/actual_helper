<script>
  let { onConfigChange } = $props();
  let configError = $state("");
  let input = $state(null);
  let configLoaded = $state(false);

  const sampleConfig = {
    global: {
      exclude_keywords: [],
      include_keywords: [],
      categories: [
        { keyword: "shopee", group: "Shopping", category: "Online" },
      ],
    },
    providers: {
      tng: { account_mappings: { "": "TNG" } },
      ryt: { account_mappings: { "": "RYT" } },
      hsbccredit: { account_mappings: { "1234 5678 9012 3456": "HSBC Credit Card" } },
      hlb: { account_mappings: { "1234 5678 9012 3456": "HLB Credit Card" } },
      uobcredit: { account_mappings: { "1234 5678 9012 3456": "UOB Credit Card" } },
      gxbank: {
        account_mappings: {
          "GX Savings Account": "GX Savings",
          "Secret stash Bonus Pocket": "GX Pocket",
        },
      },
    },
  };

  function downloadSample() {
    const blob = new Blob([JSON.stringify(sampleConfig, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "provider_config.example.json";
    a.click();
    URL.revokeObjectURL(url);
  }

  async function importConfig(event) {
    const file = event.target?.files?.[0];
    if (!file) return;
    configError = "";
    try {
      const result = JSON.parse(actualHelperSetConfig(await file.text()));
      if (result.error) throw new Error(result.error);
      configLoaded = true;
      onConfigChange?.();
    } catch (error) {
      configError = error.message || "Invalid provider config";
    } finally {
      if (input) input.value = "";
    }
  }

  function unloadConfig() {
    actualHelperResetConfig();
    configLoaded = false;
    onConfigChange?.();
  }
</script>

<div class="card bg-base-100 shadow-sm mb-4">
  <div class="card-body py-4">
    <h2 class="font-semibold">Provider configuration</h2>
    <p class="text-xs text-base-content/60">Load a JSON config to set filters, categories, and account mappings for this browser session. Never uploaded.</p>
    <div class="flex gap-2 mt-2">
      <label class="btn btn-sm btn-outline w-fit">
        Load JSON
        <input bind:this={input} class="hidden" type="file" accept="application/json,.json" onchange={importConfig} />
      </label>
      <button class="btn btn-sm btn-ghost w-fit" onclick={downloadSample}>Download Sample</button>
      {#if configLoaded}
        <button class="btn btn-sm btn-ghost text-error w-fit" onclick={unloadConfig}>Unload</button>
      {/if}
    </div>
    {#if configError}<p class="text-sm text-error mt-2" role="alert">{configError}</p>{/if}
  </div>
</div>
