# Implementation Plan

We will build SmithAI iteratively, ensuring each step yields a clean, functional, and testable slice of the application. The primary focus is simplicity and avoiding magic.

## Phase 1: Foundation & Persistence
**Goal:** Establish the project structure, configuration, and fundamental storage mechanisms.
1. **Initialize Project:** Create `go.mod` and the initial folder structure (`cmd`, `internal`, `pkg`).
2. **Settings Management:** Implement `internal/persistence/settings`. Create configuration structs passed explicitly (no globals). Define the system prompt structure (Competence, Mood, Instructions).
3. **Storage Interfaces:** Set up the SQLite database schema and implement local storage for chat history, settings, and usage logs (`internal/persistence/history`, `internal/persistence/logs`).
4. **Protocol Definitions:** Define the request and response formats (types) in `internal/agent/protocol`, including token estimation logic and standardizing context management.

## Phase 2: Agent Core & Providers
**Goal:** Implement the core reasoning loop and the ability to talk to an LLM.
1. **Provider Adapters:** Implement `internal/agent/adapter` for a primary provider (e.g., OpenAI or Anthropic API formats). Ensure the adapter can handle streaming Server-Sent Events from the model.
2. **Context Management:** Implement token counting and context window awareness within the protocol layer.
3. **Thinking Loop:** Implement `internal/agent/loop` to manage the request/response cycle, error recovery, and timeouts. Fail fast and cleanly on unrecoverable errors.
4. **Tool API:** Define the JSON tool format and implement the dispatcher to route tool calls safely.

## Phase 3: Out-of-the-box Tools
**Goal:** Equip the agent with its foundational tools.
1. **Security/Consent Gateway:** Implement the user-dialog pause logic for sensitive tools. Ensure "run/auto/block" functionality is presented to the user via the UI.
2. **File System Tool:** Safe read/write access to the local disk.
3. **Terminal Tool:** Safely execute terminal commands using `os/exec`. Implement the `.smithai-whitelist` parser (with wildcard support) and a shell command parser to identify and block chained commands from unintentionally bypassing the whitelist.
4. **Web Search / Browser Tool:** Implement browser automation using `chromedp` to allow the agent to interact with full web pages after explicit user consent.
5. **MCP Dummy Client:** Implement a basic client to test and verify standard MCP integration capabilities.

## Phase 4: API Layer
**Goal:** Expose the agent via a robust HTTP API.
1. **Middleware:** Add standard logging, timeout, and panic recovery middleware (`internal/api/middleware`).
2. **Handlers:** Implement REST endpoints for settings/history and SSE endpoints for streaming chat responses and thoughts.
3. **Integration:** Wire the Agent Layer to the API Layer in `cmd/smith/main.go` and ensure it runs functionally as a headless HTTP server.

## Phase 5: UI Layer
**Goal:** Build the interactive web frontend without complex build steps.
1. **Template Setup:** Create base Go HTML templates (`internal/ui/templates`) and link Vanilla CSS/JS (`internal/ui/static`).
2. **HTMX Integration:** Use HTMX to submit chat forms and consume the SSE stream from the API layer natively.
3. **Interactivity:** Add simple, modern visual cues: highlight on hover/click/fail, and Vanilla JS toasters to confirm success or inform of issues.
4. **Consent UI:** Build the user dialog for the "pause and consent" workflow required by Phase 3.

## Phase 6: Refinement & Advanced Features
**Goal:** Implement remaining complex features and polish the codebase.
1. **Vector DB (Theme-based memory):** Integrate the `sqlite-vec` extension into our SQLite setup within `internal/persistence/vector` for fast, efficient long-term memory retrieval via local vector similarity search.
2. **Library Abstraction:** Ensure all core logic is neatly exposed via `pkg/smithai` for future IDE/TUI consumers.
3. **Self-Improvement Flow:** Allow the agent to update and persist its own settings/prompts. Create an automatic backup system so these updates are easily revertable.
4. **Final Polish:** Review code against Requirement #7 (clean, simple, no unnecessary comments) and #8 (no external frameworks). Check binary compile size and efficiency.
