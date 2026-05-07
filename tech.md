# Technology Stack

## Core Language
- **Go (Golang)**: Chosen for its simplicity, performance, and strong standard library. Go easily compiles to a single, self-sufficient, cross-platform binary that runs anywhere, fulfilling Requirement #1. Its standard library is powerful enough to handle HTTP servers, concurrency (goroutines for streaming), and templating without external frameworks.

## UI Layer
- **HTML/CSS/JS (Vanilla)**: Keeps the project simple and free of bloated build steps or complex frontend frameworks.
- **HTMX**: Used for interactivity. HTMX allows us to access AJAX, CSS Transitions, WebSockets, and Server-Sent Events (SSE) directly in HTML attributes. This perfectly matches the requirement to stream LLM responses and provide interactive feedback (highlight on hover, toasters) while keeping the frontend strictly simple.
- **Go `html/template`**: Standard library templating to dynamically render HTML and inject JS/CSS where needed.

## Agent Layer & API
- **Standard HTTP (`net/http`)**: Go's robust HTTP package will serve both the API layer (handlers/middleware) and the UI layer.
- **Server-Sent Events (SSE)**: To fulfill the requirement for streaming LLM thinking and reasoning, SSE will be used in the API layer and consumed by HTMX or vanilla JS in the UI layer.

## Persistence Layer
- **SQLite3**: We will use SQLite as our primary local database to handle structured persistence cleanly, including chat history, settings, and logs.
- **Vector DB**: We will utilize the `sqlite-vec` extension (a popular, lightweight SQLite extension for vector search) to enable efficient, local vector storage and fast similarity search directly within our SQLite database. 
- **Go Driver**: We will use a standard Go SQLite driver (such as `mattn/go-sqlite3` with CGO or a CGO-free WASM alternative like `ncruces/go-sqlite3` with `sqlite-vec-go-bindings`) to communicate with the database. This satisfies the requirement for a simple, fully local persistence layer while introducing powerful vector capabilities.

## Tools Layer
- **File System**: Go `os` and `path/filepath` packages.
- **Terminal**: Go `os/exec` package to safely run commands. Execution is gated by a `.smithai-whitelist` plaintext file (supporting gitignore-like wildcards) at the project root. The user is prompted to `run/auto/block` commands. "Auto" adds the command to the whitelist. The agent must parse and identify chained commands (e.g., `&&`, `|`, `;`) to prevent unintentional auto-execution of rogue scripts.
- **Web Search / Browser**: Blocked by default and prompted for user permission. Once allowed, the agent will spawn a real browser to interact with the web. We will use **`chromedp`** (`github.com/chromedp/chromedp`) because it is a pure Go library that communicates directly with a local Chrome installation via the Chrome DevTools Protocol (CDP). This avoids the heavy external driver binaries required by Playwright, strictly aligning with our "minimal external dependencies" philosophy.

## Why This Stack?
This stack guarantees the project requirements are met:
1. **No External Dependencies (mostly)**: Promotes stability, security, and extremely clean code.
2. **Versatility**: The core logic resides in pure Go packages (`internal/agent`, `pkg/smithai`) fully decoupled from the API or UI.
3. **Configurability**: Go's struct-based architecture makes it easy to pass settings down explicitly instead of relying on globals or environment variables scattered everywhere.
