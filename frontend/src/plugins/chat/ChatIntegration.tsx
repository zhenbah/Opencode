/* @jsxImportSource preact */
import { render } from 'preact';
import { ChatOverlay } from './ChatOverlay';

// Chat Integration for Motion Canvas
export function initializeChatIntegration() {
  console.log('initializeChatIntegration called');
  
  // Create a container for the chat overlay
  let chatContainer = document.getElementById('motion-canvas-chat');
  if (!chatContainer) {
    console.log('Creating chat container');
    chatContainer = document.createElement('div');
    chatContainer.id = 'motion-canvas-chat';
    document.body.appendChild(chatContainer);
  } else {
    console.log('Chat container already exists');
  }

  console.log('Rendering ChatOverlay component');
  // Render the chat overlay - let the component manage its own state
  render(<ChatOverlay />, chatContainer);
  
  console.log('Chat overlay rendered successfully');

  return {
    container: chatContainer
  };
}

// Auto-initialize when this module is loaded
let chatInstance: ReturnType<typeof initializeChatIntegration> | null = null;

export function getChatInstance() {
  if (!chatInstance) {
    chatInstance = initializeChatIntegration();
  }
  return chatInstance;
}

// Make it available globally for easy access
if (typeof window !== 'undefined') {
  (window as any).MotionCanvasChat = {
    init: initializeChatIntegration,
    getInstance: getChatInstance
  };
}
