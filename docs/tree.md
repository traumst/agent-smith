# Project Structure

```text
smithai/
├── cmd/
│   └── ping_mcp/             # MCP testing utility
│       └── main.go
├── src/
│   ├── persistence/          # Persistence Layer: SQLite, files, memory, settings, logs
│   │   ├── db/               # SQLite setup, migrations, extension loading
│   │   ├── vector/           # Vector DB implementation via sqlite-vec
│   │   ├── refs/             # References table (local files, web URLs)
│   │   ├── settings/         # Settings storage and loading (JSON)
│   │   ├── history/          # Chat history storage (SQLite)
│   │   ├── logs/             # Usage logs (SQLite)
│   │   └── memory/           # Long-term memory management
│   ├── agent/                # Agent Layer: Core logic and protocol
│   │   ├── adapter/          # LLM API providers (Gemini)
│   │   ├── protocol/         # Request/Response types
│   │   ├── loop/             # Thinking loop and tool execution
│   │   ├── tools/            # Built-in tools (FS, Terminal, Browser)
│   │   ├── ratelimit/        # RPM rate limiter
│   │   └── consent/          # Tool execution consent management
│   ├── api/                  # API Layer: HTTP handling
│   │   ├── middleware/       # logging, timeout, recovery
│   │   └── handlers/         # UI, Chat, Settings, History handlers
│   ├── ui/                   # UI Layer
│   │   ├── embed.go          # Embedded FS definitions
│   │   └── static/           # Static assets
│   │       ├── templates/    # Go HTML templates
│   │       ├── app.css       # Vanilla CSS
│   │       ├── app.js        # Vanilla JS
│   │       └── htmx.min.js   # Vendored HTMX
│   └── test/                 # Test utilities
├── data/                     # Default persistence directory (git-ignored)
├── main.go                   # Main entry point
├── run.sh                    # Helper script to run the app
├── go.mod                    # Go module definition
└── go.sum                    # Dependencies checksums
```

### Layer Separation

1. **Persistence Layer (`src/persistence`)**: Exclusively handles disk and database operations. Isolated from the agent's logic. SQLite for structured data, plaintext files for long-term memory.
2. **Agent Layer (`src/agent`)**: The core brain. Contains all protocol definitions, context window management, and tool routing. Completely decoupled from HTTP or UI.
3. **API Layer (`src/api`)**: Bridges the Agent Layer to the outside world via HTTP and Server-Sent Events (SSE).
4. **UI Layer (`src/ui`)**: Purely presentational. Consumes the API Layer via HTMX and dynamic Go templates. Static assets embedded into the binary via Go `embed`.
