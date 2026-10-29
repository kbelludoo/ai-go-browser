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
}

impl Default for HttpSnapshotBrowser {
    fn default() -> Self {
        Self::new(SafeHttpTransport::default())
    }
}

impl HttpSnapshotBrowser {
    pub fn new(transport: SafeHttpTransport) -> Self {
        Self { transport, snapshot: None, current_text: None }
    }
}

#[async_trait]
impl Browser for HttpSnapshotBrowser {
    async fn navigate(&mut self, url: &str) -> Result<SemanticSnapshot, BrowserError> {
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
        self.current_text = Some(text);
        self.snapshot = Some(page.snapshot.clone());
        Ok(page.snapshot)
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
            BrowserAction::Back | BrowserAction::Forward | BrowserAction::Reload => Err(
                BrowserError::Unsupported("history navigation is not implemented".into())
            ),
        }
    }

    fn current_snapshot(&self) -> Option<&SemanticSnapshot> {
        self.snapshot.as_ref()
    }
}
