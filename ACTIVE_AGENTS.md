# Active Agents

This document tracks active Worker Agents managed by the OpenCode Nexus Orchestrator.

## Agent Schema:

Each active agent can be represented as follows:

*   **Agent ID:** Unique identifier for the agent (e.g., UUID).
*   **Status:** Initializing | Idle | Busy | Error | Terminated
*   **Assigned Task ID:** ID of the task the agent is currently working on (from `TASKS.md`).
*   **Worktree Path:** Filesystem path to the agent's dedicated worktree.
*   **Process ID (PID):** PID of the worker agent process (if applicable).
*   **Spawned Timestamp:** When the agent was launched.
*   **Last Heartbeat:** Timestamp of the last communication received from the agent.

## Current Active Agents:

| Agent ID | Status | Assigned Task ID | Worktree Path | PID | Spawned Timestamp | Last Heartbeat |
|----------|--------|------------------|---------------|-----|-------------------|----------------|
|          |        |                  |               |     |                   |                |

*(This table will be populated and managed by the Orchestrator.)*
