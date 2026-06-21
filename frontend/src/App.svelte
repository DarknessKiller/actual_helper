<script lang="ts">
  import { onMount } from 'svelte';
  import ApiUsageHistory from './components/ApiUsageHistory.svelte';
  import ProcessUploader from './components/ProcessUploader.svelte';

  let version = $state('');

  onMount(async () => {
    try {
      const res = await fetch('/version');
      const data = await res.json();
      version = data.version;
    } catch {
      version = '';
    }
  });
</script>

<div class="min-h-screen bg-base-200">
  <div class="navbar bg-base-100 shadow-sm mb-4 md:mb-8">
    <div class="navbar-start">
      <a href="/" class="btn btn-ghost text-xl">Actual Helper</a>
    </div>
    <div class="navbar-end">
      <span class="badge badge-soft badge-accent">{#if version}v{version}{/if}</span>
    </div>
  </div>

  <main class="max-w-2xl mx-auto px-2 md:px-4 pb-12">
    <div class="flex justify-center mb-6 md:mb-8">
      <ApiUsageHistory />
    </div>

    <ProcessUploader />
  </main>
</div>
