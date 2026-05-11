# SmithAI

SmithAI is a simple, powerful agentic AI assistant built in Go.
It uses the Gemini API for reasoning and includes a suite of local tools
for interacting with your file system, terminal, and the web.

## Features

- **Grug Brain Implementation**: Simple code, minimal magic, standard library focused.
- **Local Tools**:
  - **File System**: Read, write, and list files.
  - **Terminal**: Execute shell commands (with user consent).
  - **Browser**: Headless web browsing via `chromedp`.
  - **MCP**: Support for Model Context Protocol servers.
- **Long-term Memory**: Keyword-indexed storage in plaintext files, powered by `sqlite-vec`.
- **Modern Web UI**: interactive chat interface built with HTMX and Tailwind CSS (vendored).
- **Embedded Assets**: All static files and templates are bundled into a single binary.

## Architecture

SmithAI is organized into four distinct layers:

1.  **Persistence Layer (`src/persistence`)**: Handles SQLite (history, settings, logs) and file-based long-term memory.
2.  **Agent Layer (`src/agent`)**: The core reasoning loop, Gemini adapter, and tool dispatcher.
3.  **API Layer (`src/api`)**: REST and SSE endpoints serving the UI and external integrations.
4.  **UI Layer (`src/ui`)**: HTMX-powered web interface.

## Quick Start

### Prerequisites

- Go 1.26 or later
- A [Gemini API Key](https://aistudio.google.com/app/apikey)

### Building and Running

1. Clone the repository:

   ```bash
   git clone https://github.com/traumst/agent-smith.git
   cd agent-smith
   ```

2. Set your API key and run:

   ```bash
   export GEMINI_API_KEY=your_api_key_here
   go run main.go
   ```

3. Open your browser to [http://localhost:8080](http://localhost:8080).

## Configuration

Settings are stored in `data/settings.json`. They are automatically created with defaults on first run.

| Field                  | Description                                                         | Default                   |
| :--------------------- | :------------------------------------------------------------------ | :------------------------ |
| `systemPrompt`         | The personality and instructions for the agent.                     | Expert, helpful, concise. |
| `geminiRPM`            | Rate limit for Gemini API requests (0 for no limit).                | `5`                       |
| `modelRefreshInterval` | How often to refresh available model list from Gemini (`H:M:S.ms`). | `1:0:0.000` (1 hour)      |

## Development

- **Tests**: run `go test ./...`
- **Smoke Test**: run `go run main.go -test`

### Control Files

SmithAI uses simple files for state.
No complex database for rules.

#### Consent (.whitelist / .blacklist)

- `.whitelist`: run command without ask
- `.blacklist`: stop command without ask
- Files hold plain text and regex patterns
- Match by glob, exact string, or command prefix
- Prompt lets user to whitelist/blacklist seamlessly

#### Availability (.available / .unavailable)

- `.available`: marks tool or model as available
- `.unavailable`: marks tool or model as broken
- CSV file format: `name, type, reason, time`
- Items stay marked broken for ~4 hours
- After 4 hours, system will try refreshing the disabled item
