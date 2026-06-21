import { writable, type Writable } from 'svelte/store';

const STORAGE_KEY = 'ac_api_call_count';

function loadCount(): number {
  try {
    const val = localStorage.getItem(STORAGE_KEY);
    return val ? parseInt(val, 10) : 0;
  } catch {
    return 0;
  }
}

function saveCount(n: number): void {
  try {
    localStorage.setItem(STORAGE_KEY, String(n));
  } catch {
    // storage unavailable
  }
}

export const apiCallCount: Writable<number> = writable(loadCount());

export function incrementApiCall(): void {
  apiCallCount.update((n: number): number => {
    const next = n + 1;
    saveCount(next);
    return next;
  });
}

export function resetApiCallCount(): void {
  apiCallCount.set(0);
  saveCount(0);
}
