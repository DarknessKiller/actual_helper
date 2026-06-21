# Frontend Retro UI Design

**Date:** 2026-06-21
**Status:** Draft

## Overview

Design the Actual Helper frontend with a warm retro aesthetic, playful animations, mobile-friendly responsive layout, and improved UX copy.

## Scope

1. Warm retro daisyUI v5 custom theme
2. Playful CSS animations (page load, counter, form interactions, success/error)
3. Mobile responsive layout (hamburger nav, touch-friendly inputs)
4. `.gitignore` entry for `frontend/dist/`
5. UX copy updates

## Color Palette

Custom daisyUI v5 theme defined in `app.css`:

| Token | Color | Hex |
|-------|-------|-----|
| `primary` | Burnt orange | `#C45A2C` |
| `secondary` | Warm amber | `#E8A838` |
| `accent` | Muted teal | `#3A8C7F` |
| `neutral` | Rich brown | `#4A3525` |
| `base-100` | Cream | `#FFF8E7` |
| `base-200` | Warm beige | `#F5E6C8` |
| `base-300` | Tan | `#E8D5A8` |
| `info` | Soft sky | `#7AB8D4` |
| `success` | Moss green | `#6B8F5E` |
| `error` | Brick red | `#B54737` |

## Animations

All animations respect `prefers-reduced-motion` and play once on load.

| Moment | Effect | Implementation |
|--------|--------|----------------|
| Page load | Stagger fade-in (nav → counter → form) | CSS keyframes + `animation-delay` |
| Counter increment | Count-up with bounce, pulse glow badge | CSS `@keyframes` + reactive class toggle |
| File upload area | Dashed border wiggle on hover, icon bob | CSS `:hover` + `@keyframes` |
| Submit button | 3D press-down (transform + box-shadow) | `active:scale-95` + box-shadow swap |
| Success toast | Slide-in from top, retro ring animation | CSS `@keyframes` with `position: fixed` |
| Error | Horizontal shake | CSS `@keyframes` on alert element |
| Loading spinner | Cassette-reel / retro square spinner | Custom CSS `@keyframes`, not daisyUI default |

## Mobile Layout

- Navbar: collapses to hamburger drawer on `< 768px` (daisyUI `drawer`)
- All cards/forms: full-width on mobile, no `min-width`
- Touch targets: minimum 44px for all interactive elements
- Padding: `px-2` mobile, `px-4 md:px-0` desktop
- No horizontal scroll at any breakpoint
- Current `max-w-2xl mx-auto` pattern preserved

## File Changes

### `.gitignore` (root)
Add `frontend/dist/`

### `frontend/src/app.css`
Replace default daisyUI import with custom theme definition + animation keyframes.

### `frontend/src/App.svelte`
- Add load animations (staggered `animate-fadeIn`)
- Responsive navbar with mobile drawer
- Copy updates

### `frontend/src/components/ApiUsageHistory.svelte`
- Rename: `API Calls` → `Files Processed`
- Rename: `Total processed transactions` → `Total files converted`
- Count-up animation on value change
- Glow pulse on increment

### `frontend/src/components/ProcessUploader.svelte`
- Drag-and-drop file zone with hover animation
- Button 3D press effect
- Shake on error
- Success toast
- Loading spinner replacement

### `frontend/src/app.css`
- Full custom daisyUI theme
- All `@keyframes` definitions
- `prefers-reduced-motion` media query
- Smooth transitions on elements

## Implementation Order

1. `.gitignore` update
2. Custom daisyUI theme + keyframes in `app.css`
3. Copy updates in components
4. Mobile responsive tweaks
5. Animation wiring in components

## Success Criteria

- Page renders with warm retro palette on load
- Elements stagger-fade in on page load
- Counter animates on increment
- File upload wiggle on hover
- Form has 3D press on button, shake on error, slide-in toast on success
- All interactive elements have 44px+ touch targets
- No horizontal scroll on mobile
- `.gitignore` ignores `frontend/dist/`
