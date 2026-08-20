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

<div class="glass rounded-2xl px-5 py-4">
  <h2 class="text-[15px] font-semibold tracking-tight">Provider configuration</h2>
  <p class="text-[13px] copy mt-0.5">Load a JSON config to set filters, categories, and account mappings for this session. Never uploaded.</p>
  <div class="flex flex-wrap gap-2 mt-3">
    <label class="btn-ghost-apple h-9 border border-[var(--ah-accent-border)]">
      Load JSON
      <input bind:this={input} class="hidden" type="file" accept="application/json,.json" onchange={importConfig} />
    </label>
    <button class="btn-ghost-apple h-9" onclick={downloadSample}>Download Sample</button>
    {#if configLoaded}
      <button class="btn-ghost-apple h-9" data-danger onclick={unloadConfig}>Unload</button>
    {/if}
  </div>
  {#if configError}<p class="text-sm text-danger mt-2" role="alert">{configError}</p>{/if}
</div>
