# ⌬ OpenCode

<p align="center"><img src="https://github.com/user-attachments/assets/9ae61ef6-70e5-4876-bc45-5bcb4e52c714" width="800"></p>

> **⚠️ Early Development Notice:** This project is in early development and is not yet ready for production use. Features may change, break, or be incomplete. Use at your own risk.

A powerful terminal-based AI assistant for developers, providing intelligent coding assistance directly in your terminal.

## Overview

OpenCode is a Go-based CLI application that brings AI assistance to your terminal. It primarily operates as an **Orchestrator** via a command-line interface (CLI), allowing you to interact with its AI capabilities and manage other AI worker agents. It also supports running in a headless **Worker Mode** for distributed task execution.

<p>For a quick video overview, check out
<a href="https://www.youtube.com/watch?v=P8luPmEa1QI"><img width="25" src="https://upload.wikimedia.org/wikipedia/commons/0/09/YouTube_full-color_icon_%282017%29.svg"> OpenCode + Gemini 2.5 Pro: BYE Claude Code! I'm SWITCHING To the FASTEST AI Coder!</a></p>

<a href="https://www.youtube.com/watch?v=P8luPmEa1QI"><img width="550" src="https://i3.ytimg.com/vi/P8luPmEa1QI/maxresdefault.jpg"></a><p>

## Features

- **Interactive Orchestrator CLI**: A command-line interface for direct interaction and agent management.
- **Multi-Agent Architecture**: Foundation for an Orchestrator to manage multiple worker agents (experimental).
- **Multiple AI Providers**: Support for OpenAI, Anthropic Claude, Google Gemini, AWS Bedrock, Groq, Azure OpenAI, and OpenRouter
- **Session Management**: Save and manage multiple conversation sessions (Note: CLI interaction context is simpler than TUI sessions).
- **Tool Integration**: AI can execute commands, search files, and modify code
- **Persistent Storage**: SQLite database for storing conversations and sessions
- **LSP Integration**: Language Server Protocol support for code intelligence (primarily for worker agents or if used by Orchestrator's agent).
- **File Change Tracking**: Track and visualize file changes during sessions (may be less relevant for CLI).
- **Named Arguments for Custom Commands**: Create powerful custom commands with multiple named placeholders

## Installation

### Using the Install Script

```bash
# Install the latest version
curl -fsSL https://raw.githubusercontent.com/opencode-ai/opencode/refs/heads/main/install | bash

# Install a specific version
curl -fsSL https://raw.githubusercontent.com/opencode-ai/opencode/refs/heads/main/install | VERSION=0.1.0 bash
```

### Using Homebrew (macOS and Linux)

```bash
brew install opencode-ai/tap/opencode
```

### Using AUR (Arch Linux)

```bash
# Using yay
yay -S opencode-ai-bin

# Using paru
paru -S opencode-ai-bin
```

### Using Go

```bash
go install github.com/opencode-ai/opencode@latest
```

## Configuration

OpenCode looks for configuration in the following locations:

- `$HOME/.opencode.json`
- `$XDG_CONFIG_HOME/opencode/.opencode.json`
- `./.opencode.json` (local directory)

### Auto Compact Feature

OpenCode includes an auto compact feature that automatically summarizes your conversation when it approaches the model's context window limit. When enabled (default setting), this feature:

- Monitors token usage during your conversation
- Automatically triggers summarization when usage reaches 95% of the model's context window
- Creates a new session with the summary, allowing you to continue your work without losing context
- Helps prevent "out of context" errors that can occur with long conversations

You can enable or disable this feature in your configuration file:

```json
{
  "autoCompact": true // default is true
}
```

### Environment Variables

You can configure OpenCode using environment variables:

| Environment Variable       | Purpose                                                |
| -------------------------- | ------------------------------------------------------ |
| `ANTHROPIC_API_KEY`        | For Claude models                                      |
| `OPENAI_API_KEY`           | For OpenAI models                                      |
| `GEMINI_API_KEY`           | For Google Gemini models                               |
| `VERTEXAI_PROJECT`         | For Google Cloud VertexAI (Gemini)                     |
| `VERTEXAI_LOCATION`        | For Google Cloud VertexAI (Gemini)                     |
| `GROQ_API_KEY`             | For Groq models                                        |
| `AWS_ACCESS_KEY_ID`        | For AWS Bedrock (Claude)                               |
| `AWS_SECRET_ACCESS_KEY`    | For AWS Bedrock (Claude)                               |
| `AWS_REGION`               | For AWS Bedrock (Claude)                               |
| `AZURE_OPENAI_ENDPOINT`    | For Azure OpenAI models                                |
| `AZURE_OPENAI_API_KEY`     | For Azure OpenAI models (optional when using Entra ID) |
| `AZURE_OPENAI_API_VERSION` | For Azure OpenAI models                                |
| `LOCAL_ENDPOINT`           | For self-hosted models                                 |
| `SHELL`                    | Default shell to use (if not specified in config)      |

### Shell Configuration

OpenCode allows you to configure the shell used by the bash tool. By default, it uses the shell specified in the `SHELL` environment variable, or falls back to `/bin/bash` if not set.

You can override this in your configuration file:

```json
{
  "shell": {
    "path": "/bin/zsh",
    "args": ["-l"]
  }
}
```

This is useful if you want to use a different shell than your default system shell, or if you need to pass specific arguments to the shell.

### Configuring LLM Providers

OpenCode allows you to connect to various LLM providers. You configure these providers in your `.opencode.json` file, primarily by providing an API key and then selecting models for different agents (like `coder`, `task`, etc.) in the `agents` section of the configuration.

API keys can often be set via environment variables (see "Environment Variables" section) or directly in the `.opencode.json` file under the `providers` object. If an API key is set in both an environment variable and the config file, the environment variable usually takes precedence.

Here are a few examples:

#### Example: Configuring OpenRouter
1.  **Set your API Key:** Add your OpenRouter API key to the `providers.openrouter` object in your `.opencode.json`:
    ```json
    {
      // ... other configurations ...
      "providers": {
        // ... other providers ...
        "openrouter": {
          "apiKey": "sk-or-v1-YOUR_OPENROUTER_API_KEY", // Replace with your actual key
          "disabled": false
        }
      },
      // ... rest of the configuration ...
    }
    ```
2.  **Select an OpenRouter Model:** In the `agents` section, specify the OpenRouter model you wish to use. Model names are typically prefixed with `openrouter/`. For example, to use Deepseek's `deepseek-r1-0528` (a free model at the time of writing):
    ```json
    {
      // ... other configurations ...
      "agents": {
        "coder": { // Or any other agent like 'task', 'title'
          "model": "openrouter/deepseek/deepseek-r1-0528:free",
          "maxTokens": 4000 // Adjust as needed
        }
        // ... other agents ...
      },
      // ... rest of the configuration ...
    }
    ```
    You can find available models on the OpenRouter website.

#### Example: Configuring Google Gemini
1.  **Set your API Key:** The recommended way for Gemini is often via the `GEMINI_API_KEY` environment variable.
    ```bash
    export GEMINI_API_KEY="AIzaSyYOUR_GEMINI_API_KEY"
    ```
    Alternatively, if supported by your OpenCode version for direct config entry (check if `providers.gemini.apiKey` exists in the schema):
    ```json
    {
      // ... other configurations ...
      "providers": {
        // ... other providers ...
        "gemini": {
          "apiKey": "AIzaSyYOUR_GEMINI_API_KEY", // Replace with your actual key
          "disabled": false
        }
      },
      // ... rest of the configuration ...
    }
    ```
2.  **Select a Gemini Model:** In the `agents` section, specify the Gemini model:
    ```json
    {
      // ... other configurations ...
      "agents": {
        "coder": {
          "model": "gemini-1.5-flash", // Or other models like gemini-pro, etc.
          "maxTokens": 4000
        }
        // ... other agents ...
      },
      // ... rest of the configuration ...
    }
    ```

After configuring your chosen provider and model, OpenCode's AI agents (both the Orchestrator's main agent and any worker agents, depending on which agent configuration you modify) will use that LLM for their operations. You interact with OpenCode as usual via its CLI, and it handles the communication with the configured LLM provider in the background.

### Configuration File Structure

```json
{
  "data": {
    "directory": ".opencode"
  },
  "providers": {
    "openai": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "anthropic": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "groq": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "openrouter": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "gemini": { // Added Gemini example
      "apiKey": "your-gemini-key-if-not-using-env-var",
      "disabled": false
    }
  },
  "agents": {
    "coder": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 5000
    },
    "task": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 5000
    },
    "title": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 80
    }
  },
  "shell": {
    "path": "/bin/bash",
    "args": ["-l"]
  },
  "mcpServers": {
    "example": {
      "type": "stdio",
      "command": "path/to/mcp-server",
      "env": [],
      "args": []
    }
  },
  "lsp": {
    "go": {
      "disabled": false,
      "command": "gopls"
    }
  },
  "debug": false,
  "debugLSP": false,
  "autoCompact": true
}
```

## Supported AI Models

OpenCode supports a variety of AI models from different providers:

### OpenAI

- GPT-4.1 family (gpt-4.1, gpt-4.1-mini, gpt-4.1-nano)
- GPT-4.5 Preview
- GPT-4o family (gpt-4o, gpt-4o-mini)
- O1 family (o1, o1-pro, o1-mini)
- O3 family (o3, o3-mini)
- O4 Mini

### Anthropic

- Claude 4 Sonnet
- Claude 4 Opus
- Claude 3.5 Sonnet
- Claude 3.5 Haiku
- Claude 3.7 Sonnet
- Claude 3 Haiku
- Claude 3 Opus

### Google

- Gemini 2.5
- Gemini 2.5 Flash
- Gemini 2.0 Flash
- Gemini 2.0 Flash Lite

### AWS Bedrock

- Claude 3.7 Sonnet

### Groq

- Llama 4 Maverick (17b-128e-instruct)
- Llama 4 Scout (17b-16e-instruct)
- QWEN QWQ-32b
- Deepseek R1 distill Llama 70b
- Llama 3.3 70b Versatile

### Azure OpenAI

- GPT-4.1 family (gpt-4.1, gpt-4.1-mini, gpt-4.1-nano)
- GPT-4.5 Preview
- GPT-4o family (gpt-4o, gpt-4o-mini)
- O1 family (o1, o1-mini)
- O3 family (o3, o3-mini)
- O4 Mini

### Google Cloud VertexAI

- Gemini 2.5
- Gemini 2.5 Flash

## General Usage

```bash
# Start OpenCode in Orchestrator CLI mode
opencode

# Start with debug logging
opencode -d

# Start with a specific working directory
opencode -c /path/to/project
```

## Orchestrator CLI Mode

When you run `opencode` without special flags (like `-p` for non-interactive or `--worker-mode`), it starts in Orchestrator CLI mode. This mode allows you to interact directly with OpenCode's primary AI agent and manage worker agents.

**Commands:**

The Orchestrator CLI accepts the following commands:

*   `<prompt>`: Any text not starting with `/` is treated as a prompt for the Orchestrator's own AI agent.
*   `/quit`: Exits the OpenCode application.
*   `/spawn <task_prompt>`: Spawns a new worker agent with the given `<task_prompt>`. The Orchestrator will output a Worker ID upon successful spawning. This feature is currently basic and under development.
*   `/result <workerID>`: Retrieves and displays the result from a worker agent that has completed its task and reported back.

**Inter-Agent Communication (Internal):**

Worker agents communicate their results back to the Orchestrator via an internal HTTP API (typically on endpoint `/report_result`). This API is for internal system use and is not a user-facing LLM API. Notifications from workers (e.g., task completion) will appear asynchronously in the Orchestrator's CLI.

## Non-interactive Prompt Mode

You can run OpenCode in non-interactive mode by passing a prompt directly as a command-line argument. This is useful for scripting, automation, or when you want a quick answer without launching the full TUI.

```bash
# Run a single prompt and print the AI's response to the terminal
opencode -p "Explain the use of context in Go"

# Get response in JSON format
opencode -p "Explain the use of context in Go" -f json

# Run without showing the spinner (useful for scripts)
opencode -p "Explain the use of context in Go" -q
```

In this mode, OpenCode will process your prompt, print the result to standard output, and then exit. All permissions are auto-approved for the session.

By default, a spinner animation is displayed while the model is processing your query. You can disable this spinner with the `-q` or `--quiet` flag, which is particularly useful when running OpenCode from scripts or automated workflows.

### Output Formats

OpenCode supports the following output formats in non-interactive mode:

| Format | Description                     |
| ------ | ------------------------------- |
| `text` | Plain text output (default)     |
| `json` | Output wrapped in a JSON object |

The output format is implemented as a strongly-typed `OutputFormat` in the codebase, ensuring type safety and validation when processing outputs.

## Worker Mode

OpenCode can be launched in a headless "Worker Mode" to perform tasks autonomously as directed by an Orchestrator instance. In this mode, OpenCode does not present an interactive CLI but instead executes a task defined in a specified file and reports its results back to the Orchestrator.

To run OpenCode in worker mode, use the `--worker-mode` (or `-w`) flag. Additional flags are required to configure the worker:

*   `--agent-id <ID>`: (Required) A unique identifier assigned to this worker agent by the Orchestrator.
*   `--task-id <ID>`: (Required) A unique identifier for the task this worker is to perform. This ID is used for correlating tasks and results.
*   `--task-file <path>`: (Required) Path to a JSON file containing the task definition (e.g., `{"task_prompt": "your task description here...", "task_id": "actual_task_id_should_match_flag"}`). The worker reads its instructions from this file.
*   `--orchestrator-api <url>`: (Required) The URL of the Orchestrator's API endpoint (e.g., `http://localhost:12345/report_result`) to which the worker will send its results.

**Example Usage (typically launched by an Orchestrator):**
```bash
opencode --worker-mode \
          --agent-id "worker-007" \
          --task-id "task-123" \
          --task-file "/tmp/task-123.json" \
          --orchestrator-api "http://localhost:12345/report_result"
```
In worker mode, all necessary tool permissions are automatically approved to ensure autonomous operation. LSP and MCP services are typically not initialized to keep workers lightweight.

## Command-line Flags

| Flag              | Short | Description                                         |
| ----------------- | ----- | --------------------------------------------------- |
| `--help`          | `-h`  | Display help information                            |
| `--debug`         | `-d`  | Enable debug mode                                   |
| `--cwd`           | `-c`  | Set current working directory                       |
| `--prompt`        | `-p`  | Run a single prompt in non-interactive mode         |
| `--output-format` | `-f`  | Output format for non-interactive mode (text, json)                 |
| `--quiet`         | `-q`  | Hide spinner in non-interactive mode                                |
| `--worker-mode`      | `-w`  | Run in worker agent mode.                                                   |
| `--agent-id`         |       | Unique ID for the worker agent (used with `--worker-mode`).                 |
| `--task-id`          |       | Unique ID for the task assigned to the worker (used with `--worker-mode`).    |
| `--task-file`        |       | Path to JSON file defining the task for the worker (used with `--worker-mode`). |
| `--orchestrator-api` |       | API endpoint of the orchestrator for reporting (used with `--worker-mode`).  |

## AI Assistant Tools

OpenCode's AI assistant has access to various tools to help with coding tasks:

### File and Code Tools

| Tool          | Description                 | Parameters                                                                               |
| ------------- | --------------------------- | ---------------------------------------------------------------------------------------- |
| `glob`        | Find files by pattern       | `pattern` (required), `path` (optional)                                                  |
| `grep`        | Search file contents        | `pattern` (required), `path` (optional), `include` (optional), `literal_text` (optional) |
| `ls`          | List directory contents     | `path` (optional), `ignore` (optional array of patterns)                                 |
| `view`        | View file contents          | `file_path` (required), `offset` (optional), `limit` (optional)                          |
| `write`       | Write to files              | `file_path` (required), `content` (required)                                             |
| `edit`        | Edit files                  | Various parameters for file editing                                                      |
| `patch`       | Apply patches to files      | `file_path` (required), `diff` (required)                                                |
| `diagnostics` | Get diagnostics information | `file_path` (optional)                                                                   |

### Other Tools

| Tool          | Description                            | Parameters                                                                                |
| ------------- | -------------------------------------- | ----------------------------------------------------------------------------------------- |
| `bash`        | Execute shell commands                 | `command` (required), `timeout` (optional)                                                |
| `fetch`       | Fetch data from URLs                   | `url` (required), `format` (required), `timeout` (optional)                               |
| `sourcegraph` | Search code across public repositories | `query` (required), `count` (optional), `context_window` (optional), `timeout` (optional) |
| `agent`       | Run sub-tasks with the AI agent        | `prompt` (required)                                                                       |

## Architecture

OpenCode is built with a modular architecture:

- **cmd**: Command-line interface using Cobra
- **internal/app**: Core application services
- **internal/config**: Configuration management
- **internal/db**: Database operations and migrations
- **internal/llm**: LLM providers and tools integration
- **internal/tui**: (Legacy) Terminal UI components and layouts. No longer primary interaction for Orchestrator.
- **internal/orchestrator**: Manages worker agents and their communication.
- **internal/app/cli.go**: Implements the Orchestrator CLI.
- **internal/logging**: Logging infrastructure
- **internal/message**: Message handling
- **internal/session**: Session management
- **internal/lsp**: Language Server Protocol integration

## Custom Commands

OpenCode supports custom commands that can be created by users to quickly send predefined prompts to the AI assistant.

### Creating Custom Commands

Custom commands are predefined prompts stored as Markdown files in one of three locations:

1. **User Commands** (prefixed with `user:`):

   ```
   $XDG_CONFIG_HOME/opencode/commands/
   ```

   (typically `~/.config/opencode/commands/` on Linux/macOS)

   or

   ```
   $HOME/.opencode/commands/
   ```

2. **Project Commands** (prefixed with `project:`):

   ```
   <PROJECT DIR>/.opencode/commands/
   ```

Each `.md` file in these directories becomes a custom command. The file name (without extension) becomes the command ID.

For example, creating a file at `~/.config/opencode/commands/prime-context.md` with content:

```markdown
RUN git ls-files
READ README.md
```

This creates a command called `user:prime-context`.

### Command Arguments

OpenCode supports named arguments in custom commands using placeholders in the format `$NAME` (where NAME consists of uppercase letters, numbers, and underscores, and must start with a letter).

For example:

```markdown
# Fetch Context for Issue $ISSUE_NUMBER

RUN gh issue view $ISSUE_NUMBER --json title,body,comments
RUN git grep --author="$AUTHOR_NAME" -n .
RUN grep -R "$SEARCH_PATTERN" $DIRECTORY
```

When you run a command with arguments, OpenCode will prompt you to enter values for each unique placeholder. Named arguments provide several benefits:

- Clear identification of what each argument represents
- Ability to use the same argument multiple times
- Better organization for commands with multiple inputs

### Organizing Commands

You can organize commands in subdirectories:

```
~/.config/opencode/commands/git/commit.md
```

This creates a command with ID `user:git:commit`.

### Using Custom Commands

1. Press `Ctrl+K` to open the command dialog
2. Select your custom command (prefixed with either `user:` or `project:`)
3. Press Enter to execute the command

The content of the command file will be sent as a message to the AI assistant.

### Built-in Commands

OpenCode includes several built-in commands:

| Command            | Description                                                                                         |
| ------------------ | --------------------------------------------------------------------------------------------------- |
| Initialize Project | Creates or updates the OpenCode.md memory file with project-specific information                    |
| Compact Session    | Manually triggers the summarization of the current session, creating a new session with the summary |

## MCP (Model Context Protocol)

OpenCode implements the Model Context Protocol (MCP) to extend its capabilities through external tools. MCP provides a standardized way for the AI assistant to interact with external services and tools.

### MCP Features

- **External Tool Integration**: Connect to external tools and services via a standardized protocol
- **Tool Discovery**: Automatically discover available tools from MCP servers
- **Multiple Connection Types**:
  - **Stdio**: Communicate with tools via standard input/output
  - **SSE**: Communicate with tools via Server-Sent Events
- **Security**: Permission system for controlling access to MCP tools

### Configuring MCP Servers

MCP servers are defined in the configuration file under the `mcpServers` section:

```json
{
  "mcpServers": {
    "example": {
      "type": "stdio",
      "command": "path/to/mcp-server",
      "env": [],
      "args": []
    },
    "web-example": {
      "type": "sse",
      "url": "https://example.com/mcp",
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  }
}
```

### MCP Tool Usage

Once configured, MCP tools are automatically available to the AI assistant alongside built-in tools. They follow the same permission model as other tools, requiring user approval before execution.

## LSP (Language Server Protocol)

OpenCode integrates with Language Server Protocol to provide code intelligence features across multiple programming languages.

### LSP Features

- **Multi-language Support**: Connect to language servers for different programming languages
- **Diagnostics**: Receive error checking and linting information
- **File Watching**: Automatically notify language servers of file changes

### Configuring LSP

Language servers are configured in the configuration file under the `lsp` section:

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

### LSP Integration with AI

The AI assistant can access LSP features through the `diagnostics` tool, allowing it to:

- Check for errors in your code
- Suggest fixes based on diagnostics

While the LSP client implementation supports the full LSP protocol (including completions, hover, definition, etc.), currently only diagnostics are exposed to the AI assistant.

## Using a self-hosted model provider

OpenCode can also load and use models from a self-hosted (OpenAI-like) provider.
This is useful for developers who want to experiment with custom models.

### Configuring a self-hosted provider

You can use a self-hosted model by setting the `LOCAL_ENDPOINT` environment variable.
This will cause OpenCode to load and use the models from the specified endpoint.

```bash
LOCAL_ENDPOINT=http://localhost:1235/v1
```

### Configuring a self-hosted model

You can also configure a self-hosted model in the configuration file under the `agents` section:

```json
{
  "agents": {
    "coder": {
      "model": "local.granite-3.3-2b-instruct@q8_0",
      "reasoningEffort": "high"
    }
  }
}
```

## Development

### Prerequisites

- Go 1.24.0 or higher

### Building from Source

```bash
# Clone the repository
git clone https://github.com/opencode-ai/opencode.git
cd opencode

# Build
go build -o opencode

# Run
./opencode
```

## Acknowledgments

OpenCode gratefully acknowledges the contributions and support from these key individuals:

- [@isaacphi](https://github.com/isaacphi) - For the [mcp-language-server](https://github.com/isaacphi/mcp-language-server) project which provided the foundation for our LSP client implementation
- [@adamdottv](https://github.com/adamdottv) - For the design direction and UI/UX architecture

Special thanks to the broader open source community whose tools and libraries have made this project possible.

## License

OpenCode is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Here's how you can contribute:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please make sure to update tests as appropriate and follow the existing code style.
