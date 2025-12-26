Go_Agent

Overview
- Go chat agent with tool calling, MCP tool bridge, and a NetAgent for multi-agent message routing.
- Supports a base agent and a ReAct-style agent.
- Built-in tools: web search, weather, current time.

Requirements
- Go 1.20+ (or your local Go toolchain)
- Python 3 (for mcp_server.py)

Quick start
1) Set your API key (recommended via env):
   - PowerShell: `$env:AGENT_API_KEY="your_key_here"`
2) Optional: edit agent.yaml for model/base_url/tool settings.
3) Run:
   - `go run .`
4) Chat in the terminal. Type `exit` to quit.

Configuration
- `agent.yaml` is loaded from the project root. You can also override via env vars:
  - `AGENT_API_KEY`
  - `AGENT_BASE_URL`
  - `AGENT_MODEL`
  - `AGENT_ALLOW_TOOLS`
  - `AGENT_SYSTEM_PROMPT`
  - `AGENT_TEMPERATURE`
  - `AGENT_MAX_CIRCLE`
  - `AGENT_REACT_ENABLED`
- If `react.enabled` is true, the ReAct agent is used.
- Do not commit real API keys.

Project layout
- `main.go`: CLI chat loop, MCP client, tool registration.
- `agent/`: agent core, prompt wrapper, config, ReAct agent, tools.
- `Agent/NetAgent/`: multi-agent network and routing logic.
- `mcp_server.py`: MCP server process started by main.

NetAgent usage
- Create nodes with `AddNode`, connect with `AddEdge`, call `Start`, then send messages.
- Tests under `Agent/NetAgent/netagent_test.go` demonstrate message routing and logging.

Tests
- `go test ./...`

Notes
- Default built-in tools are registered in `main.go` via `registerTools`.
- The MCP server is launched over stdio using Python.
