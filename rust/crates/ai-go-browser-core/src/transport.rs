use std::{
    fmt,
    net::{IpAddr, SocketAddr},
    sync::Arc,
    time::Duration,
};

use reqwest::{Client, StatusCode, header::LOCATION, redirect::Policy};
use url::{Host, Url};

use crate::UrlPolicyError;
use crate::destination::{DestinationPolicy, HostResolver, PublicInternet, SystemResolver};
use crate::url_policy::validate_syntax;

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum TransportError {
    Policy(UrlPolicyError),
    Dns(String),
    Request(String),
    TooManyRedirects,
    ResponseTooLarge { limit: usize },
    InvalidRedirect,
    InvalidUtf8,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct HttpResponse {
    pub final_url: String,
    pub status: u16,
    pub content_type: Option<String>,
    pub body: Vec<u8>,
}

/// An HTTP response parsed into the project's semantic HTML snapshot.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FetchedPage {
    pub response: HttpResponse,
    pub snapshot: crate::SemanticSnapshot,
}

impl FetchedPage {
    fn from_response(response: HttpResponse) -> Result<Self, TransportError> {
        let html = std::str::from_utf8(&response.body).map_err(|_| TransportError::InvalidUtf8)?;
        let snapshot = crate::parse_semantic_html(response.final_url.clone(), html);
        Ok(Self { response, snapshot })
    }
}

/// Minimal HTTP GET transport with explicit DNS validation, address pinning,
/// redirect revalidation, timeouts, and a response-size cap.
///
/// It deliberately disables environment proxies because a proxy would bypass
/// the pinned destination-address guarantee. Deployments requiring a proxy
/// should provide a separately reviewed, policy-enforcing proxy adapter.
#[derive(Clone)]
pub struct SafeHttpTransport {
    timeout: Duration,
    max_response_bytes: usize,
    max_redirects: usize,
    policy: Arc<dyn DestinationPolicy>,
    resolver: Arc<dyn HostResolver>,
}

impl fmt::Debug for SafeHttpTransport {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("SafeHttpTransport")
            .field("timeout", &self.timeout)
            .field("max_response_bytes", &self.max_response_bytes)
            .field("max_redirects", &self.max_redirects)
            .finish_non_exhaustive()
    }
}

impl Default for SafeHttpTransport {
    fn default() -> Self {
        Self::new(Duration::from_secs(15), 4 * 1024 * 1024, 5)
    }
}

impl SafeHttpTransport {
    /// Build a transport with the production policy and the system resolver.
    pub fn new(timeout: Duration, max_response_bytes: usize, max_redirects: usize) -> Self {
        Self::with_adapters(
            timeout,
            max_response_bytes,
            max_redirects,
            Arc::new(PublicInternet),
            Arc::new(SystemResolver),
        )
    }

    /// Build a transport with explicit destination policy and resolver seams.
    ///
    /// The policy is consulted for every hop URL and every resolved address, so
    /// a substitute policy narrows the public default instead of bypassing the
    /// chain. Intended for adapters and network integration tests.
    pub fn with_adapters(
        timeout: Duration,
        max_response_bytes: usize,
        max_redirects: usize,
        policy: Arc<dyn DestinationPolicy>,
        resolver: Arc<dyn HostResolver>,
    ) -> Self {
        Self {
            timeout,
            max_response_bytes,
            max_redirects,
            policy,
            resolver,
        }
    }

    /// Fetch a page and parse its response body as UTF-8 HTML.
    /// This is a static HTML snapshot, not a JavaScript-capable browser DOM.
    pub async fn get_page(&self, input: &str) -> Result<FetchedPage, TransportError> {
        FetchedPage::from_response(self.get(input).await?)
    }

    pub async fn get(&self, input: &str) -> Result<HttpResponse, TransportError> {
        let mut current = validate_syntax(input).map_err(TransportError::Policy)?;

        for redirect_count in 0..=self.max_redirects {
            self.policy
                .check_url(&current)
                .map_err(TransportError::Policy)?;
            let (host, addresses) = self.resolve_addresses(&current).await?;
            let mut builder = Client::builder()
                .redirect(Policy::none())
                .timeout(self.timeout)
                .no_proxy();

            if let Some(domain) = host {
                builder = builder.resolve_to_addrs(&domain, &addresses);
            }

            let client = builder
                .build()
                .map_err(|e| TransportError::Request(e.to_string()))?;
            let response = client
                .get(current.clone())
                .send()
                .await
                .map_err(|e| TransportError::Request(e.to_string()))?;

            if is_redirect(response.status()) {
                if redirect_count == self.max_redirects {
                    return Err(TransportError::TooManyRedirects);
                }
                let location = response
                    .headers()
                    .get(LOCATION)
                    .and_then(|value| value.to_str().ok())
                    .ok_or(TransportError::InvalidRedirect)?;
                current = resolve_redirect(self.policy.as_ref(), &current, location)?;
                continue;
            }

            let status = response.status().as_u16();
            let content_type = response
                .headers()
                .get(reqwest::header::CONTENT_TYPE)
                .and_then(|value| value.to_str().ok())
                .map(str::to_owned);
            if response
                .content_length()
                .is_some_and(|n| n > self.max_response_bytes as u64)
            {
                return Err(TransportError::ResponseTooLarge {
                    limit: self.max_response_bytes,
                });
            }

            let mut response = response;
            let mut body = Vec::new();
            while let Some(chunk) = response
                .chunk()
                .await
                .map_err(|e| TransportError::Request(e.to_string()))?
            {
                append_limited_body(&mut body, &chunk, self.max_response_bytes)?;
            }

            return Ok(HttpResponse {
                final_url: current.to_string(),
                status,
                content_type,
                body,
            });
        }

        Err(TransportError::TooManyRedirects)
    }
}

/// Resolve and validate a redirect destination before any subsequent DNS lookup or request.
fn resolve_redirect(
    policy: &dyn DestinationPolicy,
    current: &Url,
    location: &str,
) -> Result<Url, TransportError> {
    let target = current
        .join(location)
        .map_err(|_| TransportError::InvalidRedirect)?;
    validate_syntax(target.as_str()).map_err(TransportError::Policy)?;
    policy.check_url(&target).map_err(TransportError::Policy)?;
    Ok(target)
}

/// Append one streamed response chunk without allowing the body to exceed its cap.
fn append_limited_body(
    body: &mut Vec<u8>,
    chunk: &[u8],
    limit: usize,
) -> Result<(), TransportError> {
    if body.len().saturating_add(chunk.len()) > limit {
        return Err(TransportError::ResponseTooLarge { limit });
    }
    body.extend_from_slice(chunk);
    Ok(())
}

fn is_redirect(status: StatusCode) -> bool {
    matches!(status.as_u16(), 301 | 302 | 303 | 307 | 308)
}

impl SafeHttpTransport {
    /// Resolve the hop destination through the injected resolver and require the
    /// policy to accept every answer, then pin exactly those answers.
    async fn resolve_addresses(
        &self,
        url: &Url,
    ) -> Result<(Option<String>, Vec<SocketAddr>), TransportError> {
        let port = url
            .port_or_known_default()
            .ok_or(TransportError::InvalidRedirect)?;
        let reject = |address: IpAddr| {
            self.policy
                .check_address(address)
                .map_err(TransportError::Policy)
        };
        match url
            .host()
            .ok_or(TransportError::Policy(UrlPolicyError::MissingHost))?
        {
            Host::Ipv4(ip) => {
                reject(IpAddr::V4(ip))?;
                Ok((None, vec![SocketAddr::new(IpAddr::V4(ip), port)]))
            }
            Host::Ipv6(ip) => {
                reject(IpAddr::V6(ip))?;
                Ok((None, vec![SocketAddr::new(IpAddr::V6(ip), port)]))
            }
            Host::Domain(domain) => {
                let found = self.resolver.resolve(domain, port).await?;
                // Fail closed if any answer is rejected; pin the remaining
                // connection to the exact answers checked above.
                for address in &found {
                    reject(address.ip())?;
                }
                Ok((Some(domain.to_owned()), found))
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn rejects_private_literal_before_opening_a_connection() {
        for url in [
            "http://127.0.0.1:1/",
            "http://10.0.0.1:1/",
            "http://169.254.169.254/latest/meta-data/",
            "http://[::1]:1/",
        ] {
            let error = SafeHttpTransport::default().get(url).await.unwrap_err();
            assert_eq!(
                error,
                TransportError::Policy(UrlPolicyError::PrivateAddress),
                "{url}"
            );
        }
    }

    #[tokio::test]
    async fn rejects_localhost_before_dns_resolution() {
        let error = SafeHttpTransport::default()
            .get("http://localhost:1/")
            .await
            .unwrap_err();
        assert_eq!(error, TransportError::Policy(UrlPolicyError::LocalHost));
    }

    #[tokio::test]
    async fn rejects_non_http_schemes_before_network_access() {
        let error = SafeHttpTransport::default()
            .get("file:///etc/passwd")
            .await
            .unwrap_err();
        assert_eq!(
            error,
            TransportError::Policy(UrlPolicyError::UnsupportedScheme)
        );
    }

    #[test]
    fn parses_response_body_into_semantic_snapshot() {
        let response = HttpResponse {
            final_url: "https://example.com/page".into(),
            status: 200,
            content_type: Some("text/html; charset=utf-8".into()),
            body: b"<title>Demo</title><button>Continue</button>".to_vec(),
        };
        let page = FetchedPage::from_response(response).unwrap();
        assert_eq!(page.snapshot.url, "https://example.com/page");
        assert_eq!(page.snapshot.title, "Demo");
        assert_eq!(page.snapshot.actionable().count(), 1);
    }

    #[test]
    fn rejects_non_utf8_html_response() {
        let response = HttpResponse {
            final_url: "https://example.com/".into(),
            status: 200,
            content_type: Some("text/html".into()),
            body: vec![0xff, 0xfe],
        };
        assert_eq!(
            FetchedPage::from_response(response),
            Err(TransportError::InvalidUtf8)
        );
    }

    #[test]
    fn redirect_resolution_rejects_private_destinations_before_fetching() {
        let current = Url::parse("https://example.com/start").unwrap();
        for location in [
            "http://127.0.0.1/admin",
            "http://localhost/admin",
            "file:///etc/passwd",
        ] {
            let error = resolve_redirect(&PublicInternet, &current, location).unwrap_err();
            assert!(
                matches!(error, TransportError::Policy(_)),
                "expected policy rejection for {location}, got {error:?}"
            );
        }
    }

    #[test]
    fn redirect_resolution_supports_relative_locations() {
        let current = Url::parse("https://example.com/a/page").unwrap();
        let target = resolve_redirect(&PublicInternet, &current, "../next").unwrap();
        assert_eq!(target.as_str(), "https://example.com/next");
    }

    #[test]
    fn streamed_body_limit_accepts_exact_limit() {
        let mut body = Vec::new();
        append_limited_body(&mut body, b"abc", 5).unwrap();
        append_limited_body(&mut body, b"de", 5).unwrap();
        assert_eq!(body, b"abcde");
    }

    #[test]
    fn streamed_body_limit_rejects_chunked_overflow_without_appending_it() {
        let mut body = b"abc".to_vec();
        let error = append_limited_body(&mut body, b"def", 5).unwrap_err();
        assert_eq!(error, TransportError::ResponseTooLarge { limit: 5 });
        assert_eq!(body, b"abc");
    }

    #[test]
    fn default_limits_are_conservative() {
        let transport = SafeHttpTransport::default();
        assert_eq!(transport.timeout, Duration::from_secs(15));
        assert_eq!(transport.max_response_bytes, 4 * 1024 * 1024);
        assert_eq!(transport.max_redirects, 5);
    }

    #[test]
    fn recognizes_only_supported_redirect_statuses() {
        assert!(is_redirect(StatusCode::MOVED_PERMANENTLY));
        assert!(is_redirect(StatusCode::FOUND));
        assert!(is_redirect(StatusCode::TEMPORARY_REDIRECT));
        assert!(!is_redirect(StatusCode::OK));
        assert!(!is_redirect(StatusCode::BAD_REQUEST));
    }
}
