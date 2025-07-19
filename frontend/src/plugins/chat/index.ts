import type { Plugin } from '@motion-canvas/core';
import { ChatTabConfig } from './ChatTabConfig';

export default function(): Plugin {
  return {
    name: 'chat-plugin',
  };
}

// Export the tab configuration separately
export const chatTab = {
  name: 'chat',
  title: 'Chat',
  icon: 'forum',
  tabComponent: ChatTabConfig,
};
