// src/tools/mod.rs
pub mod fs_tools;

use serde_json::Value;

pub(super) fn get_string_arg(args: &Value, name: &str) -> Result<String, String> {
    args.get(name)
        .and_then(Value::as_str)
        .map(String::from)
        .ok_or_else(|| format!("Missing or invalid string argument: {}", name))
}

pub(super) fn get_optional_string_arg(args: &Value, name: &str) -> Option<String> {
    args.get(name).and_then(Value::as_str).map(String::from)
}
