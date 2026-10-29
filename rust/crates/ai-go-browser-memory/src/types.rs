use std::collections::HashMap;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// DomainPlaybook stores learned knowledge, successful selectors, routes, and connection profiles for a web domain.
#[derive(Clone, Debug, Default, Serialize, Deserialize)]
pub struct DomainPlaybook {
    pub domain: String,
    #[serde(default)]
    pub fast_paths: HashMap<String, String>,
    #[serde(default)]
    pub known_selectors: HashMap<String, String>,
    #[serde(default)]
    pub dismiss_modals: Vec<String>,
    #[serde(default)]
    pub requires_warp: bool,
    #[serde(default)]
    pub insecure_tls: bool,
    #[serde(default)]
    pub last_status_code: i32,
    #[serde(default)]
    pub last_diagnosis: String,
    #[serde(default)]
    pub success_count: i32,
    #[serde(default)]
    pub failure_count: i32,
    #[serde(default = "Utc::now")]
    pub last_visited: DateTime<Utc>,
    #[serde(default = "Utc::now")]
    pub updated_at: DateTime<Utc>,
}

/// PageRecord represents an indexed web page with full text for instant retrieval.
#[derive(Clone, Debug, Default, Serialize, Deserialize)]
pub struct PageRecord {
    pub url: String,
    pub title: String,
    pub domain: String,
    #[serde(default)]
    pub snippet: String,
    #[serde(default)]
    pub content: String,
    #[serde(default)]
    pub rank: f64,
    #[serde(default = "Utc::now")]
    pub visited_at: DateTime<Utc>,
    #[serde(default)]
    pub status: i32,
}

/// ActionFeedback represents real-time execution feedback for self-healing playbooks.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ActionFeedback {
    pub domain: String,
    pub action: String,
    pub selector: String,
    pub success: bool,
    pub latency_ms: i64,
}

/// HandoffSession allows transferring live navigation context between agents.
#[derive(Clone, Debug, Default, Serialize, Deserialize)]
pub struct HandoffSession {
    pub id: String,
    pub url: String,
    pub title: String,
    pub goal: String,
    #[serde(default)]
    pub steps_taken: Vec<String>,
    #[serde(default)]
    pub cookies: HashMap<String, String>,
    #[serde(default = "Utc::now")]
    pub saved_at: DateTime<Utc>,
}
