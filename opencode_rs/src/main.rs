mod config;
mod session;
mod app;
mod tui; // Add this

use clap::Parser;
use config::Config;
use app::App;
use tui::Tui;
use anyhow::Result; // Use anyhow for main error type

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[arg(short, long, help = "Enable debug logging")]
    debug: bool,
    // other CLI args
}

#[tokio::main]
async fn main() -> Result<()> { // Return anyhow::Result
    let cli = Cli::parse();
    let config = Config::load(); // Load config first

    let debug_enabled = cli.debug || config.debug.unwrap_or(false);
    env_logger::Builder::from_env(
        env_logger::Env::default().default_filter_or(if debug_enabled { "debug" } else { "info" })
    ).init();

    log::info!("OpenCode Rust version starting...");
    log::debug!("CLI args: {:?}", cli);
    log::debug!("Loaded configuration: {:?}", config);

    let mut app = App::new(config).await?; // App::new is now async and returns Result

    // Example messages for testing TUI display - these should be await'ed
    // if you keep them. They will also be saved to DB.
    // Consider removing these from main or making them conditional.
    // if app.get_active_session().map_or(true, |s| s.messages.is_empty()) {
    //    app.add_text_message_to_active_session(session::Author::User, "What is Rust? (DB Test)".to_string()).await;
    // }

    // Conceptual testing for tool use:
    // Check if the current active session is empty. If so, add a complex prompt and kick off LLM.
    let is_active_session_empty = app.get_active_session().map_or(true, |s| s.messages.is_empty());
    if is_active_session_empty {
        log::info!("Active session is empty. Adding complex prompt for tool testing.");
        app.add_text_message_to_active_session(
            session::Author::User,
            "List files in the current directory, then write 'hello rust tool' to a file named 'rust_tool_test.txt', and finally show me the content of 'rust_tool_test.txt'."
        ).await;
        // Note: In a real scenario, the TUI would typically handle the first send via Ctrl+S.
        // For this conceptual test, we might auto-send or expect user to press Ctrl+S.
        // To ensure it runs for this test, we can call send_current_session_to_llm here.
        // However, this might run twice if user also presses Ctrl+S immediately.
        // For now, let's assume user will trigger via TUI or we just set up the prompt.
        // For a non-interactive test of this setup, uncommenting the next line would be needed:
        // app.send_current_session_to_llm().await;
    }


    let mut tui = Tui::new()?; // Tui::new is not async
    tui.run_loop(&mut app).await?;

    log::info!("Application finished.");
    Ok(())
}
