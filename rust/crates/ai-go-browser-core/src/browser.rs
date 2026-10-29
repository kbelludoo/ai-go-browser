use crate::{SemanticSnapshot, UrlPolicyError};

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum BrowserError {
    Policy(UrlPolicyError),
    Navigation(String),
    Unsupported(String),
}

pub trait Browser {
    /// Navigate only after applying URL policy and transport-level SSRF checks.
    fn navigate(&mut self, url: &str) -> Result<SemanticSnapshot, BrowserError>;
    fn current_snapshot(&self) -> Option<&SemanticSnapshot>;
}
