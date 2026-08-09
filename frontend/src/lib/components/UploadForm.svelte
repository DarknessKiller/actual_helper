<script>
  import { convertLocally } from "$lib/api.js";
  import { mergeConfigs } from "$lib/rules.js";
  import { addConversion } from "$lib/stores/history.js";
  import { fade, fly } from "svelte/transition";

  let { onConversionComplete } = $props();

  let provider = $state("");
  let file = $state(null);
  let password = $state("");
  let status = $state("idle");
  let errorMsg = $state("");
  let dragOver = $state(false);
  let statusMessage = $state("");
  let config = $state(null);
  let configName = $state("");
  let configInput = $state(null);
  const emptyConfig = { exclude_keywords: [], include_keywords: [], categories: [], account_mappings: {} };

  const providers = [
    { id: "tng", label: "TNG E-wallet" },
    { id: "ryt", label: "RYT Bank" },
    { id: "hsbccredit", label: "HSBC Credit Card" },
    { id: "hlb", label: "HLB Credit Card & HL Bank" },
    { id: "uobcredit", label: "UOB Credit Card" },
    { id: "gxbank", label: "GX Bank" },
  ];
  let fileInput = $state(null);

  function handleFileSelect(e) {
    const f = e.target?.files?.[0];
    if (f) file = f;
  }

  function handleDragOver(e) {
    e.preventDefault();
    dragOver = true;
  }

  function handleDragLeave() {
    dragOver = false;
  }

  function handleDrop(e) {
    e.preventDefault();
    dragOver = false;
    const f = e.dataTransfer?.files?.[0];
    if (f) file = f;
  }

  function isPDF(f) {
    return (
      f?.name?.toLowerCase().endsWith(".pdf") || f?.type === "application/pdf"
    );
  }

  async function handleSubmit() {
    if (!provider || !file) return;

    status = "uploading";
    errorMsg = "";

    try {
      const activeConfig = config || emptyConfig;
      const response = await convertLocally(provider, file, password || undefined, activeConfig, (progress) => {
        status = `converting-${progress.stage}`;
        statusMessage = progress.message || '';
      });

      const blob = response;
      const url = URL.createObjectURL(blob);
      const filename = `${provider}_actual_budget.csv`;
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);

      const newHistory = addConversion({
        id: crypto.randomUUID(),
        provider,
        filename: file.name,
        timestamp: new Date().toISOString(),
        success: true,
      });

      status = "success";
      if (onConversionComplete) onConversionComplete(newHistory);

      provider = "";
      file = null;
      password = "";
      if (fileInput) fileInput.value = "";
    } catch (err) {
      status = "error";
      errorMsg = err.message || "Something went wrong";
    }
  }

  function handleDismissError() {
    status = "idle";
    errorMsg = "";
  }
</script>

<div class="card bg-base-100 shadow-md" in:fade={{ duration: 400 }}>
  <div class="card-body">
    <h2 class="card-title text-lg">Convert Transaction File</h2>

    {#if status === "error"}
      <div
        role="alert"
        class="alert alert-error mb-4"
        in:fly={{ y: -20, duration: 300 }}
      >
        <span>{errorMsg}</span>
        <button class="btn btn-sm btn-ghost" onclick={handleDismissError}
          >Dismiss</button
        >
      </div>
    {/if}

    <div class="form-control w-full mb-3">
      <label class="label" for="provider-select">
        <span class="label-text font-medium">Provider</span>
      </label>
      <select
        id="provider-select"
        class="select select-bordered w-full"
        bind:value={provider}
        disabled={status.startsWith('converting-')}
      >
        <option value="" disabled>Select a provider</option>
        {#each providers as p}
          <option value={p.id}>{p.label}</option>
        {/each}
      </select>
    </div>

    <div class="form-control w-full mb-3">
      <label class="label" for="file-upload">
        <span class="label-text font-medium">Transaction File</span>
      </label>

      <input
        id="file-upload"
        type="file"
        class="hidden"
        bind:this={fileInput}
        accept=".csv,.pdf"
        onchange={handleFileSelect}
        disabled={status.startsWith('converting-')}
      />

      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="flex flex-col items-center justify-center w-full min-h-[120px] border-2 border-dashed rounded-lg p-6 cursor-pointer transition-all duration-200 {dragOver
          ? 'border-primary bg-primary/10'
          : 'border-base-300 bg-base-200 hover:border-primary hover:bg-base-200'}"
        onclick={() => document.getElementById("file-upload")?.click()}
        ondragover={handleDragOver}
        ondragleave={handleDragLeave}
        ondrop={handleDrop}
        role="button"
        tabindex="0"
        onkeydown={(e) => {
          if (e.key === "Enter" || e.key === " ")
            document.getElementById("file-upload")?.click();
        }}
      >
        {#if file}
          <div class="flex items-center gap-3 w-full min-w-0">
            <span class="text-3xl">{isPDF(file) ? "📄" : "📋"}</span>
            <div class="min-w-0 flex-1">
              <p class="font-medium truncate">{file.name}</p>
              <p class="text-sm text-base-content/50">
                {(file.size / 1024).toFixed(1)} KB
              </p>
            </div>
            <button
              class="btn btn-ghost btn-xs btn-circle"
              onclick={(e) => {
                e.stopPropagation();
                file = null;
                if (fileInput) fileInput.value = "";
              }}
              disabled={status.startsWith('converting-')}>✕</button
            >
          </div>
        {:else}
          <div class="flex flex-col items-center gap-2 text-base-content/50">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-10 w-10"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
              />
            </svg>
            <p class="text-sm">Drop a CSV or PDF here, or click to browse</p>
          </div>
        {/if}
      </div>
    </div>

    {#if file && isPDF(file)}
      <div class="form-control w-full mb-3" in:fly={{ y: 10, duration: 200 }}>
        <label class="label" for="pdf-password">
          <span class="label-text font-medium">PDF Password (optional)</span>
        </label>
        <input
          id="pdf-password"
          type="password"
          class="input input-bordered w-full"
          placeholder="Enter password if encrypted"
          bind:value={password}
          disabled={status.startsWith('converting-')}
        />
      </div>
    {/if}

    <button
      class="btn btn-primary w-full"
      class:btn-disabled={!provider || !file || status.startsWith('converting-')}
      onclick={handleSubmit}
    >
      {#if status.startsWith('converting-')}
        <span class="loading loading-spinner loading-sm"></span>
        {statusMessage || 'Converting...'}
      {:else}
        Convert to Actual CSV
      {/if}
    </button>

    <div class="divider text-xs">Config (optional)</div>

    <input
      type="file"
      class="hidden"
      bind:this={configInput}
      accept=".json"
      onchange={(e) => {
        const f = e.target?.files?.[0];
        if (!f) return;
        const reader = new FileReader();
        reader.onload = () => {
          try {
            config = JSON.parse(reader.result);
            configName = f.name;
          } catch {
            errorMsg = 'Invalid JSON config';
          }
        };
        reader.readAsText(f);
      }}
    />

    {#if config}
      <div class="flex items-center gap-2 text-sm">
        <span class="text-success">✓</span>
        <span class="truncate flex-1">{configName}</span>
        <button
          class="btn btn-ghost btn-xs"
          onclick={() => { config = null; configName = ''; if (configInput) configInput.value = ''; }}
        >Remove</button>
      </div>
    {:else}
      <button
        class="btn btn-outline btn-sm w-full"
        onclick={() => configInput?.click()}
      >Load provider_config.json</button>
    {/if}
  </div>
</div>
