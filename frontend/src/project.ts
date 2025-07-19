import {makeProject} from '@motion-canvas/core';

// Dynamic import wrapper to catch errors
async function loadExampleScene() {
  console.log('Loading example scene...');
  try {
    const module = await import('./scenes/example?scene');
    return module.default;
  } catch (error) {
    console.error('Error importing example scene:', error);
    // Return a fallback scene
    return function* (view) {
      view.fill('#ff0000'); // Red background to indicate error
      yield;
    };
  }
}

const example = await loadExampleScene();

// Initialize chat integration
import { getChatInstance } from './plugins/chat/ChatIntegration';

// Auto-initialize chat when project loads
if (typeof window !== 'undefined') {
  console.log('Motion Canvas project loading, initializing chat...');
  // Delay initialization to ensure DOM is ready
  setTimeout(() => {
    console.log('Initializing chat instance...');
    try {
      const chatInstance = getChatInstance();
      console.log('Chat instance created:', chatInstance);
    } catch (error) {
      console.error('Error initializing chat:', error);
    }
  }, 1000);
}

export default makeProject({
  scenes: [example],
});
