# SmithAI

Custom coding agent, built to be highly configurable and reliable.

## Details

SmithAI is built to be a versatile base for an agententic usage.
Provides an API to be used via TUI or as IDE plugin.
This agent is being built to be safety-first.
It respects users privacy and does not send any data without explicit consent.
Any time a new resource is about to receive your info, the agent will pause and raise a user dialog to confirm the action.
The agent provides an easy way to define custom tools in JSON format.
The agent should have a configurable system prompt composed from 3 parts:

1. Competance - what the agent knows and can do, like a resume of the agent, a list of tools, project structure, etc.
2. Mood - flavor or accent for the agent, that affects thinking but not code quality. Like "Yarrr, you are a pirate, matey!" vs "You grok, why say many word when few do trick?"
3. Instructions - Specific task at hand. Derived from vanilla user prompt.

Agent should define a clear protocol to interact with the underlying models. There should be a request and response formats defined as types. Request must contain: system prompt(competance, mood, instructions), user prompt, history of previous requests and responses, tools definitions. Response must contain: whether the task is completed, raw and column number where changes start and end, delta of changes, request additional data to load into context, etc. Think about which other fields are commonly used and we should include to keep our protocol as simple as possible. For starters we are only interested in chatting about and editing code.

Agent should be aware of the context window limitations and estimate token consumption for prompts (not including llm responses). It should account for the system prompt overhead.

Agent should be able to use certain tools out of the box.
This includes:

- File system access - reading and writing files
- Web search - ability to search the web
- Terminal - ability to run commands in the terminal
- MCP dummy client - to verify MCP is working and is integrated properly

## Requirements

1. It should be self sufficient. It I should be able to compile it into a binary to run on any machine.
2. It should be versatile. I should be able to use it as a library to integrate the same agent into an IDE plugin or build a TUI for it late.
3. It should be highly configurable. All adjustable settings should be passed explicitly via function arguments. Env variables should only provide defaults.
4. It should be able to update its own settings, prompts, etc.. Agent self-improvement should be persisted separately and able to be reverted.
5. It should be able to maintain long-term memory across sessions - if user so chooses. Size and location of this memory on disk should be user configurable. Size should be capped at configurable limit.
6. It should be responsive to the user. Stream LLM thinking/reasoning and responses as they arrive. Attempt to recover from transient errors. Use timeouts to signal issues to the user. Fail fast and in controlled manner.
7. Code should be simple, clean, well-structured, and easy to explain. Prefer plain code over magic. Never write comments except when absolutely necessary to explain a design choice or a complex algorithm.
8. Avoid the use of any external dependencies. Default to the standard library. Do not use any external frameworks or libraries unless explicitly allowed.

# Structure

We want to have a clearly defined layers of separation:

1. persistence layer - handling local files, agent memory, settings, logs
   - vector db - for theme-based memory
   - ref db - to reference docs on disk or the web
   - settings - agent configuration
   - chat history - conversation history
   - usage logs - usage logs
2. agent layer - handling agent logic
   - adapters further nested per provider
   - thinking loop
   - tool calls
   - mcp support
3. api layer - handling user requests and responses
   - middleware
   - handlers
4. ui layer - handling the user interface
   - web-ui only for start
   - html/css/js + HTMX templates
   - simple and interactive: highlight on hover/click/fail, toaster to confirm success or inform of an issue.

RAW HTML, JS or CSS should be put in dedicated files and structured properly.
They should be rendered dynamically using Go template.

# Stack

Use GO and plain HTML/CS/JS and a pinch of HTMX for interactivity. Do not use any other frameworks.
