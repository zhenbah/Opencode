# Motion Canvas Chat Overlay

This project includes a floating chat overlay for collaborative communication within the Motion Canvas editor.

## Features

- **Floating chat overlay** that appears over the Motion Canvas editor
- **Toggle button** for easy show/hide functionality
- **Username customization** for each user
- **Message timestamps** for better organization
- **Auto-scrolling** to the latest messages
- **Keyboard shortcuts** (Enter to send, Shift+Enter for new lines)
- **Modern dark theme** that complements Motion Canvas
- **System messages** for notifications
- **Responsive design** that works on different screen sizes

## Usage

1. **Start the development server:**
   ```bash
   npm run serve
   ```

2. **Access the chat:**
   - Open Motion Canvas editor in your browser
   - Look for the chat button (💬) in the bottom-right corner
   - Click to open/close the chat overlay
   - Set your username and start messaging!

## Files Structure

```
src/
├── project.ts                          # Main project file with chat integration
├── scenes/
│   └── example.tsx                     # Example scene
└── plugins/
    └── chat/
        ├── ChatOverlay.tsx             # Chat overlay component
        ├── ChatIntegration.tsx         # Integration script
        └── index.ts                    # Original plugin (unused)
```

## Implementation Details

The chat system is implemented as a floating overlay rather than a sidebar plugin because Motion Canvas doesn't support custom sidebar plugins in its current architecture. The overlay:

- **Positions absolutely** over the Motion Canvas editor
- **Uses a floating action button** for access
- **Provides full chat functionality** without interfering with the editor
- **Auto-initializes** when the project loads

## Customization

You can customize the chat overlay by modifying `ChatOverlay.tsx`:

- **Styling**: Update the inline styles for colors, positioning, and sizing
- **Features**: Add features like message reactions, file sharing, or user lists
- **Backend Integration**: Connect to a real chat service or WebSocket server
- **Persistence**: Add local storage or database integration for message history

## Global Access

The chat system is available globally via:

```javascript
// Get the chat instance
const chat = window.MotionCanvasChat.getInstance();

// Control chat programmatically
chat.show();    // Show chat
chat.hide();    // Hide chat
chat.toggle();  // Toggle chat visibility
```

## Notes

- The chat currently stores messages in memory only
- Messages will reset when you refresh the page
- The overlay uses Preact for the interface
- All styling uses modern CSS with dark theme colors
- The chat is positioned as a fixed overlay and won't interfere with Motion Canvas functionality

## Troubleshooting

If you encounter issues:

1. **Chat not appearing**: Check browser console for errors and ensure Preact is installed
2. **Build errors**: Ensure all dependencies are installed (`npm install`)
3. **Styling issues**: Check that the chat container is properly positioned
4. **TypeScript errors**: Ensure `tsconfig.json` includes Preact configuration

The chat overlay provides a seamless collaborative experience for team communication during Motion Canvas animation development!
