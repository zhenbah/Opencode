use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Debug, Clone)]
pub enum Author {
    User,
    Assistant,
    System,
    Tool, // For tool requests/results if we want to differentiate
}

#[derive(Debug, Clone)]
pub enum ContentPart {
    Text(String),
    ToolRequest { id: String, name: String, input: String },
    ToolResult { id: String, name: String, output: String, is_error: bool },
    // Potentially add Image, etc. later
}

#[derive(Debug, Clone)]
pub struct Message {
    pub id: Uuid,
    pub author: Author,
    pub parts: Vec<ContentPart>, // Changed from content to parts
    pub timestamp: DateTime<Utc>,
    // pub tool_calls: Option<Vec<ToolCall>>, // Consider if this is better than ContentPart::ToolRequest
    // pub tool_call_id: Option<String>, // For tool responses
}

impl Message {
    pub fn new(author: Author, parts: Vec<ContentPart>) -> Self {
        Message {
            id: Uuid::new_v4(),
            author,
            parts,
            timestamp: Utc::now(),
        }
    }

    // Helper to create a simple text message
    pub fn new_text(author: Author, text: String) -> Self {
        Message::new(author, vec![ContentPart::Text(text)])
    }
}

#[derive(Debug, Clone)]
pub struct Session {
    pub id: Uuid,
    pub title: String,
    pub messages: Vec<Message>,
    pub created_at: DateTime<Utc>,
    pub last_activity_at: DateTime<Utc>,
    // Potentially add model info, context window, etc.
}

impl Session {
    pub fn new(title: Option<String>) -> Self {
        let now = Utc::now();
        Session {
            id: Uuid::new_v4(),
            title: title.unwrap_or_else(|| format!("Session {}", now.format("%Y-%m-%d %H:%M:%S"))),
            messages: Vec::new(),
            created_at: now,
            last_activity_at: now,
        }
    }

    pub fn add_message(&mut self, message: Message) {
        self.messages.push(message);
        self.last_activity_at = Utc::now();
    }
}
