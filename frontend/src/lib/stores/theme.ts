export type Theme = "light" | "dark" | "amoled";

const STORAGE_KEY = "actual-helper-theme";
const PREFERS_DARK = "(prefers-color-scheme: dark)";

function initialTheme(): Theme {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "light" || saved === "dark" || saved === "amoled") return saved;
  } catch {}
  return window.matchMedia(PREFERS_DARK).matches ? "dark" : "light";
}

function apply(t: Theme) {
  document.documentElement.setAttribute("data-theme", t);
}

export function initTheme() {
  const t = initialTheme();
  apply(t);
  try {
    localStorage.setItem(STORAGE_KEY, t);
  } catch {}
}

export function setTheme(t: Theme) {
  apply(t);
  try {
    localStorage.setItem(STORAGE_KEY, t);
  } catch {}
  return t;
}

export function getTheme(): Theme {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "light" || saved === "dark" || saved === "amoled") return saved;
  } catch {}
  return window.matchMedia(PREFERS_DARK).matches ? "dark" : "light";
}