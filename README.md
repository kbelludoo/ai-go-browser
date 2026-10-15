# AI Go Browser v0.7 — Universal Hybrid Browser for AI Agents (React Edition)

A high-performance semantic web exploration and action-discovery workbench for AI agents, rewritten as a full-featured React application while faithfully preserving all core features, multi-layer architecture, deterministic action resolution, and benchmark suites from the original project.

## Universal 3-Layer Architecture

1. **Layer 1: Fast DOM Engine (Standard for 80%+ of the Web)**:
   - High-throughput, low-overhead semantic DOM parser with text/markdown extraction.
   - Operates in sub-50ms with a simulated memory footprint under 25 MB RAM (average 7–12 MB).
   - Designed for Wikipedia, news portals, documentation, and content-rich pages.

2. **Layer 2: Structured Data & Hydration Payload Extractor**:
   - Automatically extracts `<script id="__NEXT_DATA__">`, Nuxt payloads, and `<script type="application/ld+json">`.
   - Captures structured schemas and entity models directly without waiting for client runtime rendering.

3. **Layer 3: Ephemeral Headless Engine (On-Demand for SPAs)**:
   - On-demand execution for single-page applications (React/Vue/Angular).
   - Automatic fallback detection (`-auto-js`) when an empty client shell is encountered (`<div id="root"></div>`).
   - Simulates ephemeral headless evaluation with instant memory reclamation.

---

## Core Features Ported

- **Conservative Action Resolver (`discover`, `inspect`, `run`)**:
  - Discovers HTML links (`href`), forms (`action`, `method`, `fields`), HTMX hypermedia attributes (`hx-get`, `hx-post`, `hx-delete`, etc.), and literal JavaScript endpoints (`fetch`, `axios`, `$.get/post`, `xhr.open`).
  - Classifies actions by confidence (`declared` vs `inferred`) and security policy (`requires_confirmation` for state mutations and unverified calls).
  - Masks sensitive tokens, credentials, and passwords (`[redacted]`) in agent-visible outputs.
- **Interactive Agent Command Console**:
  - Supports both human-readable CLI input (`open`, `snapshot`, `discover`, `inspect a1`, `run a1`, `press b1`, `fill i1 "valor"`, `query button`, `back`, `help`) and strict JSONL payloads.
  - Persistent stateful JavaScript event loop across button presses (e.g. click counter state updates).
- **Token Reduction Benchmark Suite (`cmd/tokenbench`)**:
  - Real-time comparison between raw network HTML bytes/tokens and distilled semantic Markdown (70%–96% token savings for LLMs).
  - Built-in benchmarks for Example Domain, Hacker News, Go Portal, Wikipedia, and custom URLs.
- **LMArena Service Bridge & Anti-Bot Stealth Simulation (`pkg/arena`, `cmd/arenaservice`)**:
  - Model catalog explorer and blind A/B evaluation runner.
  - reCAPTCHA v3 stealth CDP token minting with human telemetry scoring (~0.9 score, 90s TTL cache).

---

## Quick Start

```bash
# Install dependencies
npm install

# Start development server on port 3000
npm run dev

# Build for production
npm run build
```
