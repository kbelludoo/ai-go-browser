//! HTTP transport with explicit DNS validation, address pinning, redirect
//! revalidation, timeouts, a response-size cap and the Go request surface.
//!
//! It deliberately disables environment proxies because a proxy would bypass
//! the pinned destination-address guarantee. Deployments requiring a proxy
//! should provide a separately reviewed, policy-enforcing proxy adapter.

use std::{
    fmt,
    net::{IpAddr, SocketAddr},
    sync::Arc,
    time::Duration,
};

use reqwest::{
    Client, Method,
    cookie::Jar,
    header::{CONTENT_TYPE, LOCATION},
    redirect::Policy,
};
use url::{Host, Url};

use crate::UrlPolicyError;
use crate::destination::{DestinationPolicy, HostResolver, PublicInternet, SystemResolver};
use crate::encoding::{self, DecodedBody};
use crate::url_policy::validate_syntax;

mod headers;
mod hops;

pub use hops::encode_form;

use hops::{append_capped, decode_error as transport_decode_error, is_redirect, resolve_redirect};

/// Go `MaxBody`: the DOM path refuses a larger response instead of truncating.
pub const MAX_BODY: usize = 5 << 20;
/// Go `maxResourceBytes`: resource responses are truncated at this size.
pub const MAX_RESOURCE_BYTES: usize = 96 << 10;
/// Go follows at most 15 redirects and then reports "many redirects".
pub const MAX_REDIRECTS: usize = 15;
/// Go `http.Client.Timeout`.
pub const DEFAULT_TIMEOUT: Duration = Duration::from_secs(20);

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum TransportError {
    Policy(UrlPolicyError),
    Dns(String),
    Request(String),
    TooManyRedirects,
    ResponseTooLarge { limit: usize },
    InvalidRedirect,
    InvalidCharset(String),
    DecodeLimit { limit: usize, decoded: bool },
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct HttpResponse {
    pub final_url: String,
    pub status: u16,
    pub content_type: Option<String>,
    /// Media type after `Content-Type` parsing and, when absent, sniffing.
    pub media_type: String,
    /// Wire bytes, truncated to the resource budget for non-HTML responses.
    pub body: Vec<u8>,
    /// Bytes observed before a resource truncation.
    pub wire_bytes: usize,
    pub truncated: bool,
}

impl HttpResponse {
    /// Build a response from an already-read body, applying the same media-type
    /// derivation as the transport. Used by adapters and tests.
    pub fn new(
        final_url: impl Into<String>,
        status: u16,
        content_type: Option<String>,
        body: Vec<u8>,
    ) -> Self {
        let media_type = content_type
            .as_deref()
            .map(encoding::media_type)
            .unwrap_or_default();
        let wire_bytes = body.len();
        Self {
            final_url: final_url.into(),
            status,
            content_type,
            media_type,
            body,
            wire_bytes,
            truncated: false,
        }
    }
}

/// An HTTP response parsed into the project's semantic HTML snapshot.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FetchedPage {
    pub response: HttpResponse,
    pub decoded: DecodedBody,
    pub snapshot: crate::SemanticSnapshot,
}

impl FetchedPage {
    fn from_response(response: HttpResponse) -> Result<Self, TransportError> {
        let content_type = response.content_type.clone().unwrap_or_default();
        let decoded = encoding::decode(&content_type, &response.body, MAX_BODY)
            .map_err(transport_decode_error)?;
        let snapshot = crate::parse_semantic_html(response.final_url.clone(), &decoded.text);
        Ok(Self {
            response,
            decoded,
            snapshot,
        })
    }
}

/// Minimal HTTP transport with the Go request headers, form submission, cookie
/// jar, redirect policy and body budgets.
#[derive(Clone)]
pub struct SafeHttpTransport {
    timeout: Duration,
    max_response_bytes: usize,
    max_redirects: usize,
    policy: Arc<dyn DestinationPolicy>,
    resolver: Arc<dyn HostResolver>,
    jar: Arc<Jar>,
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
        Self::new(DEFAULT_TIMEOUT, MAX_BODY, MAX_REDIRECTS)
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
            jar: Arc::new(Jar::default()),
        }
    }

    /// Fetch a page and parse its response body into a semantic snapshot.
    /// This is a static HTML snapshot, not a JavaScript-capable browser DOM.
    pub async fn get_page(&self, input: &str) -> Result<FetchedPage, TransportError> {
        FetchedPage::from_response(self.get(input).await?)
    }

    pub async fn get(&self, input: &str) -> Result<HttpResponse, TransportError> {
        self.request(Method::GET, input, None).await
    }

    /// Submit an HTML form. The body is encoded exactly like Go's
    /// `url.Values.Encode`: sorted keys, repeated values preserved.
    pub async fn post_form(
        &self,
        input: &str,
        fields: &[(String, String)],
    ) -> Result<HttpResponse, TransportError> {
        self.request(Method::POST, input, Some(fields)).await
    }

    /// Issue `method` against `input`, following redirects with Go semantics.
    pub async fn request(
        &self,
        method: Method,
        input: &str,
        form: Option<&[(String, String)]>,
    ) -> Result<HttpResponse, TransportError> {
        let mut current = validate_syntax(input).map_err(TransportError::Policy)?;
        let mut method = method;
        if method == Method::GET
            && let Some(fields) = form
        {
            // HTML GET submission replaces the action URL's query component.
            current.set_query(Some(&encode_form(fields)));
        }
        let mut body = if method == Method::POST {
            form.map(encode_form)
        } else {
            None
        };
        let mut redirects = 0;

        loop {
            self.policy
                .check_url(&current)
                .map_err(TransportError::Policy)?;
            let (host, addresses) = self.resolve_addresses(&current).await?;
            let mut builder = Client::builder()
                .redirect(Policy::none())
                .timeout(self.timeout)
                .no_proxy()
                .cookie_provider(self.jar.clone());
            if let Some(domain) = host {
                builder = builder.resolve_to_addrs(&domain, &addresses);
            }
            let client = builder
                .build()
                .map_err(|error| TransportError::Request(error.to_string()))?;

            let mut request = client.request(method.clone(), current.clone());
            for (name, value) in headers::REQUEST_HEADERS {
                request = request.header(name, value);
            }
            if let Some(payload) = &body {
                request = request
                    .header(CONTENT_TYPE, headers::FORM_CONTENT_TYPE)
                    .body(payload.clone());
            }
            let response = request
                .send()
                .await
                .map_err(|error| TransportError::Request(error.to_string()))?;

            if is_redirect(response.status()) {
                if redirects == self.max_redirects {
                    return Err(TransportError::TooManyRedirects);
                }
                let location = response
                    .headers()
                    .get(LOCATION)
                    .and_then(|value| value.to_str().ok())
                    .ok_or(TransportError::InvalidRedirect)?;
                let next = resolve_redirect(self.policy.as_ref(), &current, location)?;
                if matches!(response.status().as_u16(), 301..=303)
                    && method != Method::GET
                    && method != Method::HEAD
                {
                    method = Method::GET;
                    body = None;
                }
                redirects += 1;
                current = next;
                continue;
            }

            return self.read_response(response, &current).await;
        }
    }

    /// Read the body with the Go budget: `limit + 1` bytes, then decide.
    async fn read_response(
        &self,
        mut response: reqwest::Response,
        final_url: &Url,
    ) -> Result<HttpResponse, TransportError> {
        let status = response.status().as_u16();
        let content_type = response
            .headers()
            .get(CONTENT_TYPE)
            .and_then(|value| value.to_str().ok())
            .map(str::to_owned);
        let mut media = content_type
            .as_deref()
            .map(encoding::media_type)
            .unwrap_or_default();
        let limit = if !media.is_empty() && !encoding::html_media_type(&media) {
            MAX_RESOURCE_BYTES
        } else {
            self.max_response_bytes
        };

        let mut body = Vec::new();
        while let Some(chunk) = response
            .chunk()
            .await
            .map_err(|error| TransportError::Request(error.to_string()))?
        {
            append_capped(&mut body, &chunk, limit + 1);
            if body.len() > limit {
                break;
            }
        }

        if media.is_empty() {
            media = encoding::sniff_media_type(&body);
        }
        let observed = body.len();
        if !encoding::html_media_type(&media) {
            let truncated = body.len() > MAX_RESOURCE_BYTES;
            body.truncate(MAX_RESOURCE_BYTES);
            return Ok(HttpResponse {
                final_url: final_url.to_string(),
                status,
                content_type,
                media_type: media,
                body,
                wire_bytes: observed,
                truncated,
            });
        }
        if body.len() > limit {
            return Err(TransportError::ResponseTooLarge { limit });
        }
        Ok(HttpResponse {
            final_url: final_url.to_string(),
            status,
            content_type,
            media_type: media,
            body,
            wire_bytes: observed,
            truncated: false,
        })
    }
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
mod tests;
