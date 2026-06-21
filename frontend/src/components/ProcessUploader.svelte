<script lang="ts">
  import { incrementApiCall } from '$lib/stores/useApiStore';

  interface Provider {
    id: string;
    label: string;
  }

  interface Feedback {
    type: string;
    message: string;
  }

  const providers: Provider[] = [
    { id: 'tng', label: 'Touch n Go' },
    { id: 'ryt', label: 'RYT Bank' },
  ];

  let password = $state('');
  let selectedProvider = $state('');
  let file: File | null = $state(null);
  let submitting = $state(false);
  let feedback: Feedback | null = $state(null);
  let dragCounter = $state(0);

  $effect(() => {
    if (feedback) {
      const t = setTimeout(() => (feedback = null), 4000);
      return () => clearTimeout(t);
    }
  });

  async function handleSubmit(e: Event) {
    e.preventDefault();

    if (!password || !selectedProvider || !file) {
      feedback = { type: 'alert-error', message: 'Please fill in all fields' };
      return;
    }

    submitting = true;
    feedback = null;

    try {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('password', password);

      const res = await fetch(`/convert/${selectedProvider}`, {
        method: 'POST',
        body: formData,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ title: 'Unknown error' }));
        throw new Error(err.title || `HTTP ${res.status}`);
      }

      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const disposition = res.headers.get('Content-Disposition') || '';
      const match = disposition.match(/filename="([^"]+)"/);
      const filename = match?.[1] ?? `${selectedProvider}_actual_budget.csv`;
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);

      incrementApiCall();
      feedback = { type: 'alert-success', message: 'File converted successfully!' };

      password = '';
      selectedProvider = '';
      file = null;
    } catch (err) {
      feedback = { type: 'alert-error', message: (err as Error).message || 'Conversion failed' };
    } finally {
      submitting = false;
    }
  }

  function handleDragEnter(e: DragEvent) {
    e.preventDefault();
    dragCounter++;
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
  }

  function handleDragLeave() {
    dragCounter--;
    if (dragCounter <= 0) {
      dragCounter = 0;
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragCounter = 0;
    const f = e.dataTransfer?.files?.[0];
    if (f) file = f;
  }
</script>

<div class="card bg-base-100 shadow-xl animate-fadeInUp" style="animation-delay: 0.2s">
  <div class="card-body">
    <h2 class="card-title">Process Transactions</h2>
    <p class="text-sm text-base-content/70 mb-4">
      Upload a CSV or encrypted PDF and get an Actual Budget-compatible CSV.
    </p>

    {#if feedback}
      <div
        role="alert"
        class="alert {feedback.type} mb-4 animate-slideDown"
        class:animate-shake={feedback.type === 'alert-error'}
      >
        <span>{feedback.message}</span>
      </div>
    {/if}

    <form method="POST" enctype="multipart/form-data" onsubmit={handleSubmit}>
      <fieldset class="fieldset mb-4">
        <legend class="fieldset-legend">Bank Provider</legend>
        <select class="select w-full min-h-[44px]" bind:value={selectedProvider} required>
          <option value="" disabled>Select a provider</option>
          {#each providers as p}
            <option value={p.id}>{p.label}</option>
          {/each}
        </select>
      </fieldset>

      <fieldset class="fieldset mb-4">
        <legend class="fieldset-legend">Password</legend>
        <input
          type="password"
          class="input w-full min-h-[44px]"
          bind:value={password}
          placeholder="PDF decryption password"
        />
        <span class="label">Optional for PDFs</span>
      </fieldset>

      <fieldset class="fieldset mb-6">
        <legend class="fieldset-legend">Transaction File</legend>
        <div
          class="file-drop-zone {dragCounter > 0 ? 'dragging' : ''}"
          role="button"
          tabindex="0"
          onclick={() => document.getElementById('file-input')?.click()}
          ondragenter={handleDragEnter}
          ondragover={handleDragOver}
          ondragleave={handleDragLeave}
          ondrop={handleDrop}
          onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') document.getElementById('file-input')?.click(); }}
        >
          <span class="file-icon" class:animate-bob={!file}>📄</span>
          <span class="text-sm">
            {#if file}
              {file.name}
            {:else}
              Drop your file here or click to browse
            {/if}
          </span>
        </div>
        <input
          id="file-input"
          type="file"
          class="file-input w-full hidden"
          accept=".csv,.pdf"
          onchange={(e: Event) => (file = (e.target as HTMLInputElement).files![0])}
          required
        />
      </fieldset>

      <button
        type="submit"
        class="btn btn-primary w-full min-h-[44px] btn-retro"
        disabled={submitting}
      >
        {#if submitting}
          <span class="animate-spinnerReel"></span>
          Processing...
        {:else}
          Convert to Actual Budget CSV
        {/if}
      </button>
    </form>
  </div>
</div>

<style>
  .file-drop-zone {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    padding: 1.5rem;
    border: 2px dashed var(--color-base-300);
    border-radius: var(--radius-box, 0.5rem);
    cursor: pointer;
    transition: border-color 0.2s, background-color 0.2s;
    min-height: 100px;
    justify-content: center;
  }

  .file-drop-zone:hover,
  .file-drop-zone.dragging {
    border-color: var(--color-primary);
    background-color: color-mix(in srgb, var(--color-primary) 8%, transparent);
    animation: wiggle 0.3s ease-in-out;
  }

  .file-icon {
    font-size: 1.5rem;
  }

  .btn-retro {
    position: relative;
    box-shadow: 0 4px 0 #8B3A1A;
    transition: transform 0.05s, box-shadow 0.05s;
  }

  .btn-retro:active {
    transform: translateY(3px);
    box-shadow: 0 1px 0 #8B3A1A;
  }
</style>
