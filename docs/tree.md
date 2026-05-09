# Project Structure

```text
smithai/
├── cmd/
│   └── smith/
│       └── main.go           # Entry point. Wires up dependencies, starts servers, and hosts the Phase 2 smoke test.
├── src/
│   ├── persistence/          # Persistence Layer: SQLite, files, memory, settings, logs
│   │   ├── db/               # SQLite setup, migrations, extension loading (sqlite-vec)
│   │   ├── vector/           # Vector DB implementation via sqlite-vec for keyword-based memory lookups
│   │   ├── refs/             # References table: pointers to local files and web URLs stored in SQLite
│   │   ├── settings/         # Settings storage and loading (JSON on disk)
│   │   ├── history/          # Chat history storage (SQLite)
│   │   ├── logs/             # Usage logs (SQLite)
│   │   └── memory/           # Long-term memory: read/write plaintext files, keyword extraction, cap enforcement
│   ├── agent/                # Agent Layer: Core agent logic and protocol
│   │   ├── adapter/          # LLM API providers (e.g., gemini)
│   │   ├── protocol/         # Request/Response types (competence, mood, instructions, tools, file deltas)
│   │   ├── loop/             # Thinking loop, stream handling, tool execution, error recovery
│   │   ├── tools/            # Out-of-the-box tools (fs, web search, terminal, mcp dummy)
│   │   ├── ratelimit/        # RPM rate limiter for LLM API calls (configurable, no bursts)
│   │   └── mcp/              # MCP client support and integration
│   ├── api/                  # API Layer: Handling HTTP requests/responses
│   │   ├── middleware/       # HTTP middleware (logging, timeout, recovery)
│   │   └── handlers/         # HTTP handlers for agent interaction (REST and SSE for streaming)
│   └── ui/                   # UI Layer: Handling the web interface
│       ├── templates/        # Go HTML templates
│       └── static/           # Static assets (JS, minimal CSS, HTMX + Tailwind vendored from CDN)
├── data/                     # Default persistence directory (next to binary, git-ignored)
│   └── memory/               # Long-term memory plaintext files
├── go.mod                    # Go module: smithai
├── go.sum                    # Go dependencies checksums
└── README.md                 # Project documentation
```

### Layer Separation

1. **Persistence Layer (`src/persistence`)**: Exclusively handles disk and database operations. Isolated from the agent's logic. SQLite for structured data, plaintext files for long-term memory.
2. **Agent Layer (`src/agent`)**: The core brain. Contains all protocol definitions, context window management, token estimation, and tool routing. Completely decoupled from HTTP or UI.
3. **API Layer (`src/api`)**: Bridges the Agent Layer to the outside world via HTTP and Server-Sent Events (SSE).
4. **UI Layer (`src/ui`)**: Purely presentational. Consumes the API Layer via HTMX and dynamic Go templates. Static assets embedded into the binary via Go `embed`.
