# Styling Strategy Deep Dive

## Options Overview
- Plain CSS (global files served from `/styles/**` or `/public/`)
- Tailwind CSS via built-in runner (recommended)
- Hybrid approaches (utility classes + small custom CSS)

## Tailwind Setup
1) Configure `vango.json`:
```json
{
  "styling": {
    "tailwind": {
      "enabled": true,
      "strategy": "auto",
      "config": "tailwind.config.js",
      "input": "styles/input.css",
      "output": "public/styles.css",
      "watch": true,
      "autoDownload": true
    }
  }
}
```
2) Create `styles/input.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```
3) Ensure `public/index.html` links `/styles.css`.

## Strategies
- `auto`: prefers local `node_modules` if available, else downloads standalone binary
- `npm`: run via local `node_modules/.bin/tailwindcss` (requires Node/npm)
- `standalone`: download and cache the Tailwind binary automatically

## Usage Examples
```go
// Builder usage with Tailwind utilities
card := builder.Div().
  Class("bg-white dark:bg-gray-900 rounded shadow p-6").
  Children(
    builder.H2().Class("text-xl font-bold").Text("Title").Build(),
    builder.P().Class("text-gray-600").Text("Body text").Build(),
  ).Build()
```

## Scoped CSS
- Create `.css` in `styles/` and reference unique class names in components
- For pseudo-scoping, generate unique prefixes in code (e.g., `cardStyles-abc123`) and concatenate in `Class`

## Dev vs Prod
- Dev runner watches and updates `/public/styles.css` with HMR
- Prod build copies `/public/styles.css` to output; no runner needed

## Tips
- Prefer classes to inline styles for smaller patch payloads
- Keep variants and dark mode in `tailwind.config.js` as needed (e.g., `darkMode: 'class'`)
