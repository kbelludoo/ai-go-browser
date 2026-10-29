use std::sync::Arc;

use async_trait::async_trait;

use crate::{
    BrowserAction, FetchedPage, SafeHttpTransport, SemanticSnapshot, TransportError, UrlPolicyError,
};

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

#[async_trait]
trait PageFetcher: Send + Sync {
    async fn get_page(&self, url: &str) -> Result<FetchedPage, TransportError>;
}

#[async_trait]
impl PageFetcher for SafeHttpTransport {
    async fn get_page(&self, url: &str) -> Result<FetchedPage, TransportError> {
        SafeHttpTransport::get_page(self, url).await
    }
}

/// HTTP-backed static-page adapter. It fetches and parses HTML but does not run
/// JavaScript or emulate a live DOM; mutating element actions are intentionally
/// unsupported until a real browser engine is integrated.
pub struct HttpSnapshotBrowser {
    fetcher: Arc<dyn PageFetcher>,
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
        Self::with_fetcher(Arc::new(transport))
    }

    fn with_fetcher(fetcher: Arc<dyn PageFetcher>) -> Self {
        Self {
            fetcher,
            snapshot: None,
            current_text: None,
            history: Vec::new(),
            history_index: None,
        }
    }

    async fn fetch_static_page(
        &self,
        url: &str,
    ) -> Result<(SemanticSnapshot, String), BrowserError> {
        let page = self
            .fetcher
            .get_page(url)
            .await
            .map_err(|error| match error {
                TransportError::Policy(policy) => BrowserError::Policy(policy),
                other => BrowserError::Transport(other),
            })?;
        let html = page.decoded.text.as_str();
        let document = scraper::Html::parse_document(html);
        let body_selector = scraper::Selector::parse("body").expect("static selector");
        let text = document
            .select(&body_selector)
            .flat_map(|node| node.text())
            .collect::<Vec<_>>()
            .join(" ")
            .split_whitespace()
            .collect::<Vec<_>>()
            .join(" ");
        Ok((page.snapshot, text))
    }

    async fn load_history_entry(&mut self, index: usize) -> Result<SemanticSnapshot, BrowserError> {
        let url = self
            .history
            .get(index)
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
            let found = self
                .snapshot
                .as_ref()
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
                    Ok(BrowserOutput {
                        snapshot: self.snapshot.clone(),
                        text: Some(current.to_owned()),
                    })
                } else {
                    Err(BrowserError::Navigation(format!(
                        "text not found in static snapshot: {text}"
                    )))
                }
            }
            BrowserAction::Click { .. }
            | BrowserAction::Fill { .. }
            | BrowserAction::SelectOption { .. }
            | BrowserAction::Submit { .. } => Err(BrowserError::Unsupported(
                "mutating actions require a JavaScript-capable browser engine".into(),
            )),
            BrowserAction::Back => {
                let index = self
                    .history_index
                    .ok_or_else(|| BrowserError::Navigation("no page in history".into()))?;
                if index == 0 {
                    return Err(BrowserError::Navigation(
                        "already at the first history entry".into(),
                    ));
                }
                let snapshot = self.load_history_entry(index - 1).await?;
                Ok(BrowserOutput {
                    snapshot: Some(snapshot),
                    text: self.current_text.clone(),
                })
            }
            BrowserAction::Forward => {
                let index = self
                    .history_index
                    .ok_or_else(|| BrowserError::Navigation("no page in history".into()))?;
                if index + 1 >= self.history.len() {
                    return Err(BrowserError::Navigation(
                        "already at the latest history entry".into(),
                    ));
                }
                let snapshot = self.load_history_entry(index + 1).await?;
                Ok(BrowserOutput {
                    snapshot: Some(snapshot),
                    text: self.current_text.clone(),
                })
            }
            BrowserAction::Reload => {
                let index = self
                    .history_index
                    .ok_or_else(|| BrowserError::Navigation("no page to reload".into()))?;
                let snapshot = self.load_history_entry(index).await?;
                Ok(BrowserOutput {
                    snapshot: Some(snapshot),
                    text: self.current_text.clone(),
                })
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
    use std::collections::HashMap;
    use std::sync::Mutex;

    struct MockFetcher {
        pages: HashMap<String, FetchedPage>,
        requests: Mutex<Vec<String>>,
    }

    #[async_trait]
    impl PageFetcher for MockFetcher {
        async fn get_page(&self, url: &str) -> Result<FetchedPage, TransportError> {
            self.requests.lock().unwrap().push(url.to_owned());
            self.pages
                .get(url)
                .cloned()
                .ok_or_else(|| TransportError::Request(format!("unexpected mock URL: {url}")))
        }
    }

    fn mock_page(url: &str, title: &str, body: &str) -> FetchedPage {
        let html = format!("<html><head><title>{title}</title></head><body>{body}</body></html>");
        FetchedPage {
            response: crate::HttpResponse::new(
                url,
                200,
                Some("text/html".into()),
                html.clone().into_bytes(),
            ),
            decoded: crate::DecodedBody {
                text: html,
                encoding: "utf-8".into(),
                bom_bytes: 0,
            },
            snapshot: SemanticSnapshot {
                url: url.to_owned(),
                title: title.to_owned(),
                elements: vec![],
            },
        }
    }

    fn mock_browser() -> (HttpSnapshotBrowser, Arc<MockFetcher>) {
        let fetcher = Arc::new(MockFetcher {
            pages: HashMap::from([
                (
                    "https://example.com/a".into(),
                    mock_page("https://example.com/a", "A", "Alpha"),
                ),
                (
                    "https://example.com/b".into(),
                    mock_page("https://example.com/b", "B", "Beta"),
                ),
                (
                    "https://example.com/c".into(),
                    mock_page("https://example.com/c", "C", "Gamma"),
                ),
            ]),
            requests: Mutex::new(Vec::new()),
        });
        (HttpSnapshotBrowser::with_fetcher(fetcher.clone()), fetcher)
    }

    #[tokio::test]
    async fn history_back_forward_reload_and_branch_truncation_work() {
        let (mut browser, fetcher) = mock_browser();

        browser.navigate("https://example.com/a").await.unwrap();
        browser.navigate("https://example.com/b").await.unwrap();

        let back = browser.perform(BrowserAction::Back).await.unwrap();
        assert_eq!(back.snapshot.as_ref().unwrap().title, "A");
        assert_eq!(back.text.as_deref(), Some("Alpha"));

        let forward = browser.perform(BrowserAction::Forward).await.unwrap();
        assert_eq!(forward.snapshot.as_ref().unwrap().title, "B");
        assert_eq!(forward.text.as_deref(), Some("Beta"));

        browser.perform(BrowserAction::Back).await.unwrap();
        browser.navigate("https://example.com/c").await.unwrap();
        assert!(matches!(
            browser.perform(BrowserAction::Forward).await,
            Err(BrowserError::Navigation(_))
        ));

        let reloaded = browser.perform(BrowserAction::Reload).await.unwrap();
        assert_eq!(reloaded.snapshot.as_ref().unwrap().title, "C");
        assert_eq!(
            fetcher.requests.lock().unwrap().as_slice(),
            [
                "https://example.com/a",
                "https://example.com/b",
                "https://example.com/a",
                "https://example.com/b",
                "https://example.com/a",
                "https://example.com/c",
                "https://example.com/c",
            ]
        );
    }

    #[tokio::test]
    async fn extracts_text_from_current_static_snapshot() {
        let mut browser = HttpSnapshotBrowser {
            snapshot: Some(SemanticSnapshot {
                url: "https://example.com".into(),
                title: "Example".into(),
                elements: vec![],
            }),
            current_text: Some("Hello from the page".into()),
            ..HttpSnapshotBrowser::default()
        };

        let output = browser.perform(BrowserAction::ExtractText).await.unwrap();
        assert_eq!(output.text.as_deref(), Some("Hello from the page"));
        assert_eq!(output.snapshot.unwrap().title, "Example");
    }

    #[tokio::test]
    async fn rejects_stale_or_forged_action_references() {
        let action = ActionRef::new("b1", "Save").unwrap();
        let mut element = SemanticElement::new("button");
        element.name = Some("Save".into());
        element.action = Some(action.clone());
        let mut browser = HttpSnapshotBrowser {
            snapshot: Some(SemanticSnapshot {
                url: "https://example.com".into(),
                title: "Example".into(),
                elements: vec![element],
            }),
            ..HttpSnapshotBrowser::default()
        };

        let forged = ActionRef::new("b1", "Delete").unwrap();
        let result = browser
            .perform(BrowserAction::Click { target: forged })
            .await;
        assert_eq!(result, Err(BrowserError::ActionTargetNotFound("b1".into())));
        assert!(
            browser
                .perform(BrowserAction::Click { target: action })
                .await
                .is_err()
        );
    }

    #[tokio::test]
    async fn explicitly_rejects_mutating_actions_without_browser_engine() {
        let target = ActionRef::new("b1", "Save").unwrap();
        let mut element = SemanticElement::new("button");
        element.name = Some("Save".into());
        element.action = Some(target.clone());
        let mut browser = HttpSnapshotBrowser {
            snapshot: Some(SemanticSnapshot {
                url: "https://example.com".into(),
                title: "Example".into(),
                elements: vec![element],
            }),
            ..HttpSnapshotBrowser::default()
        };

        assert_eq!(
            browser.perform(BrowserAction::Click { target }).await,
            Err(BrowserError::Unsupported(
                "mutating actions require a JavaScript-capable browser engine".into()
            ))
        );
    }
}
