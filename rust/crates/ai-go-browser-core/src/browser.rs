use async_trait::async_trait;

use crate::{BrowserAction, SafeHttpTransport, SemanticSnapshot, TransportError, UrlPolicyError};

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum BrowserError {
    Policy(UrlPolicyError),
    Navigation(String),
    Transport(TransportError),
    ActionTargetNotFound(String),
    Unsupported(String),
}

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct BrowserOutput {
    pub snapshot: Option<SemanticSnapshot>,
    pub text: Option<String>,
}

/// Async browser adapter contract. Implementations must validate navigation
/// destinations and revalidate action references against the current snapshot.
#[async_trait]
pub trait Browser {
    async fn navigate(&mut self, url: &str) -> Result<SemanticSnapshot, BrowserError>;
    async fn perform(&mut self, action: BrowserAction) -> Result<BrowserOutput, BrowserError>;
    fn current_snapshot(&self) -> Option<&SemanticSnapshot>;
}

/// HTTP-backed static-page adapter. It fetches and parses HTML but does not run
/// JavaScript or emulate a live DOM; mutating element actions are intentionally
/// unsupported until a real browser engine is integrated.
pub struct HttpSnapshotBrowser {
    transport: SafeHttpTransport,
    snapshot: Option<SemanticSnapshot>,
    current_text: Option<String>,
    history: Vec<String>,
    history_index: Option<usize>,
}

impl Default for HttpSnapshotBrowser {
    fn default() -> Self {
        Self::new(SafeHttpTransport::default())
    }
}

impl HttpSnapshotBrowser {
    pub fn new(transport: SafeHttpTransport) -> Self {
        Self {
            transport,
            snapshot: None,
            current_text: None,
            history: Vec::new(),
            history_index: None,
        }
    }

    async fn fetch_static_page(&self, url: &str) -> Result<(SemanticSnapshot, String), BrowserError> {
        let page = self.transport.get_page(url).await.map_err(|error| {
            match error {
                TransportError::Policy(policy) => BrowserError::Policy(policy),
                other => BrowserError::Transport(other),
            }
        })?;
        let html = std::str::from_utf8(&page.response.body)
            .map_err(|error| BrowserError::Navigation(error.to_string()))?;
        let document = scraper::Html::parse_document(html);
        let body_selector = scraper::Selector::parse("body").expect("static selector");
        let text = document.select(&body_selector)
            .flat_map(|node| node.text())
            .collect::<Vec<_>>()
            .join(" ")
            .split_whitespace()
            .collect::<Vec<_>>()
            .join(" ");
        Ok((page.snapshot, text))
    }

    async fn load_history_entry(&mut self, index: usize) -> Result<SemanticSnapshot, BrowserError> {
        let url = self.history.get(index)
            .cloned()
            .ok_or_else(|| BrowserError::Navigation("history entry does not exist".into()))?;
        let (snapshot, text) = self.fetch_static_page(&url).await?;
        self.snapshot = Some(snapshot.clone());
        self.current_text = Some(text);
        self.history_index = Some(index);
        Ok(snapshot)
    }
}

#[async_trait]
impl Browser for HttpSnapshotBrowser {
    async fn navigate(&mut self, url: &str) -> Result<SemanticSnapshot, BrowserError> {
        let (snapshot, text) = self.fetch_static_page(url).await?;
        if let Some(index) = self.history_index {
            self.history.truncate(index + 1);
        } else {
            self.history.clear();
        }
        self.history.push(snapshot.url.clone());
        self.history_index = Some(self.history.len() - 1);
        self.current_text = Some(text);
        self.snapshot = Some(snapshot.clone());
        Ok(snapshot)
    }

    async fn perform(&mut self, action: BrowserAction) -> Result<BrowserOutput, BrowserError> {
        if let Some(target) = action.target() {
            let found = self.snapshot.as_ref()
                .and_then(|snapshot| snapshot.find_action(&target.id));
            if found.is_none() {
                return Err(BrowserError::ActionTargetNotFound(target.id.clone()));
            }
            if found.and_then(|element| element.action.as_ref()) != Some(target) {
                return Err(BrowserError::ActionTargetNotFound(target.id.clone()));
            }
        }

        match action {
            BrowserAction::ExtractText => Ok(BrowserOutput {
                snapshot: self.snapshot.clone(),
                text: self.current_text.clone(),
            }),
            BrowserAction::WaitForText { text } => {
                let current = self.current_text.as_deref().unwrap_or_default();
                if current.contains(&text) {
                    Ok(BrowserOutput { snapshot: self.snapshot.clone(), text: Some(current.to_owned()) })
                } else {
                    Err(BrowserError::Navigation(format!("text not found in static snapshot: {text}")))
                }
            }
            BrowserAction::Click { .. }
            | BrowserAction::Fill { .. }
            | BrowserAction::SelectOption { .. }
            | BrowserAction::Submit { .. } => Err(BrowserError::Unsupported(
                "mutating actions require a JavaScript-capable browser engine".into(),
            )),
            BrowserAction::Back => {
                let index = self.history_index.ok_or_else(|| {
                    BrowserError::Navigation("no page in history".into())
                })?;
                if index == 0 {
                    return Err(BrowserError::Navigation("already at the first history entry".into()));
                }
                let snapshot = self.load_history_entry(index - 1).await?;
                Ok(BrowserOutput { snapshot: Some(snapshot), text: self.current_text.clone() })
            }
            BrowserAction::Forward => {
                let index = self.history_index.ok_or_else(|| {
                    BrowserError::Navigation("no page in history".into())
                })?;
                if index + 1 >= self.history.len() {
                    return Err(BrowserError::Navigation("already at the latest history entry".into()));
                }
                let snapshot = self.load_history_entry(index + 1).await?;
                Ok(BrowserOutput { snapshot: Some(snapshot), text: self.current_text.clone() })
            }
            BrowserAction::Reload => {
                let index = self.history_index.ok_or_else(|| {
                    BrowserError::Navigation("no page to reload".into())
                })?;
                let snapshot = self.load_history_entry(index).await?;
                Ok(BrowserOutput { snapshot: Some(snapshot), text: self.current_text.clone() })
            }
        }
    }

    fn current_snapshot(&self) -> Option<&SemanticSnapshot> {
        self.snapshot.as_ref()
    }
}


#[cfg(test)]
mod tests {
    use super::*;
    use crate::{ActionRef, SemanticElement};

    #[tokio::test]
    async fn extracts_text_from_current_static_snapshot() {
        let mut browser = HttpSnapshotBrowser::default();
        browser.snapshot = Some(SemanticSnapshot {
            url: "https://example.com".into(),
            title: "Example".into(),
            elements: vec![],
        });
        browser.current_text = Some("Hello from the page".into());

        let output = browser.perform(BrowserAction::ExtractText).await.unwrap();
        assert_eq!(output.text.as_deref(), Some("Hello from the page"));
        assert_eq!(output.snapshot.unwrap().title, "Example");
    }

    #[tokio::test]
    async fn rejects_stale_or_forged_action_references() {
        let mut browser = HttpSnapshotBrowser::default();
        let action = ActionRef::new("b1", "Save").unwrap();
        let mut element = SemanticElement::new("button");
        element.name = Some("Save".into());
        element.action = Some(action.clone());
        browser.snapshot = Some(SemanticSnapshot {
            url: "https://example.com".into(),
            title: "Example".into(),
            elements: vec![element],
        });

        let forged = ActionRef::new("b1", "Delete").unwrap();
        let result = browser.perform(BrowserAction::Click { target: forged }).await;
        assert_eq!(result, Err(BrowserError::ActionTargetNotFound("b1".into())));
        assert!(browser.perform(BrowserAction::Click { target: action }).await.is_err());
    }

    #[tokio::test]
    async fn explicitly_rejects_mutating_actions_without_browser_engine() {
        let mut browser = HttpSnapshotBrowser::default();
        let target = ActionRef::new("b1", "Save").unwrap();
        let mut element = SemanticElement::new("button");
        element.name = Some("Save".into());
        element.action = Some(target.clone());
        browser.snapshot = Some(SemanticSnapshot {
            url: "https://example.com".into(),
            title: "Example".into(),
            elements: vec![element],
        });

        assert_eq!(
            browser.perform(BrowserAction::Click { target }).await,
            Err(BrowserError::Unsupported(
                "mutating actions require a JavaScript-capable browser engine".into()
            ))
        );
    }
}
