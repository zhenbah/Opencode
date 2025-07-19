Let's create a clean WebSocket-only implementation. Here's my comprehensive plan for refactoring the frontend:

  WebSocket-Only Frontend Integration Plan

  1. Remove Legacy Backend Code

  - Delete all Flask and IBA backend logic
  - Remove backend selection (import.meta.env.BACKEND)
  - Clean up sendMessageToBackend to be WebSocket-only

  2. WebSocket Connection Management

  // New state for persistent WebSocket connection
  const [wsConnection, setWsConnection] = useState<WebSocket | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected');
  const [backendSessionId, setBackendSessionId] = useState<string>(''); // Store backend session for restoration

  3. Connection Lifecycle

  - On Chat Open: Establish persistent WebSocket connection
  - During Chat: Reuse same connection for all messages
  - On Chat Close: Close WebSocket connection
  - On Reconnect: Restore using stored backendSessionId

  4. Message Flow Design

  // Simplified WebSocket-only flow:
  1. User sends message -> Use persistent WebSocket connection
  2. Send JSON message: {type: 'message', prompt: contextualMessage, session_id?: backendSessionId}
  3. Receive streaming responses: session_created, agent_response, agent_done, error
  4. Update UI in real-time
  5. Store session_id for chat restoration

  5. Session Restoration Strategy

  - Frontend Chat ID → Backend Session ID mapping
  - Store in localStorage or component state
  - On chat restoration, send session_id to continue conversation

  6. Key Functions to Implement

  A. Connection Management:
  const connectWebSocket = () => { /* Establish WS connection */ }
  const disconnectWebSocket = () => { /* Clean close */ }
  const sendWebSocketMessage = (message: object) => { /* Send via WS */ }

  B. Message Handlers:
  const handleSessionCreated = (sessionId: string) => { /* Store session */ }
  const handleAgentResponse = (content: string) => { /* Stream response */ }
  const handleAgentDone = () => { /* Complete response */ }
  const handleError = (error: string) => { /* Show error */ }

  C. Updated Core Function:
  const sendMessage = async () => {
    // WebSocket-only implementation
    // Include context (example code + console errors)
    // Send via persistent WebSocket
    // Handle streaming responses
  }

  7. Context Integration

  Keep existing context gathering but send via WebSocket:
  // Build contextual message (same as current)
  const contextualMessage = buildContextMessage(messageText, includeContext, exampleCode, consoleErrors);

  // Send via WebSocket instead of HTTP
  sendWebSocketMessage({
    type: 'message',
    prompt: contextualMessage,
    session_id: backendSessionId || undefined
  });

  8. Session Storage

  // Map frontend chat sessions to backend session IDs
  interface ChatSession {
    id: string; // Frontend chat ID
    backendSessionId: string; // Backend session ID for restoration
    title: string;
    lastMessage: string;
    timestamp: Date;
    messageCount: number;
  }

  9. Error Handling & Reconnection

  - Connection drops → Show connection status
  - Auto-reconnect with session restoration
  - Graceful degradation for connection issues

  10. Removed Code Blocks

  - All Flask backend logic (lines ~396-443)
  - All IBA backend logic (lines ~294-393)
  - Backend selection logic (lines ~261-266, ~560-596)
  - Environment variable detection

  Implementation Benefits:

  ✅ Simpler codebase - Single backend path✅ Real-time streaming - Native WebSocket streaming✅ Persistent connection - Better performance✅
  Session continuity - Restore conversations✅ Cleaner state management - No backend switching logic

  Questions Before Implementation:

  1. WebSocket URL: Should it be ws://localhost:3000 or configurable?
  2. Connection timing: Connect on chat open or component mount?
  3. Reconnection strategy: Automatic retry count/interval?
  4. Session storage: localStorage vs in-memory for session restoration?

  Ready to implement this clean WebSocket-only solution?