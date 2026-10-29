use crate::ActionRef;

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct SemanticElement {
    pub tag: String,
    pub role: Option<String>,
    pub name: Option<String>,
    pub text: Option<String>,
    pub href: Option<String>,
    pub action: Option<ActionRef>,
}

impl SemanticElement {
    pub fn new(tag: impl Into<String>) -> Self {
        Self {
            tag: tag.into(),
            role: None,
            name: None,
            text: None,
            href: None,
            action: None,
        }
    }

    pub fn accessible_name(&self) -> Option<&str> {
        self.name
            .as_deref()
            .filter(|s| !s.trim().is_empty())
            .or_else(|| self.text.as_deref().filter(|s| !s.trim().is_empty()))
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct SemanticSnapshot {
    pub url: String,
    pub title: String,
    pub elements: Vec<SemanticElement>,
}

impl SemanticSnapshot {
    pub fn actionable(&self) -> impl Iterator<Item = &SemanticElement> {
        self.elements.iter().filter(|element| element.action.is_some())
    }

    /// Resolve an action reference against this exact snapshot. Adapters should
    /// resolve immediately before acting so stale references cannot silently drift.
    pub fn find_action(&self, id: &str) -> Option<&SemanticElement> {
        self.actionable()
            .find(|element| element.action.as_ref().is_some_and(|action| action.id == id))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn name_prefers_accessible_name() {
        let mut element = SemanticElement::new("button");
        element.name = Some("Save".into());
        element.text = Some("Save draft".into());
        assert_eq!(element.accessible_name(), Some("Save"));
    }

    #[test]
    fn resolves_action_only_from_current_snapshot() {
        let mut element = SemanticElement::new("button");
        element.action = ActionRef::new("b1", "Save");
        let snapshot = SemanticSnapshot {
            url: "https://example.com".into(),
            title: "Example".into(),
            elements: vec![element],
        };
        assert_eq!(snapshot.find_action("b1").unwrap().tag, "button");
        assert!(snapshot.find_action("b2").is_none());
    }
}
