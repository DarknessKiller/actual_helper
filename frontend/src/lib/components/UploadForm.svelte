<script>
  import { convertFile } from "$lib/api.js";
  import { addConversion } from "$lib/stores/history.js";
  import { fade, fly } from "svelte/transition";

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
        onchange={handleProviderChange}
        disabled={disabled || status === "uploading"}
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
        accept={selectedProvider()?.csv ? ".csv,.pdf" : ".pdf"}
        onchange={handleFileSelect}
        disabled={disabled || status === "uploading"}
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
              disabled={disabled || status === "uploading"}>✕</button
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
          disabled={disabled || status === "uploading"}
        />
      </div>
    {/if}

    <button
      class="btn btn-primary w-full"
      class:btn-disabled={disabled || !provider || !file || status === "uploading"}
      disabled={disabled || status === "uploading"}
      onclick={handleSubmit}
    >
      {#if status === "uploading"}
        <span class="loading loading-spinner loading-sm"></span>
        Converting...
      {:else}
        Convert to Actual CSV
      {/if}
    </button>
  </div>
</div>
