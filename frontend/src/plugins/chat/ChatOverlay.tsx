/* @jsxImportSource preact */
import { useState, useRef, useEffect } from 'preact/hooks';

interface Message {
  id: string;
  username: string;
  text: string;
  timestamp: Date;
  type: 'user' | 'system';
}

interface MessagePayload {
  message: string;
  includeContext?: boolean;
  errorStatus?: 'none' | 'error' | 'warning';
  errorDetails?: string[];
  metadata?: {
    [key: string]: any;
  };
}

interface ChatSession {
  id: string;
  backendSessionId: string; // Backend session ID for restoration
  title: string;
  lastMessage: string;
  timestamp: Date;
  messageCount: number;
}

interface WebSocketMessage {
  type: 'message';
  prompt: string;
  session_id?: string;
}

export function ChatOverlay() {
  const [isVisible, setIsVisible] = useState(false);
  const [showChatList, setShowChatList] = useState(false);
  const [messages, setMessages] = useState<Message[]>([]);
  const [chatSessions, setChatSessions] = useState<ChatSession[]>([]);
  const [currentChatId, setCurrentChatId] = useState<string>('');
  const [username, setUsername] = useState('Anonymous');
  const [currentMessage, setCurrentMessage] = useState('');
  const [isEditingUsername, setIsEditingUsername] = useState(false);
  const [backendSessionId, setBackendSessionId] = useState<string>(''); // Backend session for restoration
  const [exampleCode, setExampleCode] = useState<string>('');
  const [consoleErrors, setConsoleErrors] = useState<string[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // WebSocket connection state
  const [wsConnection, setWsConnection] = useState<WebSocket | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected');

  const toggleChat = () => {
    setIsVisible(!isVisible);
  };

  const toggleChatList = () => {
    setShowChatList(!showChatList);
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Create a new chat session
  const createNewChat = () => {
    const newChatId = `chat_${Date.now()}`;
    setCurrentChatId(newChatId);
    setMessages([]);
    setShowChatList(false);
  };

  // Load a chat session
  const loadChatSession = (chatId: string) => {
    // In a real implementation, you'd load messages from storage
    // For now, we'll just switch to an empty chat
    setCurrentChatId(chatId);
    setMessages([]);
    setShowChatList(false);
  };

  // Update chat session when messages change
  useEffect(() => {
    if (messages.length > 0 && currentChatId) {
      console.log(`number of messages is now ${messages.length}`);
      const lastUserMessage = messages.filter(m => m.type === 'user').pop();
      const title = lastUserMessage ? 
        (lastUserMessage.text.length > 30 ? 
          lastUserMessage.text.substring(0, 30) + '...' : 
          lastUserMessage.text) : 
        'New Chat';
      
      setChatSessions(prev => {
        const existingIndex = prev.findIndex(session => session.id === currentChatId);
        const updatedSession: ChatSession = {
          id: currentChatId,
          backendSessionId: backendSessionId,
          title,
          lastMessage: messages[messages.length - 1].text,
          timestamp: new Date(),
          messageCount: messages.length
        };
        
        if (existingIndex >= 0) {
          const updated = [...prev];
          updated[existingIndex] = updatedSession;
          return updated;
        } else {
          return [updatedSession, ...prev];
        }
      });
    }
  }, [messages, currentChatId]);

  // Initialize first chat if none exists
  useEffect(() => {
    if (!currentChatId) {
      createNewChat();
    }
  }, []);

  // Fetch example code from file
  const fetchExampleCode = async (): Promise<string> => {
    try {
      const response = await fetch('/src/scenes/example.tsx');
      if (response.ok) {
        const code = await response.text();
        setExampleCode(code);
        return code;
      } else {
        console.warn('Could not fetch example.tsx file');
        return '';
      }
    } catch (error) {
      console.warn('Error fetching example.tsx:', error);
      return '';
    }
  };

  // Fetch example code on component mount
  useEffect(() => {
    fetchExampleCode();
  }, []);

  // Capture console errors and runtime errors
  useEffect(() => {
    const originalConsoleError = console.error;
    const originalConsoleWarn = console.warn;
    
    // Override console.error to capture errors
    console.error = (...args) => {
      const errorMessage = args.map(arg => 
        typeof arg === 'string' ? arg : JSON.stringify(arg)
      ).join(' ');
      
      // Filter for rendering/Motion Canvas related errors
      if (
        errorMessage.includes('kicker') ||
        errorMessage.includes('Scene2D') ||
        errorMessage.includes('runnerFactory') ||
        errorMessage.includes('Generator') ||
        errorMessage.includes('yield') ||
        errorMessage.includes('function') ||
        errorMessage.includes('TypeError') ||
        errorMessage.includes('ReferenceError') ||
        errorMessage.includes('motion-canvas') ||
        errorMessage.includes('.tsx') ||
        errorMessage.includes('example.tsx')
      ) {
        setConsoleErrors(prev => {
          const newErrors = [...prev, `ERROR: ${errorMessage}`].slice(-5); // Keep last 5 errors
          return newErrors;
        });
      }
      // log console error to console
      console.warn('Captured Console Error:', errorMessage);
      // Call original console.error
      originalConsoleError.apply(console, args);
    };

    // Override console.warn for warnings
    console.warn = (...args) => {
      const warnMessage = args.map(arg => 
        typeof arg === 'string' ? arg : JSON.stringify(arg)
      ).join(' ');
      
      if (
        warnMessage.includes('motion-canvas') ||
        warnMessage.includes('Scene2D') ||
        warnMessage.includes('.tsx')
      ) {
        setConsoleErrors(prev => {
          const newErrors = [...prev, `WARN: ${warnMessage}`].slice(-5);
          return newErrors;
        });
      }
      
      originalConsoleWarn.apply(console, args);
    };

    // Capture unhandled errors
    const handleError = (event: ErrorEvent) => {
      const errorMessage = `${event.message} at ${event.filename}:${event.lineno}:${event.colno}`;
      if (
        errorMessage.includes('kicker') ||
        errorMessage.includes('Scene2D') ||
        errorMessage.includes('motion-canvas') ||
        errorMessage.includes('.tsx')
      ) {
        setConsoleErrors(prev => {
          const newErrors = [...prev, `RUNTIME ERROR: ${errorMessage}`].slice(-5);
          return newErrors;
        });
      }
    };

    // Capture unhandled promise rejections
    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      const errorMessage = `Unhandled Promise Rejection: ${event.reason}`;
      setConsoleErrors(prev => {
        const newErrors = [...prev, errorMessage].slice(-5);
        return newErrors;
      });
    };

    window.addEventListener('error', handleError);
    window.addEventListener('unhandledrejection', handleUnhandledRejection);

    return () => {
      // Restore original console methods
      console.error = originalConsoleError;
      console.warn = originalConsoleWarn;
      window.removeEventListener('error', handleError);
      window.removeEventListener('unhandledrejection', handleUnhandledRejection);
    };
  }, []);

  // WebSocket connection management
  const connectWebSocket = () => {
    if (wsConnection) return; // Already connected

    console.log('Connecting to WebSocket...');
    setConnectionStatus('connecting');

    try {
      const ws = new WebSocket('ws://localhost:3000');
      
      ws.onopen = () => {
        console.log('WebSocket connected successfully');
        setConnectionStatus('connected');
        setWsConnection(ws);
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          handleWebSocketMessage(message);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.onclose = (event) => {
        console.log('WebSocket connection closed:', event.code, event.reason);
        setConnectionStatus('disconnected');
        setWsConnection(null);
        
        // Auto-reconnect after 3 seconds if not manually closed
        if (event.code !== 1000) {
          setTimeout(() => {
            if (connectionStatus !== 'connected') {
              connectWebSocket();
            }
          }, 3000);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        setConnectionStatus('error');
      };

    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      setConnectionStatus('error');
    }
  };

  const disconnectWebSocket = () => {
    if (wsConnection) {
      wsConnection.close(1000, 'User closed chat');
      setWsConnection(null);
      setConnectionStatus('disconnected');
    }
  };

  // Handle incoming WebSocket messages
  const handleWebSocketMessage = (message: any) => {
    console.log('Received WebSocket message:', message);

    switch (message.type) {
      case 'session_created':
        setBackendSessionId(message.session_id);
        console.log('Backend session created:', message.session_id);
        break;

      case 'agent_response':
        // Stream response into current assistant message
        const content = message.content || '';
        setMessages(prev => {
          const lastMessage = prev[prev.length - 1];
          if (lastMessage && lastMessage.username === 'Assistant') {
            // Update existing assistant message
            const updated = [...prev];
            updated[updated.length - 1] = {
              ...lastMessage,
              text: lastMessage.text + content
            };
            return updated;
          } else {
            // Create new assistant message
            const assistantMessage: Message = {
              id: (Date.now()).toString(),
              username: 'Assistant',
              text: content,
              timestamp: new Date(),
              type: 'system'
            };
            return [...prev, assistantMessage];
          }
        });
        break;

      case 'agent_done':
        console.log('Agent processing completed for session:', message.session_id);
        // Remove any connecting status messages
        setMessages(prev => prev.filter(msg => !msg.text.includes('Sending message to server')));
        break;

      case 'error':
        console.error('Backend error:', message.error);
        setMessages(prev => [
          ...prev.filter(msg => !msg.text.includes('Sending message to server')),
          {
            id: Date.now().toString(),
            username: 'System',
            text: `Error: ${message.error}`,
            timestamp: new Date(),
            type: 'system'
          }
        ]);
        break;

      default:
        console.warn('Unknown message type:', message.type);
    }
  };

  // Send message via WebSocket (replaces old backend logic)
  const sendMessageToBackend = async (payload: MessagePayload) => {
    const { message: messageText, includeContext = false, errorStatus = 'none', errorDetails = [] } = payload;

    // Add user message to chat
    const userMessage: Message = {
      id: Date.now().toString(),
      username,
      text: messageText,
      timestamp: new Date(),
      type: 'user'
    };
    setMessages(prev => [...prev, userMessage]);

    // Ensure WebSocket connection
    if (!wsConnection || wsConnection.readyState !== WebSocket.OPEN) {
      const connectingMessage: Message = {
        id: (Date.now() + 1).toString(),
        username: 'System',
        text: 'Connecting to server...',
        timestamp: new Date(),
        type: 'system'
      };
      setMessages(prev => [...prev, connectingMessage]);
      
      // Try to connect and wait briefly
      connectWebSocket();
      
      // Wait for connection or timeout
      let attempts = 0;
      while ((!wsConnection || wsConnection.readyState !== WebSocket.OPEN) && attempts < 10) {
        await new Promise(resolve => setTimeout(resolve, 500));
        attempts++;
      }
      
      // Remove connecting message
      setMessages(prev => prev.filter(msg => msg.id !== connectingMessage.id));
      
      if (!wsConnection || wsConnection.readyState !== WebSocket.OPEN) {
        setMessages(prev => [...prev, {
          id: Date.now().toString(),
          username: 'System',
          text: 'Failed to connect to server. Please check if the backend is running on localhost:3000.',
          timestamp: new Date(),
          type: 'system'
        }]);
        return;
      }
    }

    // Add processing message
    const statusMessage: Message = {
      id: (Date.now() + 2).toString(),
      username: 'System',
      text: 'Sending message to server...',
      timestamp: new Date(),
      type: 'system'
    };
    setMessages(prev => [...prev, statusMessage]);

    // Fetch the latest example code if context is needed
    const contextCode = includeContext ? await fetchExampleCode() : exampleCode;

    // Prepare contextual message
    let contextualMessage = '';
    
    // Add current code context
    if (includeContext && contextCode) {
      contextualMessage += `Context - Current example.tsx content:\n\`\`\`typescript\n${contextCode}\n\`\`\`\n\n`;
    }
    
    // Add recent console errors if any and context is needed
    if (includeContext && consoleErrors.length > 0) {
      contextualMessage += `Recent Console Errors/Warnings:\n\`\`\`\n${consoleErrors.join('\n')}\n\`\`\`\n\n`;
    }
    
    // Add error details from payload if provided
    if (errorDetails.length > 0) {
      contextualMessage += `Error Details (${errorStatus}):\n\`\`\`\n${errorDetails.join('\n')}\n\`\`\`\n\n`;
    }
    
    // Add user request
    contextualMessage += `User request: ${messageText}`;
    
    // If no context, just use the message
    if (!includeContext || (!contextCode && consoleErrors.length === 0 && errorDetails.length === 0)) {
      contextualMessage = messageText;
    }

    // Send via WebSocket with HTTP POST to /generateScene
    try {
      const response = await fetch('http://localhost:3000/generateScene', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ prompt: contextualMessage })
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      // The server will upgrade to WebSocket automatically
      // Remove status message since WebSocket will handle responses
      setMessages(prev => prev.filter(msg => msg.id !== statusMessage.id));

    } catch (error) {
      console.error('Error sending message:', error);
      setMessages(prev => [
        ...prev.filter(msg => msg.id !== statusMessage.id),
        {
          id: Date.now().toString(),
          username: 'System',
          text: `Error: ${error instanceof Error ? error.message : 'Failed to send message'}. Is the server running?`,
          timestamp: new Date(),
          type: 'system'
        }
      ]);
    }
  };
  
  const sendMessage = async () => {
    if (!currentMessage.trim()) return;
    
    const messageText = currentMessage.trim();
    setCurrentMessage(''); // Clear input immediately
    
    await sendMessageToBackend({ message: messageText });
  };

  const resolveErrors = async () => {
    if (consoleErrors.length === 0) return;

    // Compose an error resolution request message
    const errorResolutionMessage = `Please fix the following errors in my Motion Canvas code:

Errors encountered:
${consoleErrors.join('\n')}

Current code in example.tsx:
\`\`\`typescript
${exampleCode}
\`\`\`

Please provide the corrected code that resolves these errors.`;

    // Clear the console errors since we're addressing them
    setConsoleErrors([]);

    // Send the error resolution request
    await sendMessageToBackend({
      message: errorResolutionMessage,
      includeContext: false,
      errorStatus: 'error',
      errorDetails: consoleErrors
    });
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    // Stop propagation to prevent Motion Canvas from receiving the event
    e.stopPropagation();
    
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
    // Allow other keys (including Tab) to work normally in the textarea
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const formatDate = (date: Date) => {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const messageDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
    
    if (messageDate.getTime() === today.getTime()) {
      return formatTime(date);
    } else {
      return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    }
  };

  const updateUsername = (newUsername: string) => {
    if (newUsername.trim() && newUsername !== username) {
      setUsername(newUsername.trim());
    }
    setIsEditingUsername(false);
  };

  const clearAllChats = async () => {
    // Clear chat messages
    setMessages([]);
    
    // Clear console errors
    setConsoleErrors([]);
    
    // Default Motion Canvas code - exact format as requested
    const defaultCode = `import {makeScene2D} from '@motion-canvas/2d';
export default makeScene2D(function* (view) { view.fill('#000000'); });`;

    try {
      // Try to write directly to the file
      const response = await fetch('/frontend/src/scenes/example.tsx', {
        method: 'PUT',
        headers: {
          'Content-Type': 'text/plain',
        },
        body: defaultCode
      });

      if (response.ok) {
        // Update local state
        setExampleCode(defaultCode);
        
        // Add success message
        const successMessage: Message = {
          id: Date.now().toString(),
          username: 'System',
          text: 'Chat cleared and example.tsx automatically reset to default code.',
          timestamp: new Date(),
          type: 'system'
        };
        setMessages([successMessage]);
      } else {
        // If direct write fails, try alternative approach using a different method
        throw new Error('Direct file write not supported');
      }
    } catch (error) {
      console.warn('Direct file write failed, trying backend approach:', error);
      
      // Try to use the backend to write the file if available
      try {
        // Get backend type
        let backend = '';
        if (import.meta.env.BACKEND) {
          backend = import.meta.env.BACKEND;
        } else {
          backend = 'FLASK';
        }

        if (backend === 'FLASK') {
          // Try Flask backend to write the file
          const response = await fetch('http://localhost:8000/write_code', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({ 
              prompt: `Please replace the entire content of example.tsx with exactly this code:

${defaultCode}

This is a reset request - overwrite everything in the file with just this code.`
            })
          });

          if (response.ok) {
            setExampleCode(defaultCode);
            const successMessage: Message = {
              id: Date.now().toString(),
              username: 'System',
              text: 'Chat cleared and example.tsx reset via backend.',
              timestamp: new Date(),
              type: 'system'
            };
            setMessages([successMessage]);
            return;
          }
        }
        
        // If backend fails or not available, fall back to manual instruction
        throw new Error('Backend write failed');
        
      } catch (backendError) {
        // Final fallback: Update local state and inform user
        setExampleCode(defaultCode);
        
        const warningMessage: Message = {
          id: Date.now().toString(),
          username: 'System',
          text: `Chat cleared and context reset. Please manually replace the entire content of example.tsx with:

\`\`\`typescript
${defaultCode}
\`\`\``,
          timestamp: new Date(),
          type: 'system'
        };
        setMessages([warningMessage]);
      }
    }
  };

  if (!isVisible) {
    return (
      <div style={{
        position: 'fixed',
        bottom: '130px',
        left: '10px',
        zIndex: 9999
      }}>
        <button
          onClick={toggleChat}
          style={{
            width: '33px',
            height: '33px',
            borderRadius: '50%',
            backgroundColor: '#007acc',
            border: 'none',
            color: 'white',
            fontSize: '14px',
            cursor: 'pointer',
            boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}
          title="Open Chat"
        >
          💬
        </button>
      </div>
    );
  }

  // Chat List Screen
  if (showChatList) {
    return (
      <div 
        style={{
          position: 'fixed',
          bottom: '20px',
          left: '20px',
          width: '350px',
          height: '700px',
          backgroundColor: '#1e1e1e',
          color: '#ffffff',
          borderRadius: '12px',
          border: '1px solid #444',
          display: 'flex',
          flexDirection: 'column',
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
          boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
          zIndex: 9999
        }}
        onKeyDown={(e) => e.stopPropagation()}
        onKeyUp={(e) => e.stopPropagation()}
        onKeyPress={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div style={{
          padding: '16px',
          borderBottom: '1px solid #444',
          backgroundColor: '#2d2d2d',
          borderTopLeftRadius: '12px',
          borderTopRightRadius: '12px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 'bold', marginBottom: '4px' }}>
              Chat History
            </div>
            <div style={{ fontSize: '12px', color: '#888' }}>
              {chatSessions.length} conversation{chatSessions.length !== 1 ? 's' : ''}
            </div>
          </div>
          <div style={{ display: 'flex', gap: '4px' }}>
            <button
              onClick={toggleChatList}
              style={{
                background: 'none',
                border: 'none',
                color: '#888',
                fontSize: '16px',
                cursor: 'pointer',
                padding: '4px'
              }}
              title="Back to Chat"
            >
              ← 
            </button>
            <button
              onClick={toggleChat}
              style={{
                background: 'none',
                border: 'none',
                color: '#888',
                fontSize: '18px',
                cursor: 'pointer',
                padding: '4px'
              }}
              title="Close Chat"
            >
              ✕
            </button>
          </div>
        </div>

        {/* New Chat Button */}
        <div style={{ padding: '12px', borderBottom: '1px solid #444' }}>
          <button
            onClick={createNewChat}
            style={{
              width: '100%',
              padding: '12px',
              backgroundColor: '#007acc',
              color: '#fff',
              border: 'none',
              borderRadius: '6px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: 'bold',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px'
            }}
          >
            + New Chat
          </button>
        </div>

        {/* Chat Sessions List */}
        <div style={{
          flex: 1,
          overflowY: 'auto',
          padding: '8px'
        }}>
          {chatSessions.length === 0 ? (
            <div style={{
              padding: '20px',
              textAlign: 'center',
              color: '#888',
              fontSize: '14px'
            }}>
              No chat history yet.<br />
              Start a new conversation!
            </div>
          ) : (
            chatSessions.map((session) => (
              <div
                key={session.id}
                onClick={() => loadChatSession(session.id)}
                style={{
                  padding: '12px',
                  marginBottom: '8px',
                  backgroundColor: session.id === currentChatId ? '#333' : '#2a2a2a',
                  border: session.id === currentChatId ? '1px solid #007acc' : '1px solid #444',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  transition: 'background-color 0.2s',
                }}
                onMouseEnter={(e) => {
                  if (session.id !== currentChatId) {
                    (e.target as HTMLElement).style.backgroundColor = '#333';
                  }
                }}
                onMouseLeave={(e) => {
                  if (session.id !== currentChatId) {
                    (e.target as HTMLElement).style.backgroundColor = '#2a2a2a';
                  }
                }}
              >
                <div style={{
                  fontSize: '14px',
                  fontWeight: 'bold',
                  marginBottom: '4px',
                  color: '#fff'
                }}>
                  {session.title}
                </div>
                <div style={{
                  fontSize: '12px',
                  color: '#888',
                  marginBottom: '4px',
                  display: '-webkit-box',
                  WebkitLineClamp: 2,
                  WebkitBoxOrient: 'vertical',
                  overflow: 'hidden'
                }}>
                  {session.lastMessage}
                </div>
                <div style={{
                  fontSize: '10px',
                  color: '#666',
                  display: 'flex',
                  justifyContent: 'space-between'
                }}>
                  <span>{formatDate(session.timestamp)}</span>
                  <span>{session.messageCount} message{session.messageCount !== 1 ? 's' : ''}</span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    );
  }

  // Main Chat Screen
  return (
    <div 
      style={{
        position: 'fixed',
        bottom: '20px',
        left: '20px',
        width: '350px',
        height: '700px',
        backgroundColor: '#1e1e1e',
        color: '#ffffff',
        borderRadius: '12px',
        border: '1px solid #444',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
        boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
        zIndex: 9999
      }}
      onKeyDown={(e) => e.stopPropagation()}
      onKeyUp={(e) => e.stopPropagation()}
      onKeyPress={(e) => e.stopPropagation()}
    >
      {/* Header */}
      <div style={{
        padding: '16px',
        borderBottom: '1px solid #444',
        backgroundColor: '#2d2d2d',
        borderTopLeftRadius: '12px',
        borderTopRightRadius: '12px',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}>
        <div>
          <div style={{ fontSize: '16px', fontWeight: 'bold', marginBottom: '4px' }}>
            Motion Canvas Chat
          </div>
          <div style={{ fontSize: '12px', color: '#ffffff', backgroundColor: '#007acc', padding: '4px 8px', borderRadius: '4px' }}>
            Describe your motion object/scene <br /> 
            {consoleErrors.length > 0 && (
              <span style={{ color: '#ffcc00' }}> • {consoleErrors.length} error(s)</span>
            )}
          </div>
          {consoleErrors.length > 0 && (
            <button
              onClick={resolveErrors}
              style={{
                fontSize: '10px',
                backgroundColor: '#ff6b6b',
                color: '#fff',
                border: 'none',
                borderRadius: '3px',
                padding: '2px 6px',
                marginTop: '4px',
                cursor: 'pointer',
                fontWeight: 'bold'
              }}
              title="Automatically send errors to AI for resolution"
            >
              Resolve Error
            </button>
          )}
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          <button
            onClick={toggleChat}
            style={{
              background: 'none',
              border: 'none',
              color: '#888',
              fontSize: '18px',
              cursor: 'pointer',
              padding: '4px'
            }}
            title="Close Chat"
          >
            ✕
          </button>
          <div style={{ display: 'flex', gap: '4px' }}>
            <button
              onClick={toggleChatList}
              style={{
                background: 'none',
                border: '1px solid #666',
                color: '#007acc',
                fontSize: '10px',
                cursor: 'pointer',
                padding: '4px 6px',
                borderRadius: '4px',
                fontWeight: 'bold'
              }}
              title="View chat history"
            >
              ☰ LIST
            </button>
            <button
              onClick={clearAllChats}
              style={{
                background: 'none',
                border: '1px solid #666',
                color: '#ff6b6b',
                fontSize: '10px',
                cursor: 'pointer',
                padding: '4px 6px',
                borderRadius: '4px',
                fontWeight: 'bold'
              }}
              title="Clear all chats and reset example.tsx"
            >
              CLEAR ALL
            </button>
          </div>
        </div>
      </div>

      {/* Messages */}
      <div style={{
        flex: 1,
        overflowY: 'auto',
        padding: '12px',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px'
      }}>
        {messages.map((message) => (
          <div
            key={message.id}
            style={{
              padding: '8px 12px',
              borderRadius: '8px',
              backgroundColor: message.type === 'system' ? '#2d4a2d' : '#333',
              border: message.type === 'system' ? '1px solid #4a7c4a' : '1px solid #555'
            }}
          >
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '4px'
            }}>
              <span style={{
                fontSize: '12px',
                fontWeight: 'bold',
                color: message.type === 'system' ? '#7fdf7f' : '#007acc'
              }}>
                {message.username}
              </span>
              <span style={{
                fontSize: '10px',
                color: '#888'
              }}>
                {formatTime(message.timestamp)}
              </span>
            </div>
            <div style={{
              fontSize: '13px',
              lineHeight: '1.4',
              whiteSpace: 'pre-wrap',
              color: '#ffffff'
            }}>
              {message.text}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div style={{
        padding: '12px',
        borderTop: '1px solid #444',
        backgroundColor: '#2d2d2d',
        borderBottomLeftRadius: '12px',
        borderBottomRightRadius: '12px'
      }}>
        <div style={{ display: 'flex', gap: '8px' }}>
          <textarea
            value={currentMessage}
            onChange={(e) => setCurrentMessage((e.target as HTMLTextAreaElement).value)}
            onKeyDown={handleKeyDown}
            placeholder="Type your message..."
            style={{
              flex: 1,
              minHeight: '36px',
              maxHeight: '80px',
              padding: '8px',
              border: '1px solid #666',
              borderRadius: '6px',
              backgroundColor: '#444',
              color: '#fff',
              fontSize: '13px',
              fontFamily: 'inherit',
              resize: 'vertical'
            }}
          />
          <button
            onClick={sendMessage}
            disabled={!currentMessage.trim()}
            style={{
              padding: '8px 16px',
              backgroundColor: currentMessage.trim() 
                ? '#007acc' 
                : '#555',
              color: '#fff',
              border: 'none',
              borderRadius: '6px',
              cursor: currentMessage.trim() ? 'pointer' : 'not-allowed',
              fontSize: '13px',
              fontWeight: 'bold'
            }}
          >
            Send
          </button>
        </div>
        <div style={{
          fontSize: '10px',
          color: '#888',
          marginTop: '4px'
        }}>
          Enter to send • Shift+Enter for new line
        </div>
      </div>
    </div>
  );
}