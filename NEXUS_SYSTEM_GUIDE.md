# Nexus System Guide

## 1. Overview

This document outlines the architecture, protocols, and operational procedures for the OpenCode Nexus multi-agent system. The system consists of an Orchestrator (a primary OpenCode instance) and one or more Worker Agents (specialized OpenCode instances). The Orchestrator manages the overall workflow, decomposes tasks, assigns them to Worker Agents, and integrates their results. Worker Agents execute tasks in isolated environments.

## 2. Inter-Agent Communication Protocol

Communication between the Orchestrator and Worker Agents is crucial.

### 2.1. Transport Mechanism

*   **Orchestrator to Worker (Initial Setup & Task Assignment):**
    *   When a Worker Agent is spawned, the Orchestrator will launch it as a new process (e.g., `opencode --worker-mode --config <path_to_worker_config>`).
    *   Initial tasking and context will be provided via command-line arguments or a temporary configuration file passed to the worker.
*   **Worker to Orchestrator (Status Updates, Results, Prompts):**
    *   Worker Agents will communicate back to the Orchestrator using a dedicated API endpoint exposed by the Orchestrator (e.g., a local HTTP server or gRPC).
    *   Alternatively, for simpler status updates, a system of watched files or named pipes could be used. (To be finalized based on implementation complexity).
*   **Orchestrator to Worker (Ongoing Interaction/Interruption):**
    *   If the Orchestrator needs to send new information or interrupt a worker, it can use OS signals (e.g., SIGUSR1) to make the worker poll a specific location for new instructions, or use the same API endpoint if the worker also exposes one.

### 2.2. Message Formats

*   All messages exchanged between agents should be in JSON format.
*   **Common Message Structure:**
    ```json
    {
      "message_id": "uuid",
      "timestamp": "iso_8601_datetime",
      "source_agent_id": "string (orchestrator|worker_uuid)",
      "target_agent_id": "string (orchestrator|worker_uuid)",
      "type": "task_assignment | status_update | result | query | error",
      "payload": {
        // Type-specific content
      }
    }
    ```
*   **Task Assignment (`orchestrator` -> `worker`):**
    ```json
    // payload:
    {
      "task_id": "uuid",
      "task_definition": "string (detailed description of the task)",
      "input_data_references": ["path/to/file1", "shared_resource_id"],
      "working_directory": "/path/to/worktree/feature_x", // Worker should operate here
      "required_tools": ["tool_name1", "tool_name2"] // Tools the worker needs
    }
    ```
*   **Status Update (`worker` -> `orchestrator`):**
    ```json
    // payload:
    {
      "task_id": "uuid",
      "status": "received | in_progress | blocked | completed | failed",
      "progress_percentage": "integer (0-100, optional)",
      "details": "string (e.g., 'Compiling code', 'Waiting for resource X')"
    }
    ```
*   **Result (`worker` -> `orchestrator`):**
    ```json
    // payload:
    {
      "task_id": "uuid",
      "status": "completed", // Should always be completed here
      "artifacts": [
        {"type": "file", "path": "/path/to/output/file1.txt"},
        {"type": "stdout", "content": "Output from command"},
        {"type": "diff", "file_path": "/path/to/original/file.go", "diff_content": "unified_diff_format"}
      ],
      "summary": "string (brief summary of results)"
    }
    ```
*   **Error (`worker` -> `orchestrator`):**
    ```json
    // payload:
    {
      "task_id": "uuid (optional, if error relates to a task)",
      "error_code": "string (e.g., 'tool_not_found', 'execution_failed')",
      "message": "string (detailed error message)",
      "stack_trace": "string (optional)"
    }
    ```

### 2.3. Communication Log
    *   All significant inter-agent messages (task assignments, final results, critical errors) MUST be logged by the Orchestrator in `AGENT_COMMUNICATIONS.log`.
    *   Format: `[TIMESTAMP] [SOURCE_AGENT_ID] -> [TARGET_AGENT_ID] [MESSAGE_TYPE]: JSON_PAYLOAD_SUMMARY`

## 3. Resource Locking Protocol

To prevent race conditions when agents access shared resources (primarily files not isolated within their worktree, or shared configuration).

*   A directory named `.nexus_locks/` will be maintained in the project root.
*   To lock a resource (e.g., `shared_config.json`), an agent must create a lock file: `.nexus_locks/shared_config.json.lock`.
    *   The lock file should contain the `agent_id` and a `timestamp`.
*   Before accessing a resource, an agent MUST check for the existence of a corresponding `.lock` file.
    *   If a lock file exists, the agent must wait or handle the conflict according to its task priority and instructions (e.g., notify Orchestrator).
*   Locks should be time-limited to prevent deadlocks. If an agent encounters an old lock file, it may report it to the Orchestrator.
*   Agents MUST release locks by deleting the `.lock` file as soon as they are done with the resource.

## 4. Task Management Protocol

Refer to `TASKS.md` for the structure of task definitions.

*   **Decomposition:** The Orchestrator decomposes high-level user objectives into smaller, manageable tasks.
*   **Assignment:** The Orchestrator assigns tasks to available Worker Agents based on their capabilities (if specialized workers are implemented) or round-robin.
*   **Tracking:** The Orchestrator updates `TASKS.md` and `ACTIVE_AGENTS.md` with task status based on worker reports.
*   **Completion:** When a task is completed, the worker signals the Orchestrator with the results. The Orchestrator verifies and integrates the results.

## 5. Git Worktree Usage Protocol

*   **Orchestrator Responsibility:** The Orchestrator is SOLELY responsible for all `git` operations that affect the main repository branches (e.g., `commit`, `push`, `merge`, `worktree add/remove`).
*   **Worker Isolation:**
    *   Each Worker Agent SHOULD operate within a dedicated Git worktree created by the Orchestrator (e.g., `worktrees/[feature_name]`).
    *   The Orchestrator will `cd` the worker process into this directory upon launch.
    *   Workers perform file modifications *within their assigned worktree only*.
*   **Reporting Changes:** Workers report changes as diffs or paths to modified files within their worktree. The Orchestrator then stages and commits these changes from the main project directory, referencing the correct worktree.

## 6. Agent Lifecycle Management

### 6.1. Worker Agent Spawning (Orchestrator Procedure)

1.  **Create Worktree (if needed):**
    `git worktree add "worktrees/[AGENT_TASK_ID]" -b "agent/[AGENT_TASK_ID]"`
2.  **Determine Worker Configuration:** Prepare a minimal configuration for the worker, specifying its ID, Orchestrator communication endpoint, and initial task.
3.  **Launch Worker Process:**
    `opencode --worker-mode --agent-id [AGENT_ID] --orchestrator-api [API_ENDPOINT] --task-file [PATH_TO_TEMP_TASK_FILE] --cwd worktrees/[AGENT_TASK_ID]`
    *   `--worker-mode`: A new flag to make OpenCode start in a minimal, non-TUI mode.
    *   `--agent-id`: Unique ID for the worker.
    *   `--orchestrator-api`: Endpoint for the worker to report back.
    *   `--task-file`: Path to a temporary JSON file containing the initial `task_assignment` message.
    *   `--cwd`: The working directory for the agent.
4.  **Monitoring:** The Orchestrator will monitor the worker process (e.g., PID) and its communications.
5.  **Record in `ACTIVE_AGENTS.md`**.

### 6.2. Worker Agent Operation

*   Worker agents run in a headless/non-interactive mode. They do not use the standard TUI.
*   They initialize, perform the assigned task using their internal tools (subset of OpenCode tools, focused on file manipulation, code generation, analysis), and communicate progress/results back to the Orchestrator via the defined API.
*   Workers should be designed to be stateless or easily re-initialized with a task.

### 6.3. Worker Agent Cleanup (Orchestrator Procedure)

1.  **Signal Completion/Termination:** Orchestrator may send a shutdown signal or rely on the worker exiting after task completion/failure.
2.  **Process Termination:** Ensure the worker process has exited.
3.  **Integrate Results:** Handle any final outputs.
4.  **Worktree Management:**
    *   If task was successful and merged: `git worktree remove "worktrees/[AGENT_TASK_ID]"` and `git branch -d "agent/[AGENT_TASK_ID]"`.
    *   If task failed or changes are not needed: `git worktree remove -f "worktrees/[AGENT_TASK_ID]"` and `git branch -D "agent/[AGENT_TASK_ID]"`.
5.  **Update `ACTIVE_AGENTS.md`**.

## 7. Worker Agent Interaction (Orchestrator with Headless Workers)

*   **No Direct Tmux:** The complex tmux send-keys sequences from the original prompt are NOT applicable here as workers are not full TUI instances.
*   **API-Driven:** Communication is primarily API-driven (see Section 2).
*   **Initial Prompt:** Delivered via `--task-file` or similar mechanism during launch. The content of this initial prompt is a `task_assignment` message.

## 8. Initial Prompt Content for Worker Agents

The initial prompt is essentially the `task_assignment` message payload (see Section 2.2), provided to the worker agent upon startup. It includes:

*   `task_id`
*   `task_definition`: Clear, natural language description of the task.
*   `input_data_references`: Paths to relevant files or data.
*   `working_directory`: The specific worktree path.
*   `required_tools`: Any specific OpenCode tools the worker is expected to use.
*   Instruction: "You are a Worker Agent for the OpenCode Nexus system. Your designated operational directory is [working_directory]. Ensure all file operations are relative to this path. Communicate status and results to the Orchestrator at [orchestrator_api_endpoint]. Adhere to all protocols in NEXUS_SYSTEM_GUIDE.md."

## 9. Orchestrator Input Handling (User and Agent Prompts)

*   The Orchestrator's main loop will need to listen for:
    *   User input from its TUI (or primary CLI interface).
    *   Incoming messages from Worker Agents (via its API endpoint).
*   A queuing mechanism will be implemented for incoming prompts/messages if the Orchestrator is busy processing.
*   User input MAY have a way to be prioritized (e.g., a specific command or shortcut that attempts to interrupt the current Orchestrator task or jump the queue).

## 10. Shell Escaping

*   Since communication is primarily JSON based and task definitions are strings within JSON, standard JSON string escaping rules apply.
*   If worker agents need to execute shell commands based on Orchestrator instructions, the `command` field in the `bash` tool payload must be carefully constructed by the worker's LLM to be safe. The Orchestrator itself should avoid directly crafting complex shell commands for workers.
