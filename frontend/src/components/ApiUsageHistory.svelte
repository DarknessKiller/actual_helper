<script lang="ts">
  import { apiCallCount } from '$lib/stores/useApiStore';

  let animating = $state(false);
  let prevCount = $apiCallCount;

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
