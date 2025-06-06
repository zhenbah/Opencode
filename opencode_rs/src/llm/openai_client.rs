use reqwest::Client;
use serde::{Deserialize, Serialize};
use crate::session::{Message as AppMessage, Author as AppAuthor, ContentPart as AppContentPart};

const OPENAI_API_URL: &str = "https://api.openai.com/v1/chat/completions";

// Structs for Tool Calling (OpenAI specific)
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ToolCallRequestPart {
    pub id: String,
    pub r#type: String, // "function"
    pub function: FunctionCall,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct FunctionCall {
    pub name: String,
    pub arguments: String, // JSON string of arguments
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ChatMessage {
    pub role: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_calls: Option<Vec<ToolCallRequestPart>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_call_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>, // For tool role, this is the function name
}

#[derive(Serialize, Debug)]
pub struct ToolDefinition {
    pub r#type: String, // "function"
    pub function: FunctionDefinition,
}

#[derive(Serialize, Debug)]
pub struct FunctionDefinition {
    pub name: String,
    pub description: String,
    pub parameters: FunctionParameters,
}

#[derive(Serialize, Debug)]
pub struct FunctionParameters {
    pub r#type: String, // "object"
    pub properties: std::collections::HashMap<String, FunctionParameterProperty>,
    pub required: Vec<String>,
}

#[derive(Serialize, Debug)]
pub struct FunctionParameterProperty {
    pub r#type: String, // "string", "integer", "boolean"
    pub description: String,
}

// Main Request and Response Structs
#[derive(Serialize, Debug)]
pub struct ChatCompletionRequest {
    pub model: String,
    pub messages: Vec<ChatMessage>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tools: Option<Vec<ToolDefinition>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_choice: Option<String>,
}

#[derive(Deserialize, Debug)]
pub struct ChatCompletionResponse {
    pub id: String,
    pub choices: Vec<Choice>,
    // Add usage, etc. later
}

#[derive(Deserialize, Debug)]
pub struct Choice {
    pub index: u32,
    pub message: ChatMessage, // This will be OpenAI's ChatMessage struct
    // pub finish_reason: String,
}

pub struct OpenAIClient {
    client: Client,
    api_key: String,
}

impl OpenAIClient {
    pub fn new(api_key: String) -> Self {
        Self {
            client: Client::new(),
            api_key,
        }
    }

    pub fn get_tool_definitions() -> Vec<ToolDefinition> {
        vec![
            ToolDefinition { // ls
                r#type: "function".to_string(),
                function: FunctionDefinition {
                    name: "ls".to_string(),
                    description: "List directory contents.".to_string(),
                    parameters: FunctionParameters {
                        r#type: "object".to_string(),
                        properties: [
                            ("path".to_string(), FunctionParameterProperty {
                                r#type: "string".to_string(),
                                description: "Optional path to list contents of. Defaults to current directory.".to_string(),
                            })
                        ].iter().cloned().collect(),
                        required: Vec::new(),
                    },
                },
            },
            ToolDefinition { // view
                r#type: "function".to_string(),
                function: FunctionDefinition {
                    name: "view".to_string(),
                    description: "View file contents.".to_string(),
                    parameters: FunctionParameters {
                        r#type: "object".to_string(),
                        properties: [
                            ("file_path".to_string(), FunctionParameterProperty {
                                r#type: "string".to_string(),
                                description: "Path to the file to view.".to_string(),
                            })
                        ].iter().cloned().collect(),
                        required: vec!["file_path".to_string()],
                    },
                },
            },
            ToolDefinition { // write
                r#type: "function".to_string(),
                function: FunctionDefinition {
                    name: "write".to_string(),
                    description: "Write content to a file. Overwrites if file exists.".to_string(),
                    parameters: FunctionParameters {
                        r#type: "object".to_string(),
                        properties: [
                            ("file_path".to_string(), FunctionParameterProperty {
                                r#type: "string".to_string(),
                                description: "Path to the file to write to.".to_string(),
                            }),
                            ("content".to_string(), FunctionParameterProperty {
                                r#type: "string".to_string(),
                                description: "Content to write to the file.".to_string(),
                            })
                        ].iter().cloned().collect(),
                        required: vec!["file_path".to_string(), "content".to_string()],
                    },
                },
            },
        ]
    }

    pub fn convert_messages(app_messages: &[AppMessage]) -> Vec<ChatMessage> {
        app_messages.iter().map(|app_msg| {
            let role = match app_msg.author {
                AppAuthor::User => "user".to_string(),
                AppAuthor::Assistant => "assistant".to_string(),
                AppAuthor::System => "system".to_string(),
                AppAuthor::Tool => "tool".to_string(),
            };

            let mut openapi_msg = ChatMessage {
                role: role.clone(),
                content: None,
                tool_calls: None,
                tool_call_id: None,
                name: None,
            };

            let mut text_parts_content = String::new();
            let mut current_tool_calls = Vec::new();

            match app_msg.author {
                AppAuthor::Tool => { // Tool role (response from a tool)
                    for part in &app_msg.parts {
                        if let AppContentPart::ToolResult { id, name, output, .. } = part {
                            openapi_msg.tool_call_id = Some(id.clone());
                            openapi_msg.name = Some(name.clone()); // name of the tool
                            text_parts_content.push_str(output);
                            break; // Assuming one ToolResult per Tool message
                        }
                    }
                }
                AppAuthor::Assistant => { // Assistant role (can make tool requests or send text)
                    for part in &app_msg.parts {
                        match part {
                            AppContentPart::Text(text) => text_parts_content.push_str(text),
                            AppContentPart::ToolRequest { id, name, input } => {
                                current_tool_calls.push(ToolCallRequestPart {
                                    id: id.clone(),
                                    r#type: "function".to_string(),
                                    function: FunctionCall { name: name.clone(), arguments: input.clone() },
                                });
                            }
                            _ => {} // Other parts like ToolResult not expected from Assistant
                        }
                    }
                    if !current_tool_calls.is_empty() {
                        openapi_msg.tool_calls = Some(current_tool_calls);
                    }
                }
                _ => { // User or System messages (primarily text)
                    for part in &app_msg.parts {
                        if let AppContentPart::Text(text) = part {
                            text_parts_content.push_str(text);
                        }
                        // User/System are not expected to make ToolRequests or send ToolResults
                    }
                }
            }

            if !text_parts_content.is_empty() {
                openapi_msg.content = Some(text_parts_content);
            }

            // OpenAI API requirements:
            // - "tool" role messages require content.
            // - "user" and "system" messages require content.
            // - "assistant" messages require content if tool_calls is not present.
            //   If tool_calls is present, content can be null/None.
            if (openapi_msg.role == "user" || openapi_msg.role == "system" || openapi_msg.role == "tool") && openapi_msg.content.is_none() {
                openapi_msg.content = Some("".to_string()); // Provide empty string if no text content but role requires it
            }

            // If assistant message is only tool_calls, content should be None.
            if openapi_msg.role == "assistant" && openapi_msg.tool_calls.is_some() && openapi_msg.content.as_deref() == Some("") {
                openapi_msg.content = None;
            }

            openapi_msg
        }).collect()
    }

    pub async fn chat_completion(
        &self,
        app_messages: &[AppMessage],
        model: String,
    ) -> Result<ChatCompletionResponse, reqwest::Error> {
        let messages = Self::convert_messages(app_messages);
        let request_payload = ChatCompletionRequest {
            model,
            messages,
            tools: Some(Self::get_tool_definitions()),
            tool_choice: Some("auto".to_string()), // Or "required" or specific tool
        };

        log::debug!("Sending OpenAI request with tools: {:?}", request_payload);

        let response = self.client
            .post(OPENAI_API_URL)
            .bearer_auth(&self.api_key)
            .json(&request_payload)
            .send()
            .await?;

        if response.status().is_success() {
            let chat_response = response.json::<ChatCompletionResponse>().await?;
            log::debug!("Received OpenAI response: {:?}", chat_response);
            Ok(chat_response)
        } else {
            let status = response.status();
            let error_text = response.text().await.unwrap_or_else(|_| "Unknown error".to_string());
            log::error!("OpenAI API Error: {} - {}", status, error_text);
            Err(reqwest::Error::from(std::io::Error::new(
                std::io::ErrorKind::Other,
                format!("OpenAI API Error: {} - {}", status, error_text),
            )))
        }
    }
}
