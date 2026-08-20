<script>
  import { convertFile } from "$lib/api.js";
  import { addConversion } from "$lib/stores/history.js";
  import { fly, fade } from "svelte/transition";

  let { onConversionComplete, disabled = false } = $props();

  let provider = $state("");
  let file = $state(null);
  let password = $state("");
  let status = $state("idle");
  let errorMsg = $state("");
  let dragOver = $state(false);

  const providers = [
    { id: "tng", label: "TNG E-wallet", csv: true },
    { id: "ryt", label: "RYT Bank", csv: false },
    { id: "hsbccredit", label: "HSBC Credit Card", csv: false },
    { id: "hlb", label: "HLB Credit Card & HL Bank", csv: false },
    { id: "uobcredit", label: "UOB Credit Card", csv: false },
    { id: "gxbank", label: "GX Bank", csv: false },
  ];
  let fileInput = $state(null);

  const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50 MB

  function selectedProvider() {
    return providers.find((p) => p.id === provider);
  }

  function validFile(f) {
    if (!f) return false;
    if (f.size > MAX_FILE_SIZE) {
      errorMsg = "File too large. Maximum size is 50 MB.";
      return false;
    }
    const pdf = isPDF(f);
    const csv = f.name?.toLowerCase().endsWith(".csv") || f.type === "text/csv";
    return pdf || (csv && selectedProvider()?.csv);
  }

  function handleFileSelect(e) {
    const f = e.target?.files?.[0];
    if (validFile(f)) file = f;
    else if (f) errorMsg = "CSV input is only supported for TNG E-wallet.";
  }

  function handleProviderChange() {
    if (file && !validFile(file)) file = null;
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
    if (validFile(f)) {
      file = f;
    } else if (f) {
      errorMsg = "CSV input is only supported for TNG E-wallet.";
    }
  }

  function isPDF(f) {
    return (
      f?.name?.toLowerCase().endsWith(".pdf") || f?.type === "application/pdf"
    );
  }

  async function handleSubmit() {
    if (disabled || !provider || !file) return;

    status = "uploading";
    errorMsg = "";

    try {
      const { csv } = await convertFile(provider, file, password);
      if (!csv) throw new Error("No CSV output");

      const newHistory = addConversion({
        id: crypto.randomUUID?.() ?? (Date.now().toString(36) + Math.random().toString(36).slice(2)),
        provider,
        filename: file.name,
        timestamp: new Date().toISOString(),
        success: true,
        csv,
      });

      if (onConversionComplete) onConversionComplete({ history: newHistory, csv });

      status = "idle";
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

<div class="sheet rounded-[var(--radius-apple-lg)] p-6 sm:p-8" in:fade={{ duration: 400 }}>
  {#if status === "error"}
    <div
      role="alert"
      class="rounded-2xl p-4 mb-5 flex items-center gap-3 text-danger"
      in:fly={{ y: -8, duration: 250 }}
    >
      <span class="flex-1 text-sm">{errorMsg}</span>
      <button class="btn-ghost-apple h-8 px-3 text-sm" onclick={handleDismissError}>Dismiss</button>
    </div>
  {/if}

  <div class="mb-6">
    <div class="field-label mb-2">Provider</div>
    <select
      id="provider-select"
      class="select-apple"
      bind:value={provider}
      onchange={handleProviderChange}
      disabled={disabled || status === "uploading"}
      aria-label="Provider"
    >
      <option value="" disabled>Select a provider</option>
      {#each providers as p}
        <option value={p.id}>{p.label}</option>
      {/each}
    </select>
  </div>

  <div class="mb-5">
    <div class="field-label mb-2">Transaction file</div>

    <input
      id="file-upload"
      type="file"
      class="hidden"
      bind:this={fileInput}
      accept={selectedProvider()?.csv ? ".csv,.pdf" : ".pdf"}
      onchange={handleFileSelect}
      disabled={disabled || status === "uploading"}
    />

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="dropzone flex flex-col items-center justify-center w-full min-h-[150px] p-6"
      data-active={dragOver || undefined}
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
            <p class="font-medium truncate tracking-tight">{file.name}</p>
            <p class="text-sm copy-dim">{(file.size / 1024).toFixed(1)} KB</p>
          </div>
          <button
            class="btn-ghost-apple h-9 w-9 px-0 text-ink-2"
            aria-label="Remove file"
            onclick={(e) => {
              e.stopPropagation();
              file = null;
              if (fileInput) fileInput.value = "";
            }}
            disabled={disabled || status === "uploading"}>✕</button
          >
        </div>
      {:else}
        <div class="flex flex-col items-center gap-3 text-center copy">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-9 w-9 text-ink-3"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
            />
          </svg>
          <div>
            <p class="text-sm font-medium text-ink">Drop a PDF or CSV here</p>
            <p class="text-sm copy-dim mt-0.5">or click to browse</p>
          </div>
        </div>
      {/if}
    </div>
  </div>

  {#if file && isPDF(file)}
    <div class="mb-5" in:fly={{ y: 6, duration: 200 }}>
      <div class="field-label mb-2">PDF password <span class="copy-dim">(optional)</span></div>
      <input
        id="pdf-password"
        type="password"
        class="input-apple"
        placeholder="Enter password if encrypted"
        bind:value={password}
        disabled={disabled || status === "uploading"}
      />
    </div>
  {/if}

  <button
    class="btn-apple w-full"
    class:opacity-50={disabled || !provider || !file || status === "uploading"}
    disabled={disabled || status === "uploading"}
    onclick={handleSubmit}
  >
    {#if status === "uploading"}
      <span class="progress-apple w-24"><div style="width: 70%"></div></span>
      Converting…
    {:else}
      Convert to Actual CSV
    {/if}
  </button>
</div>
