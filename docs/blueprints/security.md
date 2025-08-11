---
title: Security & Hardening
slug: security
version: 0.1
phase: Q1-2025
status: draft
requires:
  - live-protocol
---

# Security & Hardening Blueprint

> **Goal**: Ship secure-by-default apps with minimal developer effort, covering transport, content security, authentication, and supply-chain aspects.

## 1. Threat Model
1. Cross-site scripting (XSS).  
2. Cross-site request forgery (CSRF) on form submissions & Live WS.  
3. Clickjacking & mixed content.  
4. Dependency tampering (malicious addon).  
5. leaked credentials in WASM binary.

## 2. Content Security Policy (CSP)
* Default template generated at build:  
  `default-src 'self'; script-src 'self' 'wasm-eval'; style-src 'self' 'sha256-...'`.
* CLI flag `--csp-report-uri` sets `report-uri`.
* Nonce workflow: `vango.GetCSPNonce(ctx)` returns per-request nonce; `UseStyle()` attaches `nonce` attr.

## 3. CSRF Protection
| Channel | Strategy |
|---------|----------|
| HTML forms | Double-submit cookie + hidden input. |
| Live WS | First WS frame must echo `X-CSRF-Token`; server validates against session. |
| Fetch API | JS helper auto-adds `X-CSRF-Token` header. |

## 4. Authentication & AuthZ
* `auth` addon provides middleware with:  
  – Session cookie, JWT, or OAuth2/OIDC flows.  
  – `RequireRole("admin")` route guard.
* WASM exports `IsAuthenticated()` for client conditionals.

## 5. WASM Binary Hardening
* TinyGo linker flag: `-wasm-abi=generic -no-debug -strip-dwarf`.  
* Optional: `--secure-linker` removes dynamic syscalls (`eval`, `new Function`).

## 6. Dependency Integrity
* `addons.lock` file lists SHA-256 for each external addon version.  
* `vango verify` re-hashes to catch tampering.

## 7. Transport Security
* Recommend HTTPS + HSTS by default; dev server auto-proxies to HTTPS with self-signed cert.

## 8. Open Questions
* Offer automatic Subresource Integrity (SRI) hashes for `<link>` and `<script>`?  
* Should auth middleware inject `SameSite=Strict` cookies globally?

## 9. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
