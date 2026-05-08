# Implementation Plan

We will build SmithAI iteratively, ensuring each step yields a clean, functional, and testable slice of the application. The primary focus is simplicity and avoiding magic.

## Approved External Dependencies

The following are explicitly approved exceptions to Requirement #8 (no external dependencies):

1. **`mattn/go-sqlite3`** — CGO-based SQLite driver. Required because Go has no built-in SQLite support. CGO is acceptable; we provide a web UI and later an API, not a WASM target.
2. **`sqlite-vec`** — SQLite extension for local vector similarity search. Required for theme-based long-term memory. Will be loaded as an extension via `mattn/go-sqlite3`.
3. **`chromedp`** (`github.com/chromedp/chromedp`) — Pure Go Chrome DevTools Protocol client. Required for browser-based web interaction and simulation tests. No external driver binaries needed — communicates directly with a local Chrome/Chromium installation.
4. **`google.golang.org/genai`** — Official Gemini Go SDK. Required for interacting with the Gemini API natively in Go.

All other functionality must use the Go standard library.

## Phase 1: Foundation & Persistence

**Goal:** Establish the project structure, configuration, storage, and protocol types.

1. **Initialize Project:** Create `go.mod` (module `smithai`) and the initial folder structure (`cmd/`, `src/`).
2. **Settings Management:** Implement `src/persistence/settings`. Create configuration structs passed explicitly (no globals). Define the system prompt structure (Competence, Mood, Instructions). Settings are stored as JSON on disk. The agent cannot modify settings — only the user can, including editing files directly outside the agent. Changes must be reflected immediately on next read.
3. **Storage Setup:** Set up the SQLite database via `mattn/go-sqlite3` with extension loading enabled. Implement schema and access for:
   - `chat_history` — conversation history storage (`src/persistence/history`)
   - `usage_logs` — usage log entries (`src/persistence/logs`)
   - `references` — reference entries pointing to local files or web URLs, stored as rows in a `references` table (`src/persistence/refs`)
4. **Vector DB:** Design and implement `src/persistence/vector` using the `sqlite-vec` extension in tandem with the rest of the persistence layer. This powers keyword-based lookups into long-term memory files.
5. **Long-Term Memory:** Long-term memory is stored as plaintext files on disk (in the `data/memory/` directory, size-capped and configurable). Each memory file is registered in the `references` table. Keywords from each file are extracted and vectorized in the vector DB for fast retrieval. Memory files can be browsed and edited by the user directly.
6. **Protocol Definitions:** Define request and response types in `src/agent/protocol`:

   **Request:**
   | Field | Type | Description |
   |-------|------|-------------|
   | `SystemPrompt` | `SystemPrompt` | Composed of Competence, Mood, Instructions |
   | `UserPrompt` | `string` | The user's current message |
   | `History` | `[]Message` | Previous request/response pairs |
   | `Tools` | `[]ToolDef` | Available tool definitions (JSON format) |
   | `MaxTokens` | `int` | Max tokens for the response |
   | `Stream` | `bool` | Whether to stream the response |

   **Response:**
   | Field | Type | Description |
   |-------|------|-------------|
   | `Done` | `bool` | Whether the task is complete |
   | `Content` | `string` | The text response from the model |
   | `ToolCalls` | `[]ToolCall` | Tool invocations requested by the model |
   | `FileDelta` | `*FileDelta` | Code change: file path, start/end line+col, replacement content |
   | `LoadContext` | `[]string` | Paths/URLs the model wants loaded into context |
   | `TokensUsed` | `int` | Tokens consumed by this response |
   | `Error` | `error` | Error if the request failed |

   **FileDelta:**
   | Field | Type | Description |
   |-------|------|-------------|
   | `Path` | `string` | File path |
   | `StartLine` | `int` | Start line of change |
   | `StartCol` | `int` | Start column of change |
   | `EndLine` | `int` | End line of change |
   | `EndCol` | `int` | End column of change |
   | `Content` | `string` | Replacement content |

## Phase 2: Agent Core & Providers

**Goal:** Implement the core reasoning loop and the ability to talk to an LLM.

1. **Provider Adapters:** Implement `src/agent/adapter` for the Gemini API using `google.golang.org/genai`. Ensure the adapter can handle streaming responses from the model.
2. **Token Estimation:** Implement token counting and context window awareness in the agent layer (near the adapter, which knows the tokenizer). The protocol types carry a `TokensUsed` field; the counting logic lives here.
3. **Thinking Loop:** Implement `src/agent/loop` to manage the request/response cycle. Error handling strategy:
   - **User STOP or timeout:** Circuit breaker — abort immediately, return partial result.
   - **Transient API errors (rate limits, 5xx, network):** Exponential backoff, retry up to 3 times, then fail with clear error message.
   - **Unrecoverable errors (auth, malformed request):** Fail fast, return error immediately.
4. **Tool Dispatch:** Define the JSON tool format and implement the dispatcher to route tool calls safely.
5. **Test Harness:** Write a short scenario in `main.go` that wires up the agent with a provider and runs a basic prompt end-to-end. This serves as the integration smoke test for Phases 2-3.

## Phase 3: Out-of-the-box Tools

**Goal:** Equip the agent with its foundational tools. All tools are headless-only in this phase — consent prompts go to stdout until the UI arrives in Phase 5.

1. **Security/Consent Gateway:** Implement the user-dialog pause logic for sensitive tools. In this phase, consent prompts are text-based (stdin/stdout). The `run/auto/block` workflow is functional but headless.
2. **File System Tool:** Safe read/write access to the local disk.
3. **Terminal Tool:** Safely execute terminal commands using `os/exec`. Implement the `.smithai-whitelist` parser (with wildcard support) and a shell command parser to identify and block chained commands from unintentionally bypassing the whitelist.
4. **Web Search / Browser Tool:** Implement browser automation using `chromedp` to allow the agent to interact with full web pages after explicit user consent. Blocked by default.
5. **MCP Dummy Client:** Implement a basic client to test and verify standard MCP integration capabilities.

## Phase 4: API Layer

**Goal:** Expose the agent via a robust HTTP API.

1. **Middleware:** Add standard logging, timeout, and panic recovery middleware (`src/api/middleware`).
2. **Handlers:** Implement REST endpoints for settings/history and SSE endpoints for streaming chat responses and thoughts.
3. **Integration:** Wire the Agent Layer to the API Layer in `main.go` and ensure it runs functionally as a headless HTTP server.

## Phase 5: UI Layer

**Goal:** Build the interactive web frontend without complex build steps.

1. **Template Setup:** Create base Go HTML templates (`src/ui/templates`) and link Vanilla CSS/JS (`src/ui/static`). Use Go `embed` directive to bundle static assets into the binary.
2. **HTMX Integration:** Vendor HTMX (latest stable from CDN, bundled into `static/`). Use HTMX to submit chat forms and consume the SSE stream from the API layer natively.
3. **Interactivity:** Add simple, modern visual cues: highlight on hover/click/fail, and Vanilla JS toasters to confirm success or inform of issues.
4. **Consent UI:** Replace the headless stdin/stdout consent prompts from Phase 3 with proper UI dialogs for the "pause and consent" workflow.

## Phase 6: Refinement & Advanced Features

**Goal:** Polish the codebase and harden remaining features.

1. **Final Polish:** Review code against Requirement #7 (clean, simple, no unnecessary comments) and #8 (no external dependencies beyond approved exceptions). Check binary compile size and efficiency.
2. **Documentation:** Update README with build instructions, configuration reference, and architecture overview.

## Testing Strategy

- **Unit Tests:** Write unit tests for all public functions across all packages. Run via `go test ./...`.
- **Simulation Tests:** Use `chromedp` to write end-to-end browser tests that interact with the running SmithAI web UI — submitting prompts, verifying streamed responses, testing consent dialogs, etc.
- **Smoke Test:** The `main.go` harness from Phase 2 serves as a quick integration check throughout development.
