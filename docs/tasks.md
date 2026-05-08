# SmithAI Implementation Tasks

This document breaks down the development plan into actionable steps across 6 phases. Each step focuses on implementing a vertical slice and wiring it up immediately to ensure end-to-end functionality, keeping in mind the Grug Brained Dev rule: favor simplicity, no over-architecting, and prioritize the standard library.

## Phase 1: Foundation & Persistence

### Step 1.1: Initialization & Protocol Types

- [x] Initialize Go module (`go mod init smithai`).
- [x] Create initial folder structure (`src/agent/protocol`, `src/persistence`).
- [x] Define core protocol types in `src/agent/protocol` (e.g. `Request`, `Response`, `SystemPrompt`, `FileDelta`, `ToolDef`, `ToolCall`).
- [x] **Wiring:** Create a minimal `main.go` that imports these types and prints a dummy request to verify the build.

### Step 1.2: Settings Management

- [x] Implement `src/persistence/settings` to manage agent configuration.
- [x] Create functions to read/write settings (Competence, Mood, Instructions) as JSON from/to disk. Ensure settings are passed explicitly via function arguments (no globals).
- [x] **Wiring:** Update `main.go` to load settings from disk on startup and print the configured mood.

### Step 1.3: Relational Storage (SQLite)

- [x] Set up `mattn/go-sqlite3` DB connection logic (CGO enabled).
- [x] Implement schema and basic CRUD for `chat_history` (`src/persistence/history`).
- [x] Implement schema and basic CRUD for `usage_logs` (`src/persistence/logs`).
- [x] Implement schema and basic CRUD for the `references` table (`src/persistence/refs`).
- [x] **Wiring:** Update `main.go` to initialize the DB, write a test chat message, and query it back.

### Step 1.4: Vector Storage & Long-Term Memory

- [x] Integrate `sqlite-vec` extension loading via `go-sqlite3`.
- [x] Implement `src/persistence/vector` for vectorized keyword lookups.
- [x] Implement logic to manage capped plaintext memory files in `data/memory/` and register them in the `references` table.
- [x] **Wiring:** Update `main.go` to create a dummy memory file, vectorize its keywords, and perform a similarity search.

## Phase 2: Agent Core & Providers

### Step 2.1: Provider Adapter & Token Estimation

- [x] Implement `src/agent/adapter` for the Gemini API using `google.golang.org/genai`.
- [x] Implement streaming responses from the Gemini provider using the SDK's streaming capabilities.
- [x] Implement token estimation logic in the agent layer to track `TokensUsed` (accounting for system prompt overhead).
- [x] **Wiring:** Write a simple CLI tool in `main.go` to send a single prompt to the provider and stream the response to stdout.

### Step 2.2: Thinking Loop & Tool Dispatcher

- [x] Implement the core `src/agent/loop` to manage the request/response cycle.
- [x] Add error handling strategy (exponential backoff for transients, fail-fast for auth, circuit breakers for timeouts/user stops).
- [x] Define the JSON tool format and implement a basic tool dispatcher to route tool calls safely.
- [x] **Wiring:** Update `main.go` with a hardcoded integration test (smoke test) that feeds a prompt requiring a dummy tool call into the loop and verifies the loop handles it.

## Phase 3: Out-of-the-box Tools

### Step 3.1: Consent Gateway & File System Tool

- [ ] Implement a headless (stdin/stdout) consent gateway for sensitive actions (`run/auto/block`).
- [ ] Implement the File System tool for safe local read/write access.
- [ ] **Wiring:** Update `main.go` to run a prompt asking the agent to write a file to disk, triggering the text-based consent prompt.

### Step 3.2: Terminal Tool

- [ ] Implement the Terminal tool using `os/exec`.
- [ ] Implement `.smithai-whitelist` parsing (with wildcard support).
- [ ] Implement a command parser to block chained commands (`&&`, `|`, `;`, etc.) from inadvertently bypassing the whitelist.
- [ ] **Wiring:** Ask the agent to run `ls`, verify it gets prompted, and verify wildcard auto-approval works.

### Step 3.3: Web Browser Tool

- [ ] Implement web browser automation tool using `chromedp`.
- [ ] Block usage by default, requiring explicit user consent.
- [ ] **Wiring:** Ask the agent to summarize a web page, verify it asks for consent, then successfully fetches the page.

### Step 3.4: MCP Dummy Client

- [ ] Implement a basic MCP client to verify integration capabilities.
- [ ] **Wiring:** Verify the agent can read from the MCP dummy client via a specific test prompt.

## Phase 4: API Layer

### Step 4.1: HTTP Middleware & REST Handlers

- [ ] Implement basic middleware: logging, timeout, panic recovery (`src/api/middleware`).
- [ ] Implement REST endpoints for fetching/updating settings, memory, and chat history.
- [ ] **Wiring:** Expose these endpoints via a basic HTTP server in `main.go` and test via `curl`.

### Step 4.2: Streaming Endpoints & Agent Wiring

- [ ] Implement SSE endpoints in the API layer for streaming chat responses and agent thoughts.
- [ ] Connect the API handlers to the `src/agent/loop`.
- [ ] **Wiring:** Use `curl` to send a chat request and verify the SSE stream outputs agent reasoning and the final response.

## Phase 5: UI Layer

### Step 5.1: Static Assets & Templates

- [ ] Create base HTML templates in `src/ui/templates`.
- [ ] Add Vanilla CSS and JS files in `src/ui/static`.
- [ ] Use the Go `embed` directive to bundle static assets into the binary.
- [ ] **Wiring:** Serve the static files and templates via the API layer, verify they load correctly in a local browser.

### Step 5.2: HTMX & UI Interactivity

- [ ] Vendor HTMX into `static/` and wire it up to the chat forms.
- [ ] Consume the API's SSE stream using HTMX to render live chat updates.
- [ ] Implement UI components for interactivity (highlight on hover/click/fail) and toast notifications for errors/success.
- [ ] Replace headless (stdin/stdout) consent prompts with interactive UI dialogs.
- [ ] **Wiring:** Fully interact with the agent through the web UI to verify end-to-end functionality.

## Phase 6: Refinement & Advanced Features

### Step 6.1: Code Polish & Documentation

- [ ] Review codebase to enforce the "Grug brain" rule (simplicity, no magic, clean structure, standard library focus).
- [ ] Remove any unnecessary comments; ensure comments only explain complex design choices.
- [ ] Update `README.md` with build instructions, configuration options, and architecture overview.

### Step 6.2: Final Testing

- [ ] Ensure all unit tests pass (`go test ./...`).
- [ ] Implement `chromedp`-based simulation tests for the Web UI (e.g., submitting prompts, verifying streamed responses, consent dialogs).
- [ ] **Wiring:** Run the full test suite in CI or a local clean environment.
