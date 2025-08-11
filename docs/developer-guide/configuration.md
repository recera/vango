# Configuration (vango.json)

A single JSON file controls dev server behavior, styling, and build defaults.

## Full Example
```json
{
  "routesDir": "app/routes",
  "wasmTarget": "wasm",
  "styling": {
    "tailwind": {
      "enabled": true,
      "strategy": "auto",
      "config": "tailwind.config.js",
      "input": "styles/input.css",
      "output": "public/styles.css",
      "watch": true,
      "minify": false,
      "version": "3.4.0",
      "autoDownload": true
    },
    "css": {
      "stylesDir": "styles",
      "modules": false,
      "postCSS": false
    }
  },
  "pwa": {
    "enabled": false,
    "manifest": "public/manifest.json",
    "serviceWorker": "public/sw.js"
  },
  "dev": {
    "port": 5173,
    "host": "localhost",
    "open": false,
    "proxy": {"/api": "http://localhost:8080"},
    "https": false
  }
}
```

## Fields
- `routesDir`: where to scan for file-based routes (`app/routes` by default)
- `wasmTarget`: TinyGo target (keep as `wasm`)
- `styling.tailwind`: enable runner; strategy selection (auto/npm/standalone); in/out paths
- `styling.css`: convenience defaults for global styles
- `pwa`: manifest and service worker paths (no runtime PWA manager included yet)
- `dev`: server address, proxy map for API backends, HTTPS config

## Defaults and Validation
- See `cmd/vango/internal/config/config.go` for defaulting logic
- Missing fields are filled with sensible defaults during load
