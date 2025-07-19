/* @jsxImportSource preact */
import { useState, useRef, useEffect } from 'preact/hooks';

interface Message {
  id: string;
  username: string;
  text: string;
  timestamp: Date;
  type: 'user' | 'system';
}

export function ChatTabConfig() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      username: 'System',
      text: 'Welcome to Motion Canvas Chat! Set your username and start collaborating.',
      timestamp: new Date(),
      type: 'system'
    }
  ]);
  const [username, setUsername] = useState('Anonymous');
  const [currentMessage, setCurrentMessage] = useState('');
  const [isEditingUsername, setIsEditingUsername] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const sendMessage = () => {
    if (currentMessage.trim()) {
      const newMessage: Message = {
        id: Date.now().toString(),
        username,
        text: currentMessage.trim(),
        timestamp: new Date(),
        type: 'user'
      };
      setMessages(prev => [...prev, newMessage]);
      setCurrentMessage('');
    }
  };

  const handleKeyPress = (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const updateUsername = (newUsername: string) => {
    if (newUsername.trim() && newUsername !== username) {
      const oldUsername = username;
      setUsername(newUsername.trim());
      
      // Add system message about username change
      const systemMessage: Message = {
        id: Date.now().toString(),
        username: 'System',
        text: `${oldUsername} changed their name to ${newUsername.trim()}`,
        timestamp: new Date(),
        type: 'system'
      };
      setMessages(prev => [...prev, systemMessage]);
    }
    setIsEditingUsername(false);
  };

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      backgroundColor: 'var(--surface-color)',
      color: 'var(--text-color)',
      fontFamily: 'var(--font-family)'
    }}>
      {/* Header */}
      <div style={{
        padding: '12px',
        borderBottom: '1px solid var(--border-color)',
        backgroundColor: 'var(--surface-variant-color)'
      }}>
        <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '8px' }}>
          Motion Canvas Chat
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary-color)' }}>
            Username:
          </span>
          {isEditingUsername ? (
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername((e.target as HTMLInputElement).value)}
              onBlur={(e) => updateUsername((e.target as HTMLInputElement).value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter') {
                  updateUsername((e.target as HTMLInputElement).value);
                }
              }}
              style={{
                backgroundColor: 'var(--input-background)',
                border: '1px solid var(--border-color)',
                borderRadius: '4px',
                padding: '2px 6px',
                fontSize: '12px',
                color: 'var(--text-color)',
                flex: 1
              }}
              autoFocus
            />
          ) : (
            <span
              onClick={() => setIsEditingUsername(true)}
              style={{
                fontSize: '12px',
                cursor: 'pointer',
                padding: '2px 6px',
                borderRadius: '4px',
                backgroundColor: 'var(--accent-color)',
                color: 'var(--accent-text-color)'
              }}
            >
              {username}
            </span>
          )}
        </div>
      </div>

      {/* Messages */}
      <div style={{
        flex: 1,
        overflowY: 'auto',
        padding: '8px',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px'
      }}>
        {messages.map((message) => (
          <div
            key={message.id}
            style={{
              padding: '8px',
              borderRadius: '8px',
              backgroundColor: message.type === 'system' 
                ? 'var(--warning-color)' 
                : 'var(--surface-variant-color)',
              border: '1px solid var(--border-color)'
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
                color: message.type === 'system' 
                  ? 'var(--warning-text-color)' 
                  : 'var(--accent-color)'
              }}>
                {message.username}
              </span>
              <span style={{
                fontSize: '10px',
                color: 'var(--text-secondary-color)'
              }}>
                {formatTime(message.timestamp)}
              </span>
            </div>
            <div style={{
              fontSize: '12px',
              lineHeight: '1.4',
              whiteSpace: 'pre-wrap',
              color: message.type === 'system' 
                ? 'var(--warning-text-color)' 
                : 'var(--text-color)'
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
        borderTop: '1px solid var(--border-color)',
        backgroundColor: 'var(--surface-variant-color)'
      }}>
        <div style={{ display: 'flex', gap: '8px' }}>
          <textarea
            value={currentMessage}
            onChange={(e) => setCurrentMessage((e.target as HTMLTextAreaElement).value)}
            onKeyPress={handleKeyPress}
            placeholder="Type your message... (Enter to send, Shift+Enter for new line)"
            style={{
              flex: 1,
              minHeight: '36px',
              maxHeight: '120px',
              padding: '8px',
              border: '1px solid var(--border-color)',
              borderRadius: '6px',
              backgroundColor: 'var(--input-background)',
              color: 'var(--text-color)',
              fontSize: '12px',
              fontFamily: 'var(--font-family)',
              resize: 'vertical'
            }}
          />
          <button
            onClick={sendMessage}
            disabled={!currentMessage.trim()}
            style={{
              padding: '8px 16px',
              backgroundColor: currentMessage.trim() 
                ? 'var(--accent-color)' 
                : 'var(--surface-color)',
              color: currentMessage.trim() 
                ? 'var(--accent-text-color)' 
                : 'var(--text-secondary-color)',
              border: '1px solid var(--border-color)',
              borderRadius: '6px',
              cursor: currentMessage.trim() ? 'pointer' : 'not-allowed',
              fontSize: '12px',
              fontWeight: 'bold'
            }}
          >
            Send
          </button>
        </div>
        <div style={{
          fontSize: '10px',
          color: 'var(--text-secondary-color)',
          marginTop: '4px'
        }}>
          Press Enter to send • Shift+Enter for new line
        </div>
      </div>
    </div>
  );
}
