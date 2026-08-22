<script>
  import { fade, fly } from "svelte/transition";
  import { downloadConfig, uploadConfig, unloadConfig } from "$lib/api.js";

  let status = $state("idle"); // idle | loading | success | error
  let message = $state("");
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
      await uploadConfig(f);
      loaded = true;
      status = "success";
      message = "Config loaded.";
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

<div class="glass rounded-2xl px-5 py-4 mb-5" in:fade={{ duration: 400 }}>
  <h2 class="text-[15px] font-semibold tracking-tight">Provider configuration</h2>
  <p class="text-[13px] copy mt-0.5">
    Load a JSON config to set filters, categories, and account mappings for this
    session. Unload to revert providers to empty tuning.
  </p>

  {#if status === "error"}
    <div
      role="alert"
      class="rounded-2xl p-3 mt-3 flex items-center gap-3 text-danger"
      in:fly={{ y: -8, duration: 250 }}
    >
      <span class="flex-1 text-sm">{message}</span>
      <button class="btn-ghost-apple h-8 px-3 text-sm" onclick={handleDismiss}>Dismiss</button>
    </div>
  {:else if status === "success" && message}
    <div
      class="rounded-2xl p-3 mt-3 flex items-center gap-3 text-success"
      in:fly={{ y: -8, duration: 250 }}
    >
      <span class="flex-1 text-sm">{message}</span>
      <button class="btn-ghost-apple h-8 px-3 text-sm" onclick={handleDismiss}>OK</button>
    </div>
  {/if}

  <div class="flex flex-wrap gap-2 mt-3">
    <label class="btn-ghost-apple h-9 border border-[var(--ah-accent-border)]" class:opacity-50={status === "loading"}>
      Load JSON
      <input
        bind:this={fileInput}
        class="hidden"
        type="file"
        accept="application/json,.json"
        onchange={handleFileSelect}
        disabled={status === "loading"}
      />
    </label>
    <button class="btn-ghost-apple h-9" onclick={handleDownload} disabled={status === "loading"}>
      Download Sample
    </button>
    {#if loaded}
      <button class="btn-ghost-apple h-9" data-danger onclick={handleUnload} disabled={status === "loading"}>
        Unload
      </button>
    {:else}
      <span class="btn-ghost-apple h-9 invisible">Unload</span>
    {/if}
    {#if status === "loading"}
      <span class="progress-apple w-16 self-center"><div style="width: 70%"></div></span>
    {:else}
      <span class="w-16 self-center"></span>
    {/if}
  </div>
</div>
