#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::{Deserialize, Serialize};
use tauri::Manager;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EditorCommand {
    pub action: String,
    pub payload: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EditorResponse {
    pub success: bool,
    pub data: Option<serde_json::Value>,
    pub error: Option<String>,
}

#[tauri::command]
fn open_file(path: String) -> Result<EditorResponse, String> {
    std::fs::read_to_string(&path)
        .map(|content| {
            EditorResponse {
                success: true,
                data: Some(serde_json::json!({ "path": path, "content": content })),
                error: None,
            }
        })
        .map_err(|e| e.to_string())
}

#[tauri::command]
fn save_file(path: String, content: String) -> Result<EditorResponse, String> {
    std::fs::write(&path, &content)
        .map(|_| {
            EditorResponse {
                success: true,
                data: Some(serde_json::json!({ "path": path })),
                error: None,
            }
        })
        .map_err(|e| e.to_string())
}

#[tauri::command]
fn show_plugins_window() -> Result<(), String> {
    // This will be implemented to show the plugins marketplace window
    Ok(())
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .invoke_handler(tauri::generate_handler![
            open_file,
            save_file,
            show_plugins_window
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
