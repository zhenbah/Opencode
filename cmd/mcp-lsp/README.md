# MCP-LSP: Language Server Protocol Diagnostics for OpenCode

MCP-LSP is a Model Context Protocol (MCP) server that provides language server protocol (LSP) diagnostics capabilities for OpenCode and other MCP clients.

## Overview

MCP-LSP connects to language servers (like gopls, typescript-language-server, etc.) and exposes diagnostics information (errors, warnings, hints) through the MCP protocol. It's designed to work independently from the main OpenCode application while reusing the same configuration and LSP client code.

## Features

- Exposes LSP diagnostics through a `diagnostics` MCP tool
- **Uses the existing OpenCode configuration file** (no separate config needed)
- Works with any MCP client, not just OpenCode
- Communicates with MCP clients via stdio
- Formats error paths to match the user's input format
- Shows both file-specific and project-wide diagnostics

## Installation

### Using Go Install

The simplest way to install MCP-LSP is using Go's install command:

```bash
go install github.com/opencode-ai/opencode/cmd/mcp-lsp@latest
```

This will install the latest version of the `mcp-lsp` binary to your `$GOPATH/bin` directory.

### Building from Source

To build from source:

```bash
git clone https://github.com/opencode-ai/opencode.git
cd opencode
go build -o mcp-lsp ./cmd/mcp-lsp
```

## Configuration

MCP-LSP reuses the **existing OpenCode configuration**. It looks for configuration in these locations (in order):

1. `$HOME/.opencode.json`
2. `$XDG_CONFIG_HOME/opencode/.opencode.json` 
3. `./.opencode.json` (in the current working directory)

The server specifically uses the `lsp` section of the configuration:

```json
{
  "lsp": {
    "go": {
      "disabled": false,
      "command": "gopls"
    },
    "typescript": {
      "disabled": false,
      "command": "typescript-language-server",
      "args": ["--stdio"]
    }
  }
}
```

## Usage with OpenCode

To use MCP-LSP with OpenCode, add it to your OpenCode configuration as an MCP server:

```json
{
  "mcpServers": {
    "lsp-diagnostics": {
      "type": "stdio",
      "command": "mcp-lsp",
      "args": []
    }
  }
}
```

Once configured, the `diagnostics` tool will be available to the AI assistant in OpenCode.

## Testing with MCP Inspector

The easiest way to test and explore the MCP-LSP server is using the MCP Inspector tool:

```bash
# Install the MCP Inspector (if not already installed)
npm install -g @modelcontextprotocol/inspector

# Run the inspector with the MCP-LSP server
npx @modelcontextprotocol/inspector mcp-lsp
```

The MCP Inspector provides an interactive interface where you can:
- See the available tool (diagnostics)
- Run the diagnostics tool on any file
- View the formatted output
- Explore tool documentation

This is a great way to verify that your LSP servers are configured correctly and that the MCP-LSP server is working as expected.

## Usage with Any MCP Client

MCP-LSP can be used with any MCP-compatible client, not just OpenCode. The server communicates via stdio using the MCP protocol.

Example request:
```json
{
  "method": "tools/call",
  "params": {
    "name": "diagnostics",
    "arguments": {
      "file_path": "test_file.go"
    }
  }
}
```

Example response:
```json
{
  "result": {
    "content": [{
      "type": "text",
      "text": "\n<file_diagnostics>\nError: test_file.go:5:10 [go] syntax error: unexpected if, expecting expression\n</file_diagnostics>\n\n<project_diagnostics>\nWarn: another_file.go:12:5 [go] unused variable: x\n</project_diagnostics>\n\n<diagnostic_summary>\nCurrent file: 1 errors, 0 warnings\nProject: 0 errors, 1 warnings\n</diagnostic_summary>\n"
    }]
  }
}
```

## How It Works

1. MCP-LSP loads the LSP configuration from the existing OpenCode config files
2. It initializes LSP clients for each configured language server
3. When a diagnostics request is received, it:
   - Opens the file in the respective LSP client
   - Waits for diagnostics notifications from the LSP server
   - Formats the diagnostics, preserving the original path format
   - Returns the diagnostics as structured text

## Path Formatting

MCP-LSP will match the path format used in the request:

- If you request diagnostics with a relative path like `test.go`, diagnostics will be displayed with relative paths
- If you use an absolute path, all diagnostics will use absolute paths
- For project-wide diagnostics, paths will follow the same format as the requested file path

## Troubleshooting

If you encounter issues:

1. Ensure the required language servers are installed and in your PATH
2. Verify your OpenCode configuration has the correct LSP settings
3. Check if the language servers work correctly with OpenCode itself
4. Run `mcp-lsp` with the `OPENCODE_DEBUG=true` environment variable for more detailed logs
5. Use the MCP Inspector tool to test the server directly