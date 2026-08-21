<script>
  import { fade, fly } from "svelte/transition";
  import { downloadConfig, uploadConfig, unloadConfig } from "$lib/api.js";

  let status = $state("idle"); // idle | loading | success | error
  let message = $state("");
  let applied = $state([]);
  let fileInput = $state(null);
  let loaded = $state(false);

  async function handleDownload() {
    status = "loading";
    message = "";
    try {
      const blob = await downloadConfig();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "provider_config.example.json";
      a.click();
      URL.revokeObjectURL(url);
      status = "success";
      message = "Sample config downloaded.";
    } catch (err) {
      status = "error";
      message = err.message || "Download failed";
    }
  }

  async function handleFileSelect(e) {
    const f = e.target?.files?.[0];
    if (!f) return;
    status = "loading";
    message = "";
    try {
      applied = await uploadConfig(f);
      loaded = true;
      status = "success";
      message = `Config applied to ${applied.length} provider${
        applied.length === 1 ? "" : "s"
      }.`;
      if (fileInput) fileInput.value = "";
    } catch (err) {
      status = "error";
      message = err.message || "Upload failed";
    }
  }

  async function handleUnload() {
    status = "loading";
    message = "";
    try {
      await unloadConfig();
      loaded = false;
      applied = [];
      status = "success";
      message = "Config cleared. Providers run with empty tuning.";
    } catch (err) {
      status = "error";
      message = err.message || "Unload failed";
    }
  }

  function handleDismiss() {
    status = "idle";
    message = "";
  }
</script>

<div class="card bg-base-100 shadow-md" in:fade={{ duration: 400 }}>
  <div class="card-body">
    <h2 class="card-title text-lg">Provider Config</h2>
    <p class="text-sm text-base-content/60 mb-3">
      Load a provider config to tune filtering and categorization, or download
      the sample to edit. Unload to revert providers to empty tuning.
    </p>

    {#if status === "error"}
      <div
        role="alert"
        class="alert alert-error mb-3"
        in:fly={{ y: -20, duration: 300 }}
      >
        <span>{message}</span>
        <button class="btn btn-sm btn-ghost" onclick={handleDismiss}
          >Dismiss</button
        >
      </div>
    {:else if status === "success" && message}
      <div
        class="alert alert-success mb-3"
        in:fly={{ y: -20, duration: 300 }}
      >
        <span>{message}</span>
        <button class="btn btn-sm btn-ghost" onclick={handleDismiss}>OK</button>
      </div>
    {/if}

    <div class="flex flex-wrap gap-2">
      <button
        class="btn btn-outline btn-sm"
        onclick={handleDownload}
        disabled={status === "loading"}
      >
        Download sample
      </button>

      <label
        class="btn btn-primary btn-sm"
        class:btn-disabled={status === "loading"}
      >
        Load config
        <input
          type="file"
          accept="application/json,.json"
          class="hidden"
          bind:this={fileInput}
          onchange={handleFileSelect}
        />
      </label>

      <button
        class="btn btn-ghost btn-sm text-error"
        onclick={handleUnload}
        disabled={status === "loading" || !loaded}
      >
        Unload
      </button>

      {#if status === "loading"}
        <span
          class="loading loading-spinner loading-sm self-center text-base-content/50"
        ></span>
      {/if}
    </div>

    {#if loaded && applied.length > 0}
      <div class="mt-3 flex flex-wrap gap-1">
        {#each applied as name}
          <span class="badge badge-outline badge-sm">{name}</span>
        {/each}
      </div>
    {/if}
  </div>
</div>
