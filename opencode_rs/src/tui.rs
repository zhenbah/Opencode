// src/tui.rs
use ratatui::{
    backend::Backend,
    layout::{Constraint, Direction, Layout, Rect},
    style::{Color, Style, Modifier},
    widgets::{Block, Borders, Clear, Paragraph, Wrap},
    Frame,
    Terminal,
};
use crossterm::event::{self, Event as CrosstermEvent, KeyCode, KeyModifiers};
use crate::app::{App, ToolPermissionScope, Author}; // Adjusted imports
use crate::session::ContentPart; // For TUI message display
use std::io;
use std::time::Duration;

pub struct Tui {
    terminal: Terminal<ratatui::backend::CrosstermBackend<io::Stdout>>,
}

impl Tui {
    pub fn new() -> io::Result<Self> {
        let stdout = io::stdout();
        let backend = ratatui::backend::CrosstermBackend::new(stdout);
        let terminal = Terminal::new(backend)?;
        Ok(Self { terminal })
    }

    fn format_json_for_display(json_str: &str) -> String {
        match serde_json::from_str::<serde_json::Value>(json_str) {
            Ok(val) => serde_json::to_string_pretty(&val).unwrap_or_else(|_| json_str.to_string()),
            Err(_) => json_str.to_string(),
        }
    }

    fn centered_rect(percent_x: u16, percent_y: u16, r: Rect) -> Rect {
        let popup_layout = Layout::default().direction(Direction::Vertical)
            .constraints([
                Constraint::Percentage((100 - percent_y) / 2),
                Constraint::Percentage(percent_y),
                Constraint::Percentage((100 - percent_y) / 2),
            ]).split(r);
        Layout::default().direction(Direction::Horizontal)
            .constraints([
                Constraint::Percentage((100 - percent_x) / 2),
                Constraint::Percentage(percent_x),
                Constraint::Percentage((100 - percent_x) / 2),
            ]).split(popup_layout[1])[1]
    }

    fn draw_permission_dialog(f: &mut Frame, app_state: &App) { // Changed to take &App
        if let Some(pending_call) = &app_state.pending_tool_call_request {
            let area = Self::centered_rect(70, 40, f.size()); // Increased y % for more space
            f.render_widget(Clear, area);

            let text_content = format!(
                "Allow tool execution?

Tool: {}
Arguments:
{}

                [A]llow Once | Allow for [S]ession | [D]eny | [Esc]ape (Deny)",
                pending_call.tool_name,
                Self::format_json_for_display(&pending_call.arguments_json)
            );

            let paragraph = Paragraph::new(text_content)
                .block(Block::default().borders(Borders::ALL).title("Permission Required").style(Style::default().fg(Color::Yellow)))
                .wrap(Wrap { trim: true });
            f.render_widget(paragraph, area);
        }
    }

    pub async fn run_loop(&mut self, app: &mut App) -> io::Result<()> {
        crossterm::terminal::enable_raw_mode()?;
        let mut stdout = io::stdout();
        crossterm::execute!(stdout, crossterm::terminal::EnterAlternateScreen, crossterm::event::EnableMouseCapture)?;
        log::info!("TUI run loop started.");

        loop {
            self.terminal.draw(|f| {
                let main_chunks = Layout::default().direction(Direction::Vertical).margin(1)
                    .constraints([Constraint::Percentage(80), Constraint::Percentage(20)].as_ref())
                    .split(f.size());

                let messages_text = app.get_active_session().map_or_else(
                    || "No active session. Press Ctrl+N for new session (not implemented).".to_string(),
                    |session| session.messages.iter().map(|msg| {
                        let author_str = match msg.author {
                            Author::User => "User", Author::Assistant => "Assistant", Author::System => "System", Author::Tool => "Tool",
                        };
                        let content_str = msg.parts.iter().filter_map(|part| {
                            match part {
                                ContentPart::Text(text) => Some(text.clone()), // Clone text
                                ContentPart::ToolRequest {name, input,..} => Some(format!("[Tool Call: {} with {}]", name, Self::format_json_for_display(input))),
                                ContentPart::ToolResult {name, output, is_error,..} => Some(format!("[Tool Result ({}): {} {}]", name, if *is_error {"ERROR:"} else {"OK:"}, output)),
                            }
                        }).collect::<Vec<String>>().join(" "); // Join Vec<String>
                        format!("{}: {}
", author_str, content_str)
                    }).collect::<String>()
                );
                let messages_paragraph = Paragraph::new(messages_text).block(Block::default().borders(Borders::ALL).title("Messages")).wrap(Wrap{trim:true});
                f.render_widget(messages_paragraph, main_chunks[0]);

                let input_text = if app.pending_tool_call_request.is_some() {
                    "Respond to permission dialog above ([A]llow Once, [S]ession Allow, [D]eny, [Esc]ape)..."
                } else {
                    "Input: [Ctrl+S to send example, q to quit]"
                };
                let input_paragraph = Paragraph::new(input_text)
                    .style(Style::default().fg(Color::White))
                    .block(Block::default().borders(Borders::ALL).title("Input"));
                f.render_widget(input_paragraph, main_chunks[1]);

                if app.pending_tool_call_request.is_some() {
                    Self::draw_permission_dialog(f, app); // Pass &App
                }
            })?;

            if crossterm::event::poll(Duration::from_millis(100))? {
                if let CrosstermEvent::Key(key) = event::read()? {
                    let mut key_handled_by_dialog = false;
                    if app.pending_tool_call_request.is_some() { // Check if dialog is active
                        match key.code {
                            KeyCode::Char('a') | KeyCode::Char('A') => {
                                app.resolve_pending_tool_call(true, Some(ToolPermissionScope::Once)).await;
                                key_handled_by_dialog = true;
                            }
                            KeyCode::Char('s') | KeyCode::Char('S') => {
                                // Check if it's 'S' for session permission, not Ctrl+S for send
                                if key.modifiers != KeyModifiers::CONTROL {
                                    app.resolve_pending_tool_call(true, Some(ToolPermissionScope::Session)).await;
                                    key_handled_by_dialog = true;
                                }
                            }
                            KeyCode::Char('d') | KeyCode::Char('D') | KeyCode::Esc => {
                                app.resolve_pending_tool_call(false, None).await;
                                key_handled_by_dialog = true;
                            }
                            _ => {}
                        }
                    }

                    if !key_handled_by_dialog { // If dialog didn't handle it, process global keys
                        match key.code {
                            KeyCode::Char('q') => {
                                if key.modifiers == KeyModifiers::NONE { // Ensure it's just 'q'
                                    log::info!("'q' pressed, exiting TUI loop.");
                                    break;
                                }
                            }
                            KeyCode::Char('s') if key.modifiers == KeyModifiers::CONTROL => {
                                if app.pending_tool_call_request.is_none() {
                                    log::info!("Ctrl+S: Sending example tool-using prompt.");
                                    let example_prompt = "List files in current directory, then write 'hello from permission test' to 'permission_test.txt'.";
                                    app.add_text_message_to_active_session(Author::User, example_prompt.to_string()).await;
                                    app.send_current_session_to_llm().await;
                                } else {
                                    log::warn!("Ctrl+S ignored: permission dialog active.");
                                }
                            }
                            _ => {}
                        }
                    }
                }
            }
        }
        crossterm::terminal::disable_raw_mode()?;
        crossterm::execute!(self.terminal.backend_mut(), crossterm::terminal::LeaveAlternateScreen, crossterm::event::DisableMouseCapture)?;
        self.terminal.show_cursor()?;
        Ok(())
    }
}
