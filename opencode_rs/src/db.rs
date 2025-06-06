use sqlx::{migrate::MigrateDatabase, Sqlite, SqlitePool, Error as SqlxError};
use std::path::Path;
use crate::session::{Session, Message as AppMessage, Author as AppAuthor, ContentPart as AppContentPart};
use crate::config::Config;
use uuid::Uuid;
use chrono::{DateTime, Utc};
use serde_json; // For serializing ContentPart

// Define structs for database records that can be serialized/deserialized
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum DbAuthor { User, Assistant, System, Tool }

impl From<AppAuthor> for DbAuthor {
    fn from(author: AppAuthor) -> Self {
        match author {
            AppAuthor::User => DbAuthor::User,
            AppAuthor::Assistant => DbAuthor::Assistant,
            AppAuthor::System => DbAuthor::System,
            AppAuthor::Tool => DbAuthor::Tool,
        }
    }
}
impl From<DbAuthor> for AppAuthor {
    fn from(author: DbAuthor) -> Self {
        match author {
            DbAuthor::User => AppAuthor::User,
            DbAuthor::Assistant => AppAuthor::Assistant,
            DbAuthor::System => AppAuthor::System,
            DbAuthor::Tool => AppAuthor::Tool,
        }
    }
}


#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum DbContentPart {
    Text(String),
    ToolRequest { id: String, name: String, input: String },
    ToolResult { id: String, name: String, output: String, is_error: bool },
}

impl From<AppContentPart> for DbContentPart {
    fn from(part: AppContentPart) -> Self {
        match part {
            AppContentPart::Text(s) => DbContentPart::Text(s),
            AppContentPart::ToolRequest { id, name, input } => DbContentPart::ToolRequest { id, name, input },
            AppContentPart::ToolResult { id, name, output, is_error } => DbContentPart::ToolResult { id, name, output, is_error },
        }
    }
}
impl From<DbContentPart> for AppContentPart {
    fn from(part: DbContentPart) -> Self {
        match part {
            DbContentPart::Text(s) => AppContentPart::Text(s),
            DbContentPart::ToolRequest { id, name, input } => AppContentPart::ToolRequest { id, name, input },
            DbContentPart::ToolResult { id, name, output, is_error } => AppContentPart::ToolResult { id, name, output, is_error },
        }
    }
}


pub async fn init_db(config: &Config) -> Result<SqlitePool, SqlxError> {
    let db_url = &config.database_url;
    if !Sqlite::database_exists(db_url).await.unwrap_or(false) {
        log::info!("Creating database: {}", db_url);
        Sqlite::create_database(db_url).await?;
    } else {
        log::info!("Database already exists: {}", db_url);
    }

    let pool = SqlitePool::connect(db_url).await?;
    run_migrations(&pool).await?;
    Ok(pool)
}

async fn run_migrations(pool: &SqlitePool) -> Result<(), SqlxError> {
    // sqlx::migrate! macro points to a ./migrations folder by default
    // For embedded migrations:
    log::info!("Running database migrations...");
    sqlx::query(
        "CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY NOT NULL,
            title TEXT NOT NULL,
            created_at TEXT NOT NULL,
            last_activity_at TEXT NOT NULL
        );"
    ).execute(pool).await?;

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS messages (
            id TEXT PRIMARY KEY NOT NULL,
            session_id TEXT NOT NULL,
            author TEXT NOT NULL, -- Store DbAuthor as JSON string or simple string
            parts TEXT NOT NULL, -- Store Vec<DbContentPart> as JSON string
            timestamp TEXT NOT NULL,
            FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE
        );"
    ).execute(pool).await?;
    log::info!("Database migrations completed.");
    Ok(())
}

pub async fn save_session(pool: &SqlitePool, session: &Session) -> Result<(), SqlxError> {
    log::debug!("Saving session to DB: {}", session.id);
    sqlx::query(
        "INSERT OR REPLACE INTO sessions (id, title, created_at, last_activity_at) VALUES (?, ?, ?, ?)"
    )
    .bind(session.id.to_string())
    .bind(&session.title)
    .bind(session.created_at.to_rfc3339())
    .bind(session.last_activity_at.to_rfc3339())
    .execute(pool)
    .await?;
    Ok(())
}

pub async fn save_message(pool: &SqlitePool, session_id: Uuid, message: &AppMessage) -> Result<(), SqlxError> {
    log::debug!("Saving message to DB for session {}: {}", session_id, message.id);
    let author_db: DbAuthor = message.author.clone().into();
    let author_str = serde_json::to_string(&author_db).map_err(|e| SqlxError::Decode(Box::new(e)))?;

    let parts_db: Vec<DbContentPart> = message.parts.iter().cloned().map(Into::into).collect();
    let parts_json = serde_json::to_string(&parts_db).map_err(|e| SqlxError::Decode(Box::new(e)))?;

    sqlx::query(
        "INSERT INTO messages (id, session_id, author, parts, timestamp) VALUES (?, ?, ?, ?, ?)"
    )
    .bind(message.id.to_string())
    .bind(session_id.to_string())
    .bind(author_str)
    .bind(parts_json)
    .bind(message.timestamp.to_rfc3339())
    .execute(pool)
    .await?;
    Ok(())
}

pub async fn load_sessions(pool: &SqlitePool) -> Result<Vec<Session>, SqlxError> {
    log::debug!("Loading all sessions from DB");
    struct SessionRow { id: String, title: String, created_at: String, last_activity_at: String }

    let rows = sqlx::query_as!(SessionRow, "SELECT id, title, created_at, last_activity_at FROM sessions ORDER BY last_activity_at DESC")
        .fetch_all(pool)
        .await?;

    let mut sessions = Vec::new();
    for row in rows {
        let session_id = Uuid::parse_str(&row.id).map_err(|e| SqlxError::Decode(Box::new(e)))?;
        let mut session = Session {
            id: session_id,
            title: row.title,
            created_at: DateTime::parse_from_rfc3339(&row.created_at).unwrap().with_timezone(&Utc),
            last_activity_at: DateTime::parse_from_rfc3339(&row.last_activity_at).unwrap().with_timezone(&Utc),
            messages: Vec::new(), // Messages will be loaded separately or on demand
        };
        session.messages = load_messages_for_session(pool, session_id).await?;
        sessions.push(session);
    }
    log::info!("Loaded {} sessions from DB", sessions.len());
    Ok(sessions)
}

pub async fn load_messages_for_session(pool: &SqlitePool, session_id: Uuid) -> Result<Vec<AppMessage>, SqlxError> {
    log::debug!("Loading messages for session ID: {}", session_id);
    struct MessageRow { id: String, author: String, parts: String, timestamp: String }

    let rows = sqlx::query_as!(MessageRow,
        "SELECT id, author, parts, timestamp FROM messages WHERE session_id = ? ORDER BY timestamp ASC",
        session_id.to_string()
    )
    .fetch_all(pool)
    .await?;

    let mut messages = Vec::new();
    for row in rows {
        let author_db: DbAuthor = serde_json::from_str(&row.author).map_err(|e| SqlxError::Decode(Box::new(e)))?;
        let parts_db: Vec<DbContentPart> = serde_json::from_str(&row.parts).map_err(|e| SqlxError::Decode(Box::new(e)))?;

        messages.push(AppMessage {
            id: Uuid::parse_str(&row.id).map_err(|e| SqlxError::Decode(Box::new(e)))?,
            author: author_db.into(),
            parts: parts_db.into_iter().map(Into::into).collect(),
            timestamp: DateTime::parse_from_rfc3339(&row.timestamp).unwrap().with_timezone(&Utc),
        });
    }
    Ok(messages)
}
