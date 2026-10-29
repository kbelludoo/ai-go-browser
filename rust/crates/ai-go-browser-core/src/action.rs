use std::fmt;

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ActionRef {
    pub id: String,
    pub label: String,
}

impl ActionRef {
    pub fn new(id: impl Into<String>, label: impl Into<String>) -> Option<Self> {
        let id = id.into();
        let label = label.into();
        if id.trim().is_empty() || label.trim().is_empty() {
            return None;
        }
        Some(Self { id, label })
    }
}

impl fmt::Display for ActionRef {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}: {}", self.id, self.label)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn valid_ref() {
        assert_eq!(ActionRef::new("a1", "Submit").unwrap().id, "a1");
    }
    #[test]
    fn rejects_empty() {
        assert!(ActionRef::new(" ", "Submit").is_none());
    }
}
