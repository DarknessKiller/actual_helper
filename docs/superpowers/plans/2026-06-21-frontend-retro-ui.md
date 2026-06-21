# Frontend Retro UI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply warm retro daisyUI theme, playful CSS animations, mobile responsive layout, and UX copy updates to the Actual Helper frontend.

**Architecture:** All changes are within the existing Svelte 5 + Vite + Tailwind CSS v4 + daisyUI v5 stack in `frontend/`. No new dependencies. Animations use pure CSS `@keyframes`. Theme uses daisyUI's `@plugin "daisyui"` with custom theme config in `app.css`.

**Tech Stack:** Svelte 5, Vite 6, Tailwind CSS 4, daisyUI 5

## Global Constraints

- All animations must respect `prefers-reduced-motion` (use `@media (prefers-reduced-motion: no-preference)` wrapper)
- Touch targets minimum 44px
- No new npm dependencies
- `frontend/dist/` must be added to root `.gitignore`
- Copy: "Files Processed" not "API Calls", "Total files converted" not "Total processed transactions"
- Color palette: primary `#C45A2C`, secondary `#E8A838`, accent `#3A8C7F`, neutral `#4A3525`, base-100 `#FFF8E7`, base-200 `#F5E6C8`, base-300 `#E8D5A8`, info `#7AB8D4`, success `#6B8F5E`, error `#B54737`

---

### Task 1: `.gitignore` update

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Add `frontend/dist/` to `.gitignore`**

Open `.gitignore` and add `frontend/dist/` on its own line before the blank line at EOF.

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: ignore frontend/dist"
```

---

### Task 2: Custom daisyUI theme + keyframes in `app.css`

**Files:**
- Modify: `frontend/src/app.css`

**Interfaces:**
- Produces: CSS custom properties consumed by daisyUI components (color tokens)
- Produces: `@keyframes` referenced by Svelte components (fadeIn, slideDown, countUp, wiggle, shake, glow, press, bob, spinnerReel, sparkle)

- [ ] **Step 1: Replace `app.css` with custom theme + keyframes**

Write `frontend/src/app.css`:

```css
@import "tailwindcss";
@plugin "daisyui";

@plugin "daisyui/theme" {
  name: "retro-warm";
  default: true;
  preferdark: false;
  --color-primary: #C45A2C;
  --color-secondary: #E8A838;
  --color-accent: #3A8C7F;
  --color-neutral: #4A3525;
  --color-neutral-content: #FFF8E7;
  --color-base-100: #FFF8E7;
  --color-base-200: #F5E6C8;
  --color-base-300: #E8D5A8;
  --color-info: #7AB8D4;
  --color-success: #6B8F5E;
  --color-error: #B54737;
}

/* ---- Load animations ---- */

@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(16px); }
  to   { opacity: 1; transform: translateY(0); }
}

@keyframes slideDown {
  from { opacity: 0; transform: translateY(-24px); }
  to   { opacity: 1; transform: translateY(0); }
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20%      { transform: translateX(-6px); }
  40%      { transform: translateX(6px); }
  60%      { transform: translateX(-4px); }
  80%      { transform: translateX(4px); }
}

@keyframes wiggle {
  0%, 100% { transform: rotate(0deg); }
  25%      { transform: rotate(-2deg); }
  75%      { transform: rotate(2deg); }
}

@keyframes bob {
  0%, 100% { transform: translateY(0); }
  50%      { transform: translateY(-4px); }
}

@keyframes countUp {
  from { transform: scale(1); }
  50%  { transform: scale(1.2); }
  to   { transform: scale(1); }
}

@keyframes glow {
  0%, 100% { box-shadow: 0 0 4px var(--color-primary); }
  50%      { box-shadow: 0 0 12px var(--color-primary), 0 0 24px var(--color-secondary); }
}

@keyframes press {
  0%   { transform: translateY(0); box-shadow: 0 4px 0 #8B3A1A; }
  100% { transform: translateY(3px); box-shadow: 0 1px 0 #8B3A1A; }
}

@keyframes spinnerReel {
  0%   { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Utility classes for animations */

.animate-fadeInUp {
  animation: fadeInUp 0.5s ease-out both;
}

.animate-slideDown {
  animation: slideDown 0.4s ease-out both;
}

.animate-shake {
  animation: shake 0.4s ease-out;
}

.animate-wiggle {
  animation: wiggle 0.3s ease-in-out;
}

.animate-bob {
  animation: bob 1s ease-in-out infinite;
}

.animate-countUp {
  animation: countUp 0.3s ease-out;
}

.animate-glow {
  animation: glow 1.5s ease-in-out infinite;
}

.animate-press {
  animation: press 0.1s ease-out forwards;
}

.animate-spinnerReel {
  animation: spinnerReel 1s linear infinite;
  display: inline-block;
  width: 1em;
  height: 1em;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
}

/* Respect reduced motion */
@media (prefers-reduced-motion: reduce) {
  .animate-fadeInUp,
  .animate-slideDown,
  .animate-shake,
  .animate-wiggle,
  .animate-bob,
  .animate-countUp,
  .animate-glow,
  .animate-press,
  .animate-spinnerReel {
    animation: none !important;
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/app.css
git commit -m "feat(ui): add retro-warm theme and animation keyframes"
```

---

### Task 3: Update `ApiUsageHistory` — copy + count-up animation

**Files:**
- Modify: `frontend/src/components/ApiUsageHistory.svelte`

**Interfaces:**
- Consumes: `apiCallCount` store (unchanged, from `$lib/stores/useApiStore.js`)

- [ ] **Step 1: Rewrite `ApiUsageHistory.svelte`**

```svelte
<script>
  import { apiCallCount } from '$lib/stores/useApiStore.js';

  let animating = $state(false);
  let prevCount = $state($apiCallCount);

  $effect(() => {
    const current = $apiCallCount;
    if (current !== prevCount) {
      animating = true;
      const t = setTimeout(() => { animating = false; }, 400);
      prevCount = current;
      return () => clearTimeout(t);
    }
  });
</script>

<div class="stats shadow animate-fadeInUp" style="animation-delay: 0.1s">
  <div class="stat">
    <div class="stat-title">Files Processed</div>
    <div class="stat-value text-primary" class:animate-countUp={animating}>
      {$apiCallCount}
    </div>
    <div class="stat-desc">Total files converted</div>
  </div>
</div>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ApiUsageHistory.svelte
git commit -m "feat(ui): update copy and add count-up animation"
```

---

### Task 4: Update `ProcessUploader` — drag-drop, animations, mobile

**Files:**
- Modify: `frontend/src/components/ProcessUploader.svelte`

**Interfaces:**
- Consumes: `incrementApiCall` from `$lib/stores/useApiStore.js`

- [ ] **Step 1: Rewrite `ProcessUploader.svelte`**

```svelte
<script>
  import { incrementApiCall } from '$lib/stores/useApiStore.js';

  const providers = [
    { id: 'tng', label: 'Touch n Go' },
    { id: 'ryt', label: 'RYT Bank' },
  ];

  let password = $state('');
  let selectedProvider = $state('');
  let file = $state(null);
  let submitting = $state(false);
  let feedback = $state(null);
  let dragging = $state(false);
  let shakeKey = $state(0);

  $effect(() => {
    if (feedback) {
      const t = setTimeout(() => (feedback = null), 4000);
      return () => clearTimeout(t);
    }
  });

  function triggerShake() {
    shakeKey++;
  }

  async function handleSubmit(e) {
    e.preventDefault();

    if (!password || !selectedProvider || !file) {
      feedback = { type: 'alert-error', message: 'Please fill in all fields' };
      triggerShake();
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
      const a = document.createElement('a');
      a.href = url;
      a.download = `${selectedProvider}_actual_budget.csv`;
      a.click();
      URL.revokeObjectURL(url);

      incrementApiCall();
      feedback = { type: 'alert-success', message: 'File converted successfully!' };

      password = '';
      selectedProvider = '';
      file = null;
    } catch (err) {
      feedback = { type: 'alert-error', message: err.message || 'Conversion failed' };
      triggerShake();
    } finally {
      submitting = false;
    }
  }

  function handleDragOver(e) {
    e.preventDefault();
    dragging = true;
  }

  function handleDragLeave() {
    dragging = false;
  }

  function handleDrop(e) {
    e.preventDefault();
    dragging = false;
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
          class="file-drop-zone {dragging ? 'dragging' : ''}"
          role="button"
          tabindex="0"
          onclick={() => document.getElementById('file-input')?.click()}
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
          onchange={(e) => (file = e.target.files[0])}
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
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ProcessUploader.svelte
git commit -m "feat(ui): add drag-drop, retro button, animations, mobile touch targets"
```

---

### Task 5: Update `App.svelte` — responsive layout, stagger animations, mobile drawer

**Files:**
- Modify: `frontend/src/App.svelte`

- [ ] **Step 1: Rewrite `App.svelte`**

```svelte
<script>
  import ApiUsageHistory from './components/ApiUsageHistory.svelte';
  import ProcessUploader from './components/ProcessUploader.svelte';
</script>

<div class="min-h-screen bg-base-200">
  <div class="navbar bg-base-100 shadow-sm mb-4 md:mb-8">
    <div class="navbar-start">
      <div class="dropdown">
        <div tabindex="0" role="button" class="btn btn-ghost btn-circle md:hidden">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </div>
        <ul tabindex="0" class="menu menu-sm dropdown-content bg-base-100 rounded-box z-1 mt-3 w-52 p-2 shadow-sm">
          <li><a class="font-bold">Actual Helper</a></li>
        </ul>
      </div>
      <a class="btn btn-ghost text-xl hidden md:flex">Actual Helper</a>
    </div>
    <div class="navbar-end">
      <span class="badge badge-soft badge-accent">v0.1.0</span>
    </div>
  </div>

  <main class="max-w-2xl mx-auto px-2 md:px-4 pb-12">
    <div class="flex justify-center mb-6 md:mb-8">
      <ApiUsageHistory />
    </div>

    <ProcessUploader />
  </main>
</div>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/App.svelte
git commit -m "feat(ui): responsive layout with mobile drawer, stagger animations"
```

---

### Task 6: Build and verify

- [ ] **Step 1: Build frontend**

```bash
cd frontend
npm run build
```

Expected: exit code 0, `frontend/dist/` updated with new hashed assets.

- [ ] **Step 2: Verify Go server compiles**

```bash
cd ..
go build ./...
```

Expected: exit code 0.

- [ ] **Step 3: Commit build output (if tracking) or verify gitignore**

```bash
git status
```

Expected: `frontend/dist/` not shown as untracked.
