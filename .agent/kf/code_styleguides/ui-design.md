# UI Design Guide

> This is the authoritative UI design reference for the kiloforge dashboard.
> When implementing any frontend component, consult this guide for colors,
> typography, spacing, and component patterns. Do not invent new design tokens.

## 1. Design Philosophy

The kiloforge visual language is **dark, glassy, technical, and premium**. It draws from the kiloforge.dev marketing site to create a cohesive brand experience. Key principles:

- **Glass over opaque** — Panels use semi-transparent backgrounds with backdrop blur, not solid fills
- **Emerald-cyan gradient** — The brand gradient replaces any single accent color
- **Generous whitespace** — Sections breathe with ample vertical spacing
- **White-opacity borders** — Borders use `rgba(255,255,255,...)` at low opacity, never opaque colored borders
- **Restrained color** — Most UI is neutral; color is reserved for brand accents, status indicators, and domain icons

## 2. Color System

### 2.1 Backgrounds

| Token | Value | Usage |
|-------|-------|-------|
| `--bg-page` | `#0a0a0a` | Page background (pure deep black) |
| `--bg-surface` | `rgba(30, 30, 30, 0.4)` | Glass panels, cards, nav, code blocks |
| `--bg-surface-solid` | `rgba(0, 0, 0, 0.6)` | Architecture cards (opaque glass) |
| `--bg-surface-elevated` | `rgba(0, 0, 0, 0.8)` | Emphasized cards (e.g., Orchestrator) |
| `--bg-footer` | `#050505` | Footer background |
| `--bg-hover` | `rgba(255, 255, 255, 0.05)` | Hover state for cards/panels |
| `--bg-icon` | `rgba(255, 255, 255, 0.05)` | Icon container backgrounds (`white/5`) |
| `--bg-button` | `rgba(255, 255, 255, 0.2)` | Button backgrounds (`white/20`) |
| `--bg-code` | `rgba(255, 255, 255, 0.1)` | Inline code backgrounds (`white/10`) |

### 2.2 Text

| Token | Value | Usage |
|-------|-------|-------|
| `--text-primary` | `#ededed` | Headings, primary text (Tailwind `neutral-200`) |
| `--text-secondary` | `#a3a3a3` | Body text, descriptions (Tailwind `neutral-400`, ~66% white) |
| `--text-dimmed` | `#737373` | Footer text, labels (Tailwind `neutral-500`, ~50% white) |
| `--text-code` | `#d4d4d4` | Code block text (Tailwind `neutral-300`) |

### 2.3 Borders

| Token | Value | Usage |
|-------|-------|-------|
| `--border-default` | `rgba(255, 255, 255, 0.05)` | Default panel/card borders (`white/5`) |
| `--border-subtle` | `rgba(255, 255, 255, 0.08)` | Glass panel borders (`white/8`) |
| `--border-medium` | `rgba(255, 255, 255, 0.1)` | Code blocks, footer border (`white/10`) |
| `--border-hover` | `rgba(255, 255, 255, 0.2)` | Hover state borders (`white/20`) |

### 2.4 Brand Gradient

The primary brand expression is an **emerald-to-cyan gradient**:

```css
background: linear-gradient(to right, #34d399, #22d3ee);
/* Tailwind: bg-gradient-to-r from-emerald-400 to-cyan-400 */
```

Used for:
- Gradient text on hero headings (`bg-clip-text text-transparent`)
- Emphasis elements and brand highlights
- The `$` prompt symbol in CLI blocks uses emerald-400

| Token | Value | Tailwind |
|-------|-------|----------|
| `--brand-emerald` | `#34d399` | `emerald-400` |
| `--brand-cyan` | `#22d3ee` | `cyan-400` |

### 2.5 Domain Accent Colors

Each domain area has a designated accent color used for icons and architecture card borders:

| Domain | Color | Hex | Tailwind | Usage |
|--------|-------|-----|----------|-------|
| Infrastructure | Emerald | `#34d399` | `emerald-400` | Private infra, local setup, cloud agents |
| Orchestration | Cyan | `#22d3ee` | `cyan-400` | Orchestrator, agent coordination |
| Dashboard / UI | Indigo | `#818cf8` | `indigo-400` | Dashboard, observability |
| Collaboration | Rose | `#fb7185` | `rose-400` | PRs, Gitea, human+AI collab |
| Tracing | Amber | `#fbbf24` | `amber-400` | Tracing, monitoring, metrics |
| Storage | Blue | `#60a5fa` | `blue-400` | Session persistence, data |
| Human Director | Amber | `#f59e0b` | `amber-500` | The Kiloforger card |

Architecture card border colors use the `500` shade at `20%` opacity:
```css
/* Example: Orchestrator card */
border-color: rgba(6, 182, 212, 0.3);   /* cyan-500/30 */
box-shadow: 0 0 50px rgba(6, 182, 212, 0.15);

/* Example: Cloud Agents card */
border-color: rgba(52, 211, 153, 0.2);  /* emerald-500/20 */
box-shadow: 0 0 30px rgba(52, 211, 153, 0.05);
```

### 2.6 Status Colors

Status colors are carried forward from the existing dashboard — they work well on dark backgrounds:

| Status | Color | Hex | Usage |
|--------|-------|-----|-------|
| Success / Running | Green | `#3dd68c` | Active agents, successful operations |
| Warning | Yellow | `#f0c541` | Warnings, rate limits approaching |
| Error / Stopped | Red | `#f06c6c` | Errors, failed operations, stopped agents |
| Pending / Info | Orange | `#e8944a` | Pending operations, informational |

### 2.7 Selection Colors

```css
::selection {
  background: #262626; /* neutral-800 */
  color: #f5f5f5;      /* neutral-100 */
}
```

## 3. Typography

### 3.1 Font Families

| Token | Value | Usage |
|-------|-------|-------|
| `--font-sans` | `"Geist", "Geist Fallback", ui-sans-serif, system-ui, sans-serif` | All UI text |
| `--font-mono` | `"Geist Mono", "Geist Mono Fallback", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace` | Code, CLI, role subtitles |

**Geist** (by Vercel) is the primary typeface. Load via `next/font` or CDN. If unavailable, fall back to system sans-serif.

### 3.2 Type Scale

| Element | Size | Weight | Line Height | Tracking | Tailwind |
|---------|------|--------|-------------|----------|----------|
| Hero H1 | 96px | 700 (bold) | 1.25 | tighter (-0.05em) | `text-8xl font-bold tracking-tighter` |
| H1 (responsive) | 60px | 700 | 1.25 | tighter | `text-6xl md:text-8xl` |
| Section H2 | 48px | 700 | 1.0 | tighter | `text-5xl font-bold tracking-tighter` |
| Card H3 | 20px | 600 (semibold) | 1.4 (28px) | tight (-0.025em) | `text-xl font-semibold tracking-tight` |
| Hero body | 20px | 400 | 1.625 (32.5px) | normal | `text-xl` |
| Body | 16px | 400 | 1.5 (24px) | normal | `text-base` |
| Nav links | 14px | 500 (medium) | normal | normal | `text-sm font-medium` |
| Labels | 14px | 400 | normal | normal | `text-sm` |
| Mono subtitle | 12px | 400 | normal | normal | `text-xs font-mono` |
| Code inline | 14px | 400 | normal | normal | `text-sm font-mono` |

### 3.3 When to Use Monospace

Use `font-mono` for:
- Role subtitles in architecture diagrams ("Human Director", "Local Control Plane")
- CLI commands and code blocks
- Technical identifiers (track IDs, session IDs, branch names)
- The `$` prompt symbol

Do **not** use monospace for headings, body text, or buttons.

## 4. Spacing & Layout

### 4.1 Spacing Scale

Follow the Tailwind spacing scale. Key values used on kiloforge.dev:

| Token | Value | Usage |
|-------|-------|-------|
| `3` | 12px | Small gaps (icon-to-text) |
| `4` | 16px | Default component padding |
| `6` | 24px | Card padding, medium gaps |
| `8` | 32px | Large card padding, section spacing |
| `12` | 48px | Section vertical padding |
| `16` | 64px | Major section spacing |
| `24` | 96px | Hero section padding |

### 4.2 Container & Layout

| Property | Value | Usage |
|----------|-------|-------|
| Max content width | `max-w-7xl` (1280px) | Primary content container |
| Narrow content | `max-w-4xl` (896px) | Hero text, centered content |
| Grid columns | 3 | Feature cards (responsive: 1 → 2 → 3) |
| Grid gap | `gap-6` (24px) | Card grids |
| Page padding | `px-6` (24px) | Horizontal page padding |

### 4.3 Grid Overlay (Optional Decorative)

For hero or landing sections, a subtle grid overlay can be applied:

```css
.grid-overlay {
  background-image:
    linear-gradient(90deg, rgba(255, 255, 255, 0.03) 1px, transparent 1px),
    linear-gradient(rgba(255, 255, 255, 0.03) 1px, transparent 1px);
  background-size: 64px 64px;
}
```

### 4.4 Section Separators

Decorative gradient lines between sections:

```css
.section-separator {
  height: 1px;
  background: linear-gradient(
    to right,
    transparent 0%,
    rgba(255, 255, 255, 0.2) 50%,
    transparent 100%
  );
  max-width: 1024px;
  margin: 0 auto;
}
```

## 5. Glass Panels

The **glass panel** is the signature surface pattern. Use it instead of opaque cards.

### 5.1 Base Glass Panel

```css
.glass-panel {
  background: rgba(30, 30, 30, 0.4);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
}
```

### 5.2 Hover State

```css
.glass-panel:hover {
  border-color: rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.05);
}
```

Optionally add a subtle glow on hover:
```css
.glass-panel:hover {
  box-shadow: 0 0 30px rgba(255, 255, 255, 0.05);
}
```

### 5.3 Variants

| Variant | Background | Border | Usage |
|---------|-----------|--------|-------|
| Default | `rgba(30,30,30,0.4)` | `white/8` | Feature cards, nav, code blocks |
| Solid | `rgba(0,0,0,0.6)` | domain color at `20%` | Architecture cards |
| Elevated | `rgba(0,0,0,0.8)` | domain color at `30%` | Emphasized cards (Orchestrator) |

### 5.4 Complete Feature Card Example

```css
.feature-card {
  /* Glass base */
  background: rgba(30, 30, 30, 0.4);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 16px;
  padding: 32px;
  transition: all 0.2s ease;
}

.feature-card:hover {
  border-color: rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.05);
  box-shadow: 0 0 30px rgba(255, 255, 255, 0.05);
}

.feature-card .icon-container {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 24px;
  transition: transform 0.2s ease;
}

.feature-card:hover .icon-container {
  transform: scale(1.1);
}
```

## 6. Interactive Elements

### 6.1 Buttons

**Primary (disabled / coming soon):**
```css
.btn-primary-disabled {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.5);
  border-radius: 12px;
  padding: 16px 32px;
  font-weight: 500;
  cursor: not-allowed;
}
```

**Ghost button (nav hamburger):**
```css
.btn-ghost {
  background: transparent;
  color: #a3a3a3; /* neutral-400 */
  padding: 8px;
  transition: color 0.2s ease;
}
.btn-ghost:hover {
  color: #ffffff;
}
```

### 6.2 Links

Nav links use `text-sm font-medium text-neutral-400` with hover to white:
```css
a.nav-link {
  color: #a3a3a3;
  font-size: 14px;
  font-weight: 500;
  transition: color 0.2s ease;
}
a.nav-link:hover {
  color: #ffffff;
}
```

### 6.3 Focus States

Use a visible focus ring for accessibility:
```css
:focus-visible {
  outline: 2px solid #22d3ee; /* cyan-400 */
  outline-offset: 2px;
}
```

### 6.4 Transitions

All interactive elements should use smooth transitions:
```css
transition: all 0.2s ease;
/* or for specific properties: */
transition: color 0.2s ease, border-color 0.2s ease, background 0.2s ease;
```

## 7. Iconography

### 7.1 Icon Library

Use **Lucide React** icons (the library used on kiloforge.dev).

### 7.2 Icon Containers

| Property | Value |
|----------|-------|
| Container size | 48x48px (feature cards), 44x44px (architecture cards) |
| Background | `rgba(255, 255, 255, 0.05)` |
| Border radius | 8px |
| Icon size | 24px (in 48px container) |
| Icon color | Domain accent color (see Section 2.5) |

For architecture cards, icon containers use domain-tinted backgrounds:
```css
/* Example: Orchestrator icon */
.icon-container--orchestration {
  background: rgba(6, 182, 212, 0.1); /* cyan-500 at 10% */
}
```

### 7.3 Domain-to-Icon Color Mapping

| Feature | Icon | Color Class |
|---------|------|-------------|
| Private Infrastructure | `Shield` | `text-emerald-400` |
| Cloud Agents | `SquareTerminal` | `text-cyan-400` |
| Real-Time Dashboard | `LayoutDashboard` | `text-indigo-400` |
| Human + AI Collaboration | `GitPullRequest` | `text-rose-400` |
| Cradle-to-Grave Tracing | `Activity` | `text-amber-400` |
| Session Persistence | `HardDrive` | `text-blue-400` |

## 8. Branding Assets

### 8.1 Logo

- **File:** `kf_logo.webp`
- **Source:** `https://kiloforge.dev/kf_logo.webp`
- **Native size:** 985x493px
- **Nav size:** 32x16px (small, alongside "Kiloforge" text)
- **Hero size:** Large, centered
- **Format:** WebP

Usage rules:
- Always display on dark backgrounds
- Do not apply filters or color overlays
- Maintain aspect ratio
- In nav: pair with "Kiloforge" text in `font-semibold tracking-tight`

### 8.2 Favicon

- **File:** `icon.png`
- **Source:** `https://kiloforge.dev/icon.png`
- **Usage:** Browser tabs, bookmarks
- **Note:** The dashboard should use this clean icon, not the full hero artwork

### 8.3 Open Graph Image

- **File:** `og_image.png`
- **Source:** `https://kiloforge.dev/og_image.png`
- **Usage:** Social sharing previews

## 9. Status Semantics

Map application states to colors consistently:

| State | Color | Hex | Dot/Badge | Usage |
|-------|-------|-----|-----------|-------|
| Running / Active | Green | `#3dd68c` | Solid green dot | Active agents, healthy services |
| Completed / Success | Emerald | `#34d399` | Check icon | Completed tracks, merged PRs |
| Warning / Degraded | Yellow | `#f0c541` | Warning triangle | Rate limit warnings, degraded health |
| Error / Failed | Red | `#f06c6c` | X icon | Crashed agents, failed merges |
| Pending / Queued | Orange | `#e8944a` | Clock icon | Pending tracks, queued operations |
| Idle / Paused | Neutral | `#a3a3a3` | Hollow dot | Idle agents, paused operations |
| Info | Cyan | `#22d3ee` | Info icon | Informational messages |

## 10. CSS Custom Properties

The complete set of CSS custom properties for the dashboard. Replace the current `index.css` `:root` block with these tokens:

```css
:root {
  /* Backgrounds */
  --bg-page: #0a0a0a;
  --bg-surface: rgba(30, 30, 30, 0.4);
  --bg-surface-solid: rgba(0, 0, 0, 0.6);
  --bg-surface-elevated: rgba(0, 0, 0, 0.8);
  --bg-footer: #050505;
  --bg-hover: rgba(255, 255, 255, 0.05);
  --bg-icon: rgba(255, 255, 255, 0.05);
  --bg-button: rgba(255, 255, 255, 0.2);
  --bg-code: rgba(255, 255, 255, 0.1);

  /* Text */
  --text-primary: #ededed;
  --text-secondary: #a3a3a3;
  --text-dimmed: #737373;
  --text-code: #d4d4d4;

  /* Borders */
  --border-default: rgba(255, 255, 255, 0.05);
  --border-subtle: rgba(255, 255, 255, 0.08);
  --border-medium: rgba(255, 255, 255, 0.1);
  --border-hover: rgba(255, 255, 255, 0.2);

  /* Brand */
  --brand-emerald: #34d399;
  --brand-cyan: #22d3ee;
  --brand-gradient: linear-gradient(to right, #34d399, #22d3ee);

  /* Domain accents */
  --accent-infrastructure: #34d399;
  --accent-orchestration: #22d3ee;
  --accent-dashboard: #818cf8;
  --accent-collaboration: #fb7185;
  --accent-tracing: #fbbf24;
  --accent-storage: #60a5fa;

  /* Status */
  --status-success: #3dd68c;
  --status-warning: #f0c541;
  --status-error: #f06c6c;
  --status-pending: #e8944a;
  --status-info: #22d3ee;
  --status-idle: #a3a3a3;

  /* Typography */
  --font-sans: "Geist", "Geist Fallback", ui-sans-serif, system-ui, sans-serif,
    "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
  --font-mono: "Geist Mono", "Geist Mono Fallback", ui-monospace, SFMono-Regular,
    Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;

  /* Border radius */
  --radius-sm: 8px;
  --radius-md: 12px;
  --radius-lg: 16px;
  --radius-pill: 9999px;

  /* Backdrop */
  --blur: 12px;

  /* Transitions */
  --transition-fast: 0.15s ease;
  --transition-default: 0.2s ease;
}
```

### 10.1 Migration from Current Dashboard Tokens

| Old Token | New Token | Notes |
|-----------|-----------|-------|
| `--bg: #0f1117` | `--bg-page: #0a0a0a` | Darker, pure black |
| `--surface: #1a1d27` | `--bg-surface: rgba(30,30,30,0.4)` | Transparent glass, not opaque |
| `--border: #2a2d3a` | `--border-subtle: rgba(255,255,255,0.08)` | White-opacity, not colored |
| `--text: #e1e4ed` | `--text-primary: #ededed` | Slightly warmer neutral |
| `--text-dim: #8b8fa3` | `--text-secondary: #a3a3a3` | Neutral gray |
| `--accent: #6c8cff` | `--brand-gradient` | Gradient replaces single blue |
| `--green: #3dd68c` | `--status-success: #3dd68c` | Unchanged |
| `--yellow: #f0c541` | `--status-warning: #f0c541` | Unchanged |
| `--red: #f06c6c` | `--status-error: #f06c6c` | Unchanged |
| `--orange: #e8944a` | `--status-pending: #e8944a` | Unchanged |
| `--radius: 8px` | `--radius-sm/md/lg` | Expanded scale |
| (none — system fonts) | `--font-sans` / `--font-mono` | Geist font family |
