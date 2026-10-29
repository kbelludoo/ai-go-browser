use std::{
    net::{IpAddr, SocketAddr},
    time::Duration,
};

use reqwest::{header::LOCATION, redirect::Policy, Client, StatusCode};
use url::{Host, Url};

use crate::url_policy::{is_non_public_ipv4, is_non_public_ipv6};
use crate::{validate_url, UrlPolicyError};

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum TransportError {
    Policy(UrlPolicyError),
    Dns(String),
    Request(String),
    TooManyRedirects,
    ResponseTooLarge { limit: usize },
    InvalidRedirect,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct HttpResponse {
    pub final_url: String,
    pub status: u16,
    pub content_type: Option<String>,
    pub body: Vec<u8>,
}

/// Minimal HTTP GET transport with explicit DNS validation, address pinning,
/// redirect revalidation, timeouts, and a response-size cap.
///
/// It deliberately disables environment proxies because a proxy would bypass
/// the pinned destination-address guarantee. Deployments requiring a proxy
/// should provide a separately reviewed, policy-enforcing proxy adapter.
#[derive(Clone, Debug)]
pub struct SafeHttpTransport {
    timeout: Duration,
    max_response_bytes: usize,
    max_redirects: usize,
}

impl Default for SafeHttpTransport {
    fn default() -> Self {
        Self::new(Duration::from_secs(15), 4 * 1024 * 1024, 5)
    }
}

impl SafeHttpTransport {
    pub fn new(timeout: Duration, max_response_bytes: usize, max_redirects: usize) -> Self {
        Self { timeout, max_response_bytes, max_redirects }
    }

    pub async fn get(&self, input: &str) -> Result<HttpResponse, TransportError> {
        let mut current = Url::parse(input).map_err(|_| TransportError::Policy(
            UrlPolicyError::MissingHost
        ))?;

        for redirect_count in 0..=self.max_redirects {
            validate_url(current.as_str()).map_err(TransportError::Policy)?;
            let (host, addresses) = resolve_public_addresses(&current).await?;
            let mut builder = Client::builder()
                .redirect(Policy::none())
                .timeout(self.timeout)
                .no_proxy();

            if let Some(domain) = host {
                builder = builder.resolve_to_addrs(&domain, &addresses);
            }

            let client = builder.build().map_err(|e| TransportError::Request(e.to_string()))?;
            let response = client.get(current.clone()).send().await
                .map_err(|e| TransportError::Request(e.to_string()))?;

            if is_redirect(response.status()) {
                if redirect_count == self.max_redirects {
                    return Err(TransportError::TooManyRedirects);
                }
                let location = response.headers().get(LOCATION)
                    .and_then(|value| value.to_str().ok())
                    .ok_or(TransportError::InvalidRedirect)?;
                current = current.join(location).map_err(|_| TransportError::InvalidRedirect)?;
                continue;
            }

            let status = response.status().as_u16();
            let content_type = response.headers().get(reqwest::header::CONTENT_TYPE)
                .and_then(|value| value.to_str().ok()).map(str::to_owned);
            if response.content_length().is_some_and(|n| n > self.max_response_bytes as u64) {
                return Err(TransportError::ResponseTooLarge { limit: self.max_response_bytes });
            }

            let mut response = response;
            let mut body = Vec::new();
            while let Some(chunk) = response.chunk().await
                .map_err(|e| TransportError::Request(e.to_string()))? {
                if body.len().saturating_add(chunk.len()) > self.max_response_bytes {
                    return Err(TransportError::ResponseTooLarge { limit: self.max_response_bytes });
                }
                body.extend_from_slice(&chunk);
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

fn is_redirect(status: StatusCode) -> bool {
    matches!(status.as_u16(), 301 | 302 | 303 | 307 | 308)
}

async fn resolve_public_addresses(url: &Url) -> Result<(Option<String>, Vec<SocketAddr>), TransportError> {
    let port = url.port_or_known_default().ok_or(TransportError::InvalidRedirect)?;
    match url.host().ok_or(TransportError::Policy(UrlPolicyError::MissingHost))? {
        Host::Ipv4(ip) => {
            if is_non_public_ipv4(ip) {
                return Err(TransportError::Policy(UrlPolicyError::PrivateAddress));
            }
            Ok((None, vec![SocketAddr::new(IpAddr::V4(ip), port)]))
        }
        Host::Ipv6(ip) => {
            if is_non_public_ipv6(ip) {
                return Err(TransportError::Policy(UrlPolicyError::PrivateAddress));
            }
            Ok((None, vec![SocketAddr::new(IpAddr::V6(ip), port)]))
        }
        Host::Domain(domain) => {
            let found = tokio::net::lookup_host((domain, port)).await
                .map_err(|e| TransportError::Dns(e.to_string()))?
                .collect::<Vec<_>>();
            if found.is_empty() {
                return Err(TransportError::Dns("hostname resolved to no addresses".into()));
            }
            // Fail closed if any answer is non-public; pin the remaining connection
            // to the exact DNS answers checked above to prevent a second lookup race.
            for address in &found {
                let unsafe_address = match address.ip() {
                    IpAddr::V4(ip) => is_non_public_ipv4(ip),
                    IpAddr::V6(ip) => is_non_public_ipv6(ip),
                };
                if unsafe_address {
                    return Err(TransportError::Policy(UrlPolicyError::PrivateAddress));
                }
            }
            Ok((Some(domain.to_owned()), found))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

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
