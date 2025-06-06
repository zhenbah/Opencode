use serde::Deserialize;
use std::path::PathBuf;
use std::env;
use std::fs;

#[derive(Deserialize, Debug, Default)]
#[serde(rename_all = "camelCase")]
pub struct Config {
    pub debug: Option<bool>,
    pub providers: Option<Providers>,
    #[serde(default)]
    pub shell: ShellConfig,
    pub agents: Option<Agents>,
    #[serde(default = "default_database_url")]
    pub database_url: String,
    pub data: Option<DataDirConfig>,
    // Add other top-level fields as needed, e.g., auto_compact
}

fn default_database_url() -> String {
    "sqlite:opencode.db".to_string()
}

#[derive(Deserialize, Debug, Default)]
#[serde(rename_all = "camelCase")]
pub struct DataDirConfig {
    #[serde(default = "default_data_directory")]
    pub directory: String,
}

fn default_data_directory() -> String {
    ".opencode".to_string()
}

#[derive(Deserialize, Debug, Default)]
#[serde(rename_all = "camelCase")]
pub struct Agents {
    pub coder: Option<AgentConfig>,
    // task, title agents later
}

#[derive(Deserialize, Debug, Default)]
#[serde(rename_all = "camelCase")]
pub struct AgentConfig {
    pub model: Option<String>,
    // maxTokens later
}

#[derive(Deserialize, Debug, Default)]
#[serde(rename_all = "camelCase")]
pub struct Providers {
    pub openai: Option<OpenAIProviderConfig>,
    pub anthropic: Option<ProviderConfig>, // Assuming similar structure for now
    pub groq: Option<ProviderConfig>,
    // Add other providers as needed
}

#[derive(Deserialize, Debug, Default)]
#[serde(rename_all = "camelCase")]
pub struct ProviderConfig {
    pub api_key: Option<String>,
    pub disabled: Option<bool>,
}

#[derive(Deserialize, Debug, Default)]
#[serde(rename_all = "camelCase")]
pub struct OpenAIProviderConfig {
    pub api_key: Option<String>,
    pub disabled: Option<bool>,
    // Potentially other OpenAI specific fields later
}

#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct ShellConfig {
    pub path: Option<String>,
    pub args: Option<Vec<String>>,
}

impl Default for ShellConfig {
    fn default() -> Self {
        ShellConfig {
            path: Some(env::var("SHELL").unwrap_or_else(|_| "/bin/bash".to_string())),
            args: Some(Vec::new()),
        }
    }
}

impl Config {
    pub fn load() -> Self {
        let mut cfg = Self::load_from_files();

        // Override with environment variables
        if let Ok(api_key) = env::var("OPENAI_API_KEY") {
            // Ensure providers and openai config exist
            let providers = cfg.providers.get_or_insert_with(Default::default);
            let openai_cfg = providers.openai.get_or_insert_with(Default::default);
            openai_cfg.api_key = Some(api_key);
        }
        // Add more environment variable overrides here (e.g., ANTHROPIC_API_KEY, GEMINI_API_KEY)

        if let Some(debug_env) = env::var("OPENCODE_DEBUG").ok().and_then(|s| s.parse::<bool>().ok()) {
            cfg.debug = Some(debug_env);
        }

        log::debug!("Loaded config: {:?}", cfg);
        cfg
    }

    fn load_from_files() -> Self {
        let mut config = Config::default();

        let config_paths = [
            dirs::home_dir().map(|p| p.join(".opencode.json")),
            dirs::config_dir().map(|p| p.join("opencode/.opencode.json")),
            Some(PathBuf::from("./.opencode.json")),
        ];

        for path_opt in config_paths.iter().rev() { // Load in reverse order of precedence, so local overrides global
            if let Some(path) = path_opt {
                if path.exists() {
                    log::debug!("Attempting to load config from: {:?}", path);
                    if let Ok(content) = fs::read_to_string(path) {
                        match serde_json::from_str::<Config>(&content) {
                            Ok(loaded_config) => {
                                // Merge loaded_config into config
                                // This is a simple merge, more sophisticated merging might be needed for nested Options
                                if loaded_config.debug.is_some() { config.debug = loaded_config.debug; }

                                if let Some(loaded_providers) = loaded_config.providers {
                                    let mut current_providers = config.providers.take().unwrap_or_default();
                                    if let Some(loaded_openai) = loaded_providers.openai {
                                        let mut current_openai = current_providers.openai.take().unwrap_or_default();
                                        if loaded_openai.api_key.is_some() { current_openai.api_key = loaded_openai.api_key; }
                                        if loaded_openai.disabled.is_some() { current_openai.disabled = loaded_openai.disabled; }
                                        current_providers.openai = Some(current_openai);
                                    }
                                    // Add merging for other providers
                                    config.providers = Some(current_providers);
                                }
                                // Basic merge for agents
                                if let Some(loaded_agents) = loaded_config.agents {
                                    let mut current_agents = config.agents.take().unwrap_or_default();
                                    if let Some(loaded_coder) = loaded_agents.coder {
                                        let mut current_coder = current_agents.coder.take().unwrap_or_default();
                                        if loaded_coder.model.is_some() { current_coder.model = loaded_coder.model; }
                                        current_agents.coder = Some(current_coder);
                                    }
                                    config.agents = Some(current_agents);
                                }
                                if loaded_config.database_url != default_database_url() && !loaded_config.database_url.is_empty() { // Check if it's not default or empty
                                    config.database_url = loaded_config.database_url;
                                }
                                if let Some(loaded_data_dir) = loaded_config.data {
                                    if loaded_data_dir.directory != default_data_directory() && !loaded_data_dir.directory.is_empty() {
                                        config.data.get_or_insert_with(Default::default).directory = loaded_data_dir.directory;
                                    }
                                }

                                 if loaded_config.shell.path.is_some() { config.shell.path = loaded_config.shell.path;}
                                 if loaded_config.shell.args.is_some() { config.shell.args = loaded_config.shell.args;}

                                log::info!("Successfully loaded and merged config from: {:?}", path);
                            }
                            Err(e) => log::warn!("Failed to parse config file at {:?}: {}", path, e),
                        }
                    } else {
                        log::warn!("Failed to read config file at {:?}", path);
                    }
                }
            }
        }
        config
    }
}
