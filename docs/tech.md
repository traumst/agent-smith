# Technology Stack

## Core Language

- **Go (Golang)**: Chosen for its simplicity, performance, and strong standard library. Go easily compiles to a single, self-sufficient, cross-platform binary that runs anywhere, fulfilling Requirement #1. Its standard library is powerful enough to handle HTTP servers, concurrency (goroutines for streaming), and templating without external frameworks.

## Approved External Dependencies

The project follows a strict "standard library first" policy (Requirement #8). The following are the only approved exceptions:

1. **`mattn/go-sqlite3`** (CGO) — Go has no built-in SQLite support. CGO is acceptable for our use case: we ship a compiled binary, serve a web UI, and have no WASM targets. Provides extension loading required for `sqlite-vec`.
2. **`sqlite-vec`** — Lightweight SQLite extension for vector similarity search. Loaded at runtime via `mattn/go-sqlite3`. Powers keyword-based lookups into long-term memory files without introducing a separate vector database.
3. **`chromedp`** (`github.com/chromedp/chromedp`) — Pure Go Chrome DevTools Protocol client. Used for two purposes: (a) the agent's web browser tool, allowing interaction with full web pages, and (b) simulation/end-to-end tests against the SmithAI web UI. Communicates directly with a local Chrome/Chromium installation — no external driver binaries.
4. **`google.golang.org/genai`** — Official Gemini Go SDK for native integration with the Gemini API.

## UI Layer

- **HTML/CSS/JS (Vanilla)**: Keeps the project simple and free of bloated build steps or complex frontend frameworks.
- **HTMX**: Latest stable version, downloaded from CDN and vendored into `src/ui/static/`. Bundled into the binary via Go `embed`. HTMX enables AJAX, CSS Transitions, and Server-Sent Events (SSE) directly in HTML attributes — streaming LLM responses and providing interactive feedback while keeping the frontend strictly simple.
- **Go `html/template`**: Standard library templating to dynamically render HTML and inject JS/CSS where needed.

## Agent Layer & API

- **Standard HTTP (`net/http`)**: Go's robust HTTP package serves both the API layer (handlers/middleware) and the UI layer.
- **Server-Sent Events (SSE)**: To fulfill the requirement for streaming LLM thinking and reasoning, SSE is used in the API layer and consumed by HTMX or vanilla JS in the UI layer.

## Persistence Layer

- **SQLite3** (via `mattn/go-sqlite3`): Primary local database for structured persistence — chat history, settings, usage logs, and the references table.
- **References Table**: A SQLite table storing pointers to local files and web URLs. Used to track long-term memory files and other referenced resources. Replaces the originally planned standalone `refdb` package.
- **Vector DB** (via `sqlite-vec`): Runs as a SQLite extension within the same database. Stores vectorized keywords extracted from long-term memory files for fast similarity-based retrieval. Designed and built in tandem with the rest of the persistence layer (Phase 1).
- **Long-Term Memory**: Stored as plaintext files on disk (`data/memory/`). Each file is registered in the references table. Keywords are extracted and vectorized for quick lookups. Size is capped at a configurable limit. Files are readable and editable by the user directly.

## Tools Layer

- **File System**: Go `os` and `path/filepath` packages.
- **Terminal**: Go `os/exec` package to safely run commands. Execution is gated by a `.smithai-whitelist` plaintext file (supporting gitignore-like wildcards) at the project root. The user is prompted to `run/auto/block` commands. "Auto" adds the command to the whitelist. The agent parses and identifies chained commands (e.g., `&&`, `|`, `;`) to prevent unintentional auto-execution of rogue scripts.
- **Web Search / Browser**: Uses `chromedp` (approved exception). Blocked by default and prompted for user permission. Once allowed, the agent spawns a headless browser to interact with full web pages.

## Testing

- **Unit Tests**: Standard `go test` for all public functions across all packages.
- **Simulation Tests**: `chromedp`-based end-to-end tests that launch SmithAI, open the web UI in a headless browser, and exercise real user flows (submitting prompts, verifying streamed responses, testing consent dialogs).

## Why This Stack?

This stack guarantees the project requirements are met:

1. **Minimal External Dependencies**: Only four approved exceptions, each with clear justification. Promotes stability, security, and clean code.
2. **Versatility**: The core logic resides in pure Go packages (`src/agent`) fully decoupled from the API or UI. The module is `smithai` — importable directly for IDE/TUI integration.
3. **Configurability**: Go's struct-based architecture makes it easy to pass settings down explicitly instead of relying on globals or environment variables scattered everywhere.
