# Project Structure

```text
smith-ai/
├── cmd/
│   └── smith/
│       └── main.go           # Entry point of the application. Wires up dependencies and starts the API/UI servers.
├── internal/
│   ├── persistence/          # Persistence Layer: Handling local files, memory, settings, logs
│   │   ├── vector/           # Vector DB implementation for theme-based memory
│   │   ├── refdb/            # Reference DB to document local files and web references
│   │   ├── settings/         # Settings storage and loading (JSON)
│   │   ├── history/          # Chat history storage
│   │   └── logs/             # Usage logs management
│   ├── agent/                # Agent Layer: Core agent logic and protocol
│   │   ├── adapter/          # LLM API providers (e.g., openai, anthropic)
│   │   ├── protocol/         # Request/Response format types (competence, mood, instructions, tools, deltas)
│   │   ├── loop/             # The main thinking/reasoning loop, stream handling, and tool execution
│   │   ├── tools/            # Out-of-the-box tools (fs, web search, terminal, mcp dummy)
│   │   └── mcp/              # MCP client support and integration
│   ├── api/                  # API Layer: Handling HTTP requests/responses
│   │   ├── middleware/       # HTTP middleware (logging, timeout, recovery)
│   │   └── handlers/         # HTTP handlers for agent interaction (REST and SSE for streaming)
│   └── ui/                   # UI Layer: Handling the web interface
│       ├── templates/        # Go HTML templates
│       └── static/           # Static assets (Vanilla CSS, JS, raw HTML, HTMX)
├── pkg/
│   └── smithai/              # Public facing library code for versatile IDE/TUI integration
├── data/                     # Default local directory for persistence (ignored in git)
├── go.mod                    # Go module file
├── go.sum                    # Go dependencies checksums
└── README.md                 # Project documentation
```

### Layer Separation
1. **Persistence Layer (`internal/persistence`)**: Exclusively handles disk operations. Isolated from the agent's logic.
2. **Agent Layer (`internal/agent`)**: The core brain. Contains all protocol definitions, context window management, and tool routing. Completely decoupled from HTTP or UI.
3. **API Layer (`internal/api`)**: Bridges the Agent Layer to the outside world via HTTP and Server-Sent Events (SSE).
4. **UI Layer (`internal/ui`)**: Purely presentational. Consumes the API Layer via HTMX and dynamic Go templates.
