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
        Self { tag: tag.into(), role: None, name: None, text: None, href: None, action: None }
    }
    pub fn accessible_name(&self) -> Option<&str> {
        self.name.as_deref().filter(|s| !s.trim().is_empty())
            .or_else(|| self.text.as_deref().filter(|s| !s.trim().is_empty()))
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct SemanticSnapshot { pub url: String, pub title: String, pub elements: Vec<SemanticElement> }

impl SemanticSnapshot {
    pub fn actionable(&self) -> impl Iterator<Item=&SemanticElement> {
        self.elements.iter().filter(|e| e.action.is_some())
    }
}

#[cfg(test)]
mod tests {
 use super::*;
 #[test] fn name_prefers_accessible_name() {
  let mut e = SemanticElement::new("button"); e.name=Some("Save".into()); e.text=Some("Save draft".into());
  assert_eq!(e.accessible_name(), Some("Save"));
 }
}
