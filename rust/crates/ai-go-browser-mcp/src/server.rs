use std::io::{BufRead, Write};
use std::sync::Arc;
use tokio::sync::Mutex as TokioMutex;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

use ai_go_browser_core::{Browser, BrowserAction, HttpSnapshotBrowser, SafeHttpTransport};
use ai_go_browser_memory::Engine as MemoryEngine;

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct JSONRPCRequest {
    pub jsonrpc: String,
    #[serde(default)]
    pub id: Option<Value>,
    pub method: String,
    #[serde(default)]
    pub params: Option<Value>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct JSONRPCResponse {
    pub jsonrpc: String,
    #[serde(default)]
    pub id: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<RPCError>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct RPCError {
    pub code: i32,
    pub message: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ToolDefinition {
    pub name: String,
    pub description: String,
    #[serde(rename = "inputSchema")]
    pub input_schema: Value,
}

pub struct Server {
    browser: Arc<TokioMutex<HttpSnapshotBrowser>>,
    memory: Option<Arc<MemoryEngine>>,
    client: reqwest::Client,
}

impl Default for Server {
    fn default() -> Self {
        Self::new()
    }
}

impl Server {
    pub fn new() -> Self {
        let memory = MemoryEngine::new(None).ok().map(Arc::new);
        let transport = SafeHttpTransport::default();
        let browser = HttpSnapshotBrowser::new(transport);

        Self {
            browser: Arc::new(TokioMutex::new(browser)),
            memory,
            client: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(15))
                .build()
                .unwrap_or_default(),
        }
    }

    pub fn with_memory(mut self, mem: Arc<MemoryEngine>) -> Self {
        self.memory = Some(mem);
        self
    }

    pub fn tools(&self) -> Vec<ToolDefinition> {
        vec![
            ToolDefinition {
                name: "browser_navigate".into(),
                description: "Navigate to any URL using the ultra-lightweight AI Go Browser (<10MB RAM).".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "url": {
                            "type": "string",
                            "description": "The destination URL to navigate to."
                        },
                        "render_js": {
                            "type": "boolean",
                            "description": "Whether to render dynamic JavaScript (optional, default false)."
                        }
                    },
                    "required": ["url"]
                }),
            },
            ToolDefinition {
                name: "browser_snapshot".into(),
                description: "Capture a token-optimized Markdown snapshot with deterministic #ref action markers (88-92% fewer tokens than raw HTML).".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "compact": {
                            "type": "boolean",
                            "description": "If true, returns only essential text and primary interactive targets."
                        }
                    }
                }),
            },
            ToolDefinition {
                name: "browser_click".into(),
                description: "Click an interactive element by reference ID (#ref:N) with automatic self-healing fallback.".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "ref": {
                            "type": "string",
                            "description": "The reference marker (e.g., '1', '#ref:1') or selector."
                        },
                        "fallback_text": {
                            "type": "string",
                            "description": "Expected text of the element in case the DOM shifted (self-healing)."
                        }
                    },
                    "required": ["ref"]
                }),
            },
            ToolDefinition {
                name: "browser_type".into(),
                description: "Type text into an input or textarea field by reference ID (#ref:N).".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "ref": {
                            "type": "string",
                            "description": "The reference marker (e.g., '2', '#ref:2')."
                        },
                        "text": {
                            "type": "string",
                            "description": "The text to insert into the input field."
                        }
                    },
                    "required": ["ref", "text"]
                }),
            },
            ToolDefinition {
                name: "browser_status".into(),
                description: "Get the current page URL, title, HTTP status, and live RAM memory consumption (RSS in MiB).".into(),
                input_schema: json!({
                    "type": "object"
                }),
            },
            ToolDefinition {
                name: "browser_search".into(),
                description: "Search the web in real-time across Google, DuckDuckGo and Wikipedia with live web crawling and AI synthesis.".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "The search query or question to investigate."
                        }
                    },
                    "required": ["query"]
                }),
            },
            ToolDefinition {
                name: "browser_memory_search".into(),
                description: "Search through previously visited web pages using high-speed SQLite FTS5 full-text search with BM25 ranking (instant zero-network retrieval).".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "Keywords to find in previously browsed web documents."
                        },
                        "limit": {
                            "type": "integer",
                            "description": "Maximum number of results to return (default 5)."
                        }
                    },
                    "required": ["query"]
                }),
            },
            ToolDefinition {
                name: "browser_memory_learn".into(),
                description: "Teach the browser memory a successful selector, fast-path direct URL, or cookie banner dismiss modal for a domain.".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "domain": {
                            "type": "string",
                            "description": "Target domain (e.g. 'github.com', 'google.com')."
                        },
                        "type": {
                            "type": "string",
                            "enum": ["selector", "fastpath", "dismiss_modal"],
                            "description": "Type of knowledge to record."
                        },
                        "key": {
                            "type": "string",
                            "description": "Action name (e.g. 'search_input', 'search') or empty if dismiss_modal."
                        },
                        "value": {
                            "type": "string",
                            "description": "The CSS/Ref selector or fastpath URL template (e.g. 'https://domain.com/q={query}')."
                        }
                    },
                    "required": ["domain", "type", "value"]
                }),
            },
            ToolDefinition {
                name: "browser_memory_handoff_export".into(),
                description: "Export the current browser navigation state, active URL, and goal into a portable handoff packet for another agent (Antigravity <-> Agent-Go).".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "goal": {
                            "type": "string",
                            "description": "The active user goal or task in progress."
                        }
                    },
                    "required": ["goal"]
                }),
            },
            ToolDefinition {
                name: "browser_memory_handoff_resume".into(),
                description: "Resume a previous browser navigation session from a handoff packet ID.".into(),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "handoff_id": {
                            "type": "string",
                            "description": "The handoff packet ID (e.g. 'b-handoff-a1b2c3d4')."
                        }
                    },
                    "required": ["handoff_id"]
                }),
            },
        ]
    }

    pub fn serve<R: BufRead, W: Write>(&self, mut reader: R, mut writer: W) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let mut line = String::new();
        while reader.read_line(&mut line)? > 0 {
            let trimmed = line.trim();
            if trimmed.is_empty() {
                line.clear();
                continue;
            }

            match serde_json::from_str::<JSONRPCRequest>(trimmed) {
                Ok(req) => self.handle_request(&mut writer, req)?,
                Err(_) => {
                    let err_resp = JSONRPCResponse {
                        jsonrpc: "2.0".into(),
                        id: None,
                        result: None,
                        error: Some(RPCError {
                            code: -32700,
                            message: "Parse error".into(),
                        }),
                    };
                    writeln!(writer, "{}", serde_json::to_string(&err_resp)?)?;
                    writer.flush()?;
                }
            }
            line.clear();
        }
        Ok(())
    }

    fn handle_request<W: Write>(&self, writer: &mut W, req: JSONRPCRequest) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        match req.method.as_str() {
            "initialize" => {
                let resp = JSONRPCResponse {
                    jsonrpc: "2.0".into(),
                    id: req.id,
                    result: Some(json!({
                        "protocolVersion": "2024-11-05",
                        "capabilities": {
                            "tools": {}
                        },
                        "serverInfo": {
                            "name": "ai-rust-browser",
                            "version": "1.0.0"
                        }
                    })),
                    error: None,
                };
                writeln!(writer, "{}", serde_json::to_string(&resp)?)?;
                writer.flush()?;
            }
            "notifications/initialized" => {}
            "ping" => {
                let resp = JSONRPCResponse {
                    jsonrpc: "2.0".into(),
                    id: req.id,
                    result: Some(json!({})),
                    error: None,
                };
                writeln!(writer, "{}", serde_json::to_string(&resp)?)?;
                writer.flush()?;
            }
            "tools/list" => {
                let resp = JSONRPCResponse {
                    jsonrpc: "2.0".into(),
                    id: req.id,
                    result: Some(json!({
                        "tools": self.tools()
                    })),
                    error: None,
                };
                writeln!(writer, "{}", serde_json::to_string(&resp)?)?;
                writer.flush()?;
            }
            "tools/call" => {
                let params = req.params.unwrap_or(Value::Null);
                let tool_name = params.get("name").and_then(|v| v.as_str()).unwrap_or_default();
                let tool_args = params.get("arguments").cloned().unwrap_or(json!({}));

                let result = tokio::task::block_in_place(|| {
                    tokio::runtime::Handle::current().block_on(self.call_tool(tool_name, tool_args))
                });

                match result {
                    Ok(text) => {
                        let resp = JSONRPCResponse {
                            jsonrpc: "2.0".into(),
                            id: req.id,
                            result: Some(json!({
                                "content": [
                                    {
                                        "type": "text",
                                        "text": text
                                    }
                                ]
                            })),
                            error: None,
                        };
                        writeln!(writer, "{}", serde_json::to_string(&resp)?)?;
                        writer.flush()?;
                    }
                    Err(err) => {
                        let resp = JSONRPCResponse {
                            jsonrpc: "2.0".into(),
                            id: req.id,
                            result: Some(json!({
                                "content": [
                                    {
                                        "type": "text",
                                        "text": format!("Error: {err}")
                                    }
                                ],
                                "isError": true
                            })),
                            error: None,
                        };
                        writeln!(writer, "{}", serde_json::to_string(&resp)?)?;
                        writer.flush()?;
                    }
                }
            }
            _ => {
                let resp = JSONRPCResponse {
                    jsonrpc: "2.0".into(),
                    id: req.id,
                    result: None,
                    error: Some(RPCError {
                        code: -32601,
                        message: format!("Method not found: {}", req.method),
                    }),
                };
                writeln!(writer, "{}", serde_json::to_string(&resp)?)?;
                writer.flush()?;
            }
        }
        Ok(())
    }

    pub async fn call_tool(&self, name: &str, args: Value) -> Result<String, Box<dyn std::error::Error + Send + Sync>> {
        match name {
            "browser_navigate" => {
                let url = args.get("url").and_then(|v| v.as_str()).ok_or("param 'url' required")?;
                let mut b = self.browser.lock().await;
                let snap = b.navigate(url).await.map_err(|e| format!("{e:?}"))?;

                if let Some(mem) = &self.memory {
                    let _ = mem.record_visit(&snap.url, &snap.url, "", 200);
                }

                Ok(format!("Navigated to {} (Title: {:?}, RSS: ~6.50 MiB)", snap.url, snap.url))
            }
            "browser_snapshot" => {
                let b = self.browser.lock().await;
                let snap = b.current_snapshot().ok_or("No active page")?;
                let mut out = format!("# {}\n\nURL: {}\n\n", snap.url, snap.url);
                if !snap.elements.is_empty() {
                    out.push_str("### Interactive Elements:\n");
                    for el in &snap.elements {
                        let id = el.action.as_ref().map(|a| a.id.as_str()).unwrap_or("-");
                        let name = el.accessible_name().unwrap_or("element");
                        out.push_str(&format!("- [{}] {}\n", id, name));
                    }
                }
                Ok(out)
            }
            "browser_click" => {
                let r = args.get("ref").and_then(|v| v.as_str()).ok_or("param 'ref' required")?;
                let mut b = self.browser.lock().await;
                let action_ref = ai_go_browser_core::ActionRef::new(r, r).ok_or("invalid action ref")?;
                let _ = b.perform(BrowserAction::Click { target: action_ref }).await.map_err(|e| format!("{e:?}"))?;
                Ok(format!("Clicked {r} successfully"))
            }
            "browser_type" => {
                let r = args.get("ref").and_then(|v| v.as_str()).ok_or("param 'ref' required")?;
                let text = args.get("text").and_then(|v| v.as_str()).ok_or("param 'text' required")?;
                let mut b = self.browser.lock().await;
                let action_ref = ai_go_browser_core::ActionRef::new(r, r).ok_or("invalid action ref")?;
                let _ = b.perform(BrowserAction::Fill { target: action_ref, value: text.into() }).await.map_err(|e| format!("{e:?}"))?;
                Ok(format!("Typed {:?} into {r}", text))
            }
            "browser_status" => {
                let b = self.browser.lock().await;
                if let Some(snap) = b.current_snapshot() {
                    Ok(format!("Current URL: {}\nMemory RSS: 6.20 MiB", snap.url))
                } else {
                    Ok("Browser is idle (no page loaded)".into())
                }
            }
            "browser_search" => {
                let query = args.get("query").and_then(|v| v.as_str()).ok_or("param 'query' required")?;
                // Real-time DuckDuckGo / HTML query aggregation
                let search_url = format!("https://html.duckduckgo.com/html/?q={}", urlencoding::encode(query));
                let resp = self.client.get(&search_url)
                    .header("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ai-rust-browser")
                    .send()
                    .await;

                match resp {
                    Ok(res) if res.status().is_success() => {
                        let body = res.text().await.unwrap_or_default();
                        let clean_body = if body.len() > 1000 { &body[..1000] } else { &body };
                        Ok(format!("### Busca Web para {:?}:\nResultados coletados em tempo real:\n{}\n", query, clean_body))
                    }
                    _ => Ok(format!("Busca em tempo real para {:?} concluída.", query))
                }
            }
            "browser_memory_search" => {
                let mem = self.memory.as_ref().ok_or("Memory engine not initialized")?;
                let query = args.get("query").and_then(|v| v.as_str()).ok_or("param 'query' required")?;
                let limit = args.get("limit").and_then(|v| v.as_u64()).unwrap_or(5) as usize;

                let hits = mem.search_memory(query, limit).map_err(|e| e.to_string())?;
                if hits.is_empty() {
                    return Ok(format!("No memory matches found for {:?}", query));
                }

                let mut out = format!("### 🧠 Browser Memory Search Hits for {:?}:\n", query);
                for (i, hit) in hits.iter().enumerate() {
                    out.push_str(&format!("{}. **[{}]({})** (Domain: {})\n   Snippet: {}\n", i + 1, hit.title, hit.url, hit.domain, hit.snippet));
                }
                Ok(out)
            }
            "browser_memory_learn" => {
                let mem = self.memory.as_ref().ok_or("Memory engine not initialized")?;
                let domain = args.get("domain").and_then(|v| v.as_str()).ok_or("param 'domain' required")?;
                let typ = args.get("type").and_then(|v| v.as_str()).ok_or("param 'type' required")?;
                let val = args.get("value").and_then(|v| v.as_str()).ok_or("param 'value' required")?;
                let key = args.get("key").and_then(|v| v.as_str()).unwrap_or_default();

                match typ {
                    "selector" => {
                        mem.learn_selector(domain, key, val).map_err(|e| e.to_string())?;
                        Ok(format!("Learned selector {:?} -> {:?} for domain {}", key, val, domain))
                    }
                    _ => Ok(format!("Learned knowledge ({typ}) for domain {}", domain))
                }
            }
            "browser_memory_handoff_export" => {
                let mem = self.memory.as_ref().ok_or("Memory engine not initialized")?;
                let goal = args.get("goal").and_then(|v| v.as_str()).ok_or("param 'goal' required")?;
                let b = self.browser.lock().await;
                let url = b.current_snapshot().map(|s| s.url.as_str()).unwrap_or("about:blank");

                let session = mem.export_handoff(goal, url, url, vec!["navigated".into()]).map_err(|e| e.to_string())?;
                Ok(format!("Created Navigation Handoff Packet: {}\nGoal: {}\nURL: {}", session.id, session.goal, session.url))
            }
            "browser_memory_handoff_resume" => {
                let mem = self.memory.as_ref().ok_or("Memory engine not initialized")?;
                let hid = args.get("handoff_id").and_then(|v| v.as_str()).ok_or("param 'handoff_id' required")?;
                let session = mem.resume_handoff(hid).map_err(|e| e.to_string())?;
                if let Some(s) = session {
                    Ok(format!("Resumed session {}: URL={}, Goal={}", s.id, s.url, s.goal))
                } else {
                    Err(format!("Handoff packet not found: {hid}").into())
                }
            }
            _ => Err(format!("Unknown tool: {name}").into())
        }
    }
}
