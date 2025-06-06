// src/tools/fs_tools.rs
use std::{fs, io, path::PathBuf}; // io might not be needed directly
use serde_json::Value;
use super::{get_string_arg, get_optional_string_arg};

pub fn run_ls(args_json: &str) -> Result<String, String> {
    log::debug!("Running ls tool with args: {}", args_json);
    let args: Value = serde_json::from_str(args_json).map_err(|e| format!("Invalid JSON arguments for ls: {}", e))?;
    let path_str = get_optional_string_arg(&args, "path").unwrap_or_else(|| ".".to_string());

    let path = PathBuf::from(&path_str); // Use &path_str
    if !path.exists() { return Err(format!("Path does not exist: {}", path.display())); }
    if !path.is_dir() { return Err(format!("Path is not a directory: {}", path.display())); }

    let mut entries_str = String::new();
    match fs::read_dir(path) {
        Ok(entries) => {
            for entry_result in entries {
                match entry_result {
                    Ok(entry) => {
                        let file_name = entry.file_name().to_string_lossy().to_string();
                        let entry_type = if entry.file_type().map_or(false, |ft| ft.is_dir()) { "[DIR]" } else { "[FILE]" };
                        entries_str.push_str(&format!("{} {}
", entry_type, file_name));
                    }
                    Err(e) => entries_str.push_str(&format!("[ERROR reading entry] {}\n", e)),
                }
            }
            Ok(entries_str.trim_end().to_string())
        }
        Err(e) => Err(format!("Failed to read directory: {}", e)),
    }
}

pub fn run_view(args_json: &str) -> Result<String, String> {
    log::debug!("Running view tool with args: {}", args_json);
    let args: Value = serde_json::from_str(args_json).map_err(|e| format!("Invalid JSON arguments for view: {}", e))?;
    let file_path_str = get_string_arg(&args, "file_path")?;

    let path = PathBuf::from(&file_path_str); // Use &file_path_str
    if !path.exists() { return Err(format!("File does not exist: {}", path.display())); }
    if !path.is_file() { return Err(format!("Path is not a file: {}", path.display())); }

    fs::read_to_string(path).map_err(|e| format!("Failed to read file: {}", e))
}

pub fn run_write(args_json: &str) -> Result<String, String> {
    log::debug!("Running write tool with args: {}", args_json);
    let args: Value = serde_json::from_str(args_json).map_err(|e| format!("Invalid JSON arguments for write: {}", e))?;
    let file_path_str = get_string_arg(&args, "file_path")?;
    let content = get_string_arg(&args, "content")?;

    let path = PathBuf::from(&file_path_str); // Use &file_path_str
    if let Some(parent_dir) = path.parent() {
        if !parent_dir.exists() {
            fs::create_dir_all(parent_dir).map_err(|e| format!("Failed to create parent directories for {}: {}", path.display(), e))?;
            log::info!("Created parent directory/ies for {}", path.display());
        }
    }

    fs::write(&path, content).map_err(|e| format!("Failed to write to file {}: {}", path.display(), e))?;
    Ok(format!("Successfully wrote to file {}", path.display()))
}
