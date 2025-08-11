---
title: Vango Documentation Home
slug: index
version: 0.1
status: draft
---

# Vango Documentation

Welcome to **Vango**, a Go-native, hybrid-rendered UI framework designed to challenge the status quo of JavaScript-first front-end stacks.  
This site is the single source of truth for every spec, design decision, tutorial, and build plan.

> **How to navigate**
>
> • **Blueprints** – in-depth technical specs of each subsystem.  
> • **Phases** – time-boxed delivery plans with task checklists.  
> • **Guides** – hands-on tutorials for day-to-day development.  
> • **ADRs** – immutable records of architecture decisions.  
> • **Examples** – small, runnable apps.

## Table of Contents

- [Getting Started](guides/quick-start.md)
- [Project Layout](#project-layout)
- Blueprints
  - [Rendering Pipeline](blueprints/rendering-pipeline.md)
  - [Cooperative Scheduler](blueprints/cooperative-scheduler.md)
  - [Live Protocol](blueprints/live-protocol.md)
  - [Styling](blueprints/styling.md)
- Phases
  - [P-0 Core WASM Engine](phases/P-0-core-wasm.md)
  - [P-1 HTML Elements Expansion](phases/P-1-html-elements.md)
  - [Q1 2025 State Store](phases/Q1-2025-state-store.md)
- [Style Guide](style-guide.md)
- [Contributing](../contributing.md)

## Project Layout
```
docs/               <— you are here
pkg/                Go source code (to be generated)
cmd/                CLI entry points
examples/           Mini apps used in docs & tests
```

## Status Badges
*(badges will appear once CI is wired)*

## License
Vango is MIT-licensed. See `LICENSE` at repo root.
