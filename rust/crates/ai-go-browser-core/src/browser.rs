use crate::{BrowserAction, SemanticSnapshot, UrlPolicyError};

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum BrowserError {
    Policy(UrlPolicyError),
    Navigation(String),
    ActionTargetNotFound(String),
    Unsupported(String),
}

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct BrowserOutput {
    pub snapshot: Option<SemanticSnapshot>,
    pub text: Option<String>,
}

/// Adapter boundary for browser implementations. A concrete implementation must
/// enforce URL policy, transport SSRF controls, action-target validation, and
/// timeouts before performing side effects.
pub trait Browser {
    /// Navigate only after applying URL policy and transport-level SSRF checks.
    fn navigate(&mut self, url: &str) -> Result<SemanticSnapshot, BrowserError>;

    /// Execute a semantic command and return the updated snapshot and/or extracted text.
    fn perform(&mut self, action: BrowserAction) -> Result<BrowserOutput, BrowserError>;

    fn current_snapshot(&self) -> Option<&SemanticSnapshot>;
}
