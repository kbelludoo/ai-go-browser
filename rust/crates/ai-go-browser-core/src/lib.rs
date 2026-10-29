//! Rust rewrite foundation for AI Go Browser.
#![forbid(unsafe_code)]

/// Stable semantic reference exposed to an agent.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct ActionRef {
    /// Reference identifier, e.g. b1, l2, or f1.
    pub id: String,
    /// Human-readable action label.
    pub label: String,
}

impl ActionRef {
    /// Creates a reference when both fields are non-empty.
    pub fn new(id: impl Into<String>, label: impl Into<String>) -> Option<Self> {
        let id = id.into();
        let label = label.into();
        if id.trim().is_empty() || label.trim().is_empty() { return None; }
        Some(Self { id, label })
    }
}

#[cfg(test)]
mod tests {
    use super::ActionRef;
    #[test]
    fn accepts_non_empty_reference_and_label() {
        assert_eq!(ActionRef::new("b1", "Submit").unwrap().id, "b1");
    }
    #[test]
    fn rejects_empty_reference_or_label() {
        assert!(ActionRef::new("", "Submit").is_none());
        assert!(ActionRef::new("b1", " ").is_none());
    }
}
