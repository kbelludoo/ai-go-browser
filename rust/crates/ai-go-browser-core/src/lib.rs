#![forbid(unsafe_code)]

mod action;
mod browser;
mod memory;
mod parser;
mod semantic;
mod session;
mod url_policy;

pub use action::ActionRef;
pub use browser::{Browser, BrowserError};
pub use memory::{MemoryEntry, MemoryStore};
pub use parser::parse_semantic_html;
pub use semantic::{SemanticElement, SemanticSnapshot};
pub use session::Session;
pub use url_policy::{validate_url, UrlPolicyError};

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn action_ref_rejects_blank_id_and_label() {
        assert!(ActionRef::new(" ", "Submit").is_none());
        assert!(ActionRef::new("b1", " ").is_none());
    }
}
