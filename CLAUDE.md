## Project Planning: Motion Canvas AI Agent Backend

- Planning a WebSocket-enabled backend for OpenCode AI agent
- Key requirements:
  * Support HTTP POST for receiving prompts (JSON payload)
  * Implement session management 
  * Integrate CoderAgent for code editing and interaction
  * Enable streaming WebSocket communication to frontend
- Backend will handle:
  * Initializing agent sessions
  * Running agent tasks
  * Maintaining conversation state
  * Streaming responses back to client
- Technology stack considerations:
  * Go backend 
  * WebSocket for real-time communication
  * JSON for prompt/response payloads
- Next steps:
  * Design WebSocket handler
  * Implement session management logic
  * Create agent initialization mechanism
  * Build communication protocol between backend and frontend