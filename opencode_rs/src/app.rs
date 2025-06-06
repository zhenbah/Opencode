// src/app.rs
use std::collections::HashMap;
use crate::llm::openai_client::{ToolCallRequestPart, FunctionCall, OpenAIClient}; // Added FunctionCall
use crate::session::{Session, Message, Author, ContentPart};
use crate::config::Config;
use crate::db;
use anyhow::Result;
use uuid::Uuid; // Added Uuid import

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum ToolPermissionScope {
    Once,
    Session,
}

#[derive(Debug, Clone)]
pub enum ToolPermissionState {
    Allowed,
    Denied,
}

#[derive(Debug, Clone)]
pub struct PendingToolCall {
    pub call_id: String,
    pub tool_name: String,
    pub arguments_json: String,
}

#[derive(Debug)]
pub struct App {
    pub sessions: HashMap<Uuid, Session>,
    pub active_session_id: Option<Uuid>,
    pub config: Config,
    pub db_pool: sqlx::SqlitePool,
    pub tool_session_permissions: HashMap<(String, Uuid), ToolPermissionState>, // Uuid for session_id
    pub pending_tool_call_request: Option<PendingToolCall>,
}

impl App {
    pub async fn new(config: Config) -> Result<Self> {
        let db_pool = db::init_db(&config).await.map_err(|e| anyhow::anyhow!("DB init failed: {}", e))?;
        let mut app = App {
            sessions: HashMap::new(),
            active_session_id: None,
            config,
            db_pool,
            tool_session_permissions: HashMap::new(),
            pending_tool_call_request: None,
        };
        app.load_sessions_from_db().await?; // This populates app.sessions
        if app.sessions.is_empty() {
            let _new_id = app.new_session(Some("Default Session".to_string())).await; // new_session makes it active
        } else {
            // Set active to most recent if sessions were loaded
            app.active_session_id = app.sessions.values().max_by_key(|s| s.last_activity_at).map(|s| s.id);
             if app.active_session_id.is_none() && !app.sessions.is_empty() { // Fallback if needed
                app.active_session_id = app.sessions.keys().next().cloned();
            }
        }
        Ok(app)
    }

    async fn load_sessions_from_db(&mut self) -> Result<()> {
        let loaded_sessions = db::load_sessions(&self.db_pool).await.map_err(|e| anyhow::anyhow!("Failed to load sessions: {}", e))?;
        for session in loaded_sessions {
            self.sessions.insert(session.id, session);
        }
        Ok(())
    }

    pub async fn new_session(&mut self, title: Option<String>) -> Uuid {
        let session = Session::new(title);
        let session_id = session.id;
        db::save_session(&self.db_pool, &session).await.unwrap_or_else(|e| log::error!("Failed to save session {}: {}", session_id, e));
        self.sessions.insert(session_id, session);
        self.active_session_id = Some(session_id);
        session_id
    }

    pub async fn add_message_to_active_session(&mut self, author: Author, parts: Vec<ContentPart>) {
        if let Some(session_id) = self.active_session_id {
            if let Some(session) = self.sessions.get_mut(&session_id) {
                let message = Message::new(author, parts);
                let msg_clone_for_db = message.clone(); // Clone before moving into session
                session.add_message(message); // This updates last_activity_at

                if let Err(e) = db::save_message(&self.db_pool, session_id, &msg_clone_for_db).await {
                     log::error!("DB save_message failed for session {}: {}",session_id, e);
                }
                if let Err(e) = db::save_session(&self.db_pool, session).await { // session is borrowed here
                     log::error!("DB save_session for last_activity failed for session {}: {}", session_id, e);
                }
            }
        }
    }

    pub async fn add_text_message_to_active_session(&mut self, author: Author, text: String) {
        self.add_message_to_active_session(author, vec![ContentPart::Text(text)]).await;
    }

    pub fn check_tool_session_permission(&self, tool_name: &str) -> Option<ToolPermissionState> {
        self.active_session_id.and_then(|sid| {
            self.tool_session_permissions.get(&(tool_name.to_string(), sid)).cloned()
        })
    }

    pub fn set_tool_session_permission(&mut self, tool_name: String, state: ToolPermissionState) {
        if let Some(sid) = self.active_session_id {
            log::info!("Setting permission for tool '{}' in session {} to {:?}", tool_name, sid, state);
            self.tool_session_permissions.insert((tool_name, sid), state);
        }
    }

    pub async fn send_current_session_to_llm(&mut self) {
        if self.pending_tool_call_request.is_some() {
            log::warn!("Attempted to send to LLM while a tool call is pending user permission.");
            self.add_text_message_to_active_session(Author::System, "[Info] Tool call pending user permission. Please respond to the dialog first.".to_string()).await;
            return;
        }

        let api_key_opt = self.config.providers.as_ref().and_then(|p|p.openai.as_ref()).and_then(|o|o.api_key.as_ref());
        if api_key_opt.is_none() { self.add_text_message_to_active_session(Author::System, "Error: OpenAI API key not configured.".to_string()).await; return; }
        let client = OpenAIClient::new(api_key_opt.unwrap().clone());

        let active_session_messages = if let Some(s) = self.get_active_session() {
            if s.messages.is_empty() { log::warn!("No messages in active session to send to LLM."); return; }
            s.messages.clone()
        } else { log::warn!("No active session to send to LLM."); return; };

        let model = self.config.agents.as_ref().and_then(|a|a.coder.as_ref()).and_then(|c|c.model.as_ref()).cloned().unwrap_or_else(||"gpt-3.5-turbo".to_string());

        log::info!("Sending {} messages to LLM (model: {})...", active_session_messages.len(), model);

        match client.chat_completion(&active_session_messages, model.clone()).await {
            Ok(response) => {
                if let Some(choice) = response.choices.into_iter().next() {
                    let assistant_response_message = choice.message.clone();
                    let assistant_response_content = assistant_response_message.content.clone();
                    let tool_calls_from_assistant = assistant_response_message.tool_calls.clone();

                    let mut assistant_message_parts: Vec<ContentPart> = Vec::new();
                    if let Some(text_content)=&assistant_response_content{ if !text_content.is_empty(){assistant_message_parts.push(ContentPart::Text(text_content.clone()));}}
                    if let Some(ref tc_reqs)=tool_calls_from_assistant{ for r in tc_reqs{assistant_message_parts.push(ContentPart::ToolRequest{id:r.id.clone(),name:r.function.name.clone(),input:r.function.arguments.clone()});}}

                    if !assistant_message_parts.is_empty() {
                         self.add_message_to_active_session(Author::Assistant, assistant_message_parts).await;
                    } else if tool_calls_from_assistant.is_none() {
                         log::info!("Assistant response was empty (no text, no tool calls).");
                    }

                    if let Some(actual_tool_calls) = tool_calls_from_assistant {
                        if !actual_tool_calls.is_empty() {
                            // For now, we'll handle the first tool call and queue the rest if permission is needed.
                            // A more sophisticated model might handle batches or parallel permissions.
                            if let Some(first_req) = actual_tool_calls.get(0) {
                                match self.check_tool_session_permission(&first_req.function.name) {
                                    Some(ToolPermissionState::Allowed) => {
                                        log::info!("Tool '{}' already permitted for this session. Executing batch of {} tools.", first_req.function.name, actual_tool_calls.len());
                                        self.execute_tool_calls_and_resend(actual_tool_calls).await;
                                    }
                                    Some(ToolPermissionState::Denied) => {
                                        log::info!("Tool '{}' previously denied for this session.", first_req.function.name);
                                        let tool_error_msg = ContentPart::ToolResult{id: first_req.id.clone(), name: first_req.function.name.clone(), output: "Tool execution denied by session policy.".to_string(), is_error: true};
                                        self.add_message_to_active_session(Author::Tool, vec![tool_error_msg]).await;
                                        self.send_current_session_to_llm().await; // Resend with denial info
                                    }
                                    None => { // Permission not yet set for this tool in this session
                                        log::info!("Tool '{}' requires user permission.", first_req.function.name);
                                        self.pending_tool_call_request = Some(PendingToolCall {
                                            call_id: first_req.id.clone(),
                                            tool_name: first_req.function.name.clone(),
                                            arguments_json: first_req.function.arguments.clone(),
                                        });
                                        // UI should now show permission dialog. No further LLM calls until resolved.
                                    }
                                }
                            }
                        } // else: no tool calls, LLM might have given final answer or just text.
                    } else { log::debug!("No tool calls from LLM this turn."); }
                } else { self.add_text_message_to_active_session(Author::System, "Error: No response choices from LLM.".to_string()).await; }
            }
            Err(e) => { self.add_text_message_to_active_session(Author::System, format!("Error: LLM request failed: {}", e)).await; }
        }
    }

    pub async fn execute_tool_calls_and_resend(&mut self, tool_calls: Vec<ToolCallRequestPart>) {
        // self.pending_tool_call_request = None; // Clear pending if we are executing a batch from allowed state.
                                               // If called from resolve_pending_tool_call, it's already cleared.

        for tool_call_request in tool_calls {
            let tool_name = tool_call_request.function.name.as_str();
            let tool_args_json = &tool_call_request.function.arguments;
            let tool_call_id = tool_call_request.id.clone();

            log::info!("Executing tool: {} (ID: {}) with args: {}", tool_name, tool_call_id, tool_args_json);
            let tool_run_result: Result<String, String> = match tool_name {
                "ls" => crate::tools::fs_tools::run_ls(tool_args_json),
                "view" => crate::tools::fs_tools::run_view(tool_args_json),
                "write" => crate::tools::fs_tools::run_write(tool_args_json),
                _ => {
                    log::warn!("Attempted to execute unknown tool: {}", tool_name);
                    Err(format!("Unknown tool: {}", tool_name))
                }
            };
            let (output_content, is_error) = match tool_run_result { Ok(s) => (s, false), Err(s) => (s, true), };
            self.add_message_to_active_session(Author::Tool, vec![
                ContentPart::ToolResult { id: tool_call_id, name: tool_name.to_string(), output: output_content, is_error, }
            ]).await;
        }

        if !tool_calls.is_empty() {
            log::info!("Resending session to LLM after tool execution cycle.");
            self.send_current_session_to_llm().await;
        }
    }

    pub async fn resolve_pending_tool_call(&mut self, allow: bool, scope_for_allow: Option<ToolPermissionScope>) {
        if let Some(pending_call) = self.pending_tool_call_request.take() { // .take() removes it
            if allow {
                log::info!("User allowed tool: {} (Scope: {:?})", pending_call.tool_name, scope_for_allow);
                if scope_for_allow == Some(ToolPermissionScope::Session) {
                     self.set_tool_session_permission(pending_call.tool_name.clone(), ToolPermissionState::Allowed);
                }
                // Construct a Vec with the single tool call to execute
                let single_tool_call_to_execute = vec![ToolCallRequestPart {
                    id: pending_call.call_id,
                    r#type: "function".to_string(), // Assuming "function" type, might need to be dynamic if other types are used
                    function: FunctionCall {
                        name: pending_call.tool_name,
                        arguments: pending_call.arguments_json,
                    },
                }];
                self.execute_tool_calls_and_resend(single_tool_call_to_execute).await;
            } else {
                log::info!("User denied tool: {}", pending_call.tool_name);
                // Optionally set session permission to Denied if that's desired behavior on explicit deny
                // self.set_tool_session_permission(pending_call.tool_name.clone(), ToolPermissionState::Denied);
                self.add_message_to_active_session(Author::Tool, vec![
                    ContentPart::ToolResult {
                        id: pending_call.call_id,
                        name: pending_call.tool_name,
                        output: "Tool execution denied by user.".to_string(),
                        is_error: true,
                    }
                ]).await;
                self.send_current_session_to_llm().await; // Resend context to LLM with denial
            }
        }
    }

    // Getter for active session, needed by TUI
    pub fn get_active_session(&self) -> Option<&Session> {
        self.active_session_id.and_then(|id| self.sessions.get(&id))
    }

    // Needed for TUI to list sessions, etc. (Not part of this subtask's core logic but good for completeness)
    pub fn list_sessions(&self) -> Vec<&Session> {
        self.sessions.values().collect()
    }

    pub fn switch_session(&mut self, session_id: Uuid) -> bool {
        if self.sessions.contains_key(&session_id) {
            self.active_session_id = Some(session_id);
            self.pending_tool_call_request = None; // Clear pending calls when switching session
            log::info!("Switched to session ID: {}", session_id);
            true
        } else {
            log::warn!("Attempted to switch to non-existent session ID: {}", session_id);
            false
        }
    }
}
