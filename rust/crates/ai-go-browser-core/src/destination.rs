//! Destination policy and DNS seams for the HTTP transport.
//!
//! [`PublicInternet`] is the production policy: it rejects local hostnames and
//! every non-public address, and the transport re-applies it to each redirect
//! hop and to every resolved address. Adapters can substitute a narrower policy
//! or a deterministic resolver, which is also how the network integration tests
//! exercise loopback fixtures without weakening the production default.

use std::net::{IpAddr, SocketAddr};

use async_trait::async_trait;
use url::Url;

use crate::url_policy::{check_public_host, is_non_public_ipv4, is_non_public_ipv6};
use crate::{TransportError, UrlPolicyError};

/// Gate applied to every request URL and every resolved destination address.
pub trait DestinationPolicy: Send + Sync {
    /// Reject a URL before DNS resolution or connection.
    fn check_url(&self, url: &Url) -> Result<(), UrlPolicyError>;

    /// Reject a resolved address before it is pinned for the connection.
    fn check_address(&self, address: IpAddr) -> Result<(), UrlPolicyError>;
}

/// Production policy: public HTTP(S) destinations only.
#[derive(Clone, Copy, Debug, Default)]
pub struct PublicInternet;

impl DestinationPolicy for PublicInternet {
    fn check_url(&self, url: &Url) -> Result<(), UrlPolicyError> {
        check_public_host(url)
    }

    fn check_address(&self, address: IpAddr) -> Result<(), UrlPolicyError> {
        let non_public = match address {
            IpAddr::V4(ip) => is_non_public_ipv4(ip),
            IpAddr::V6(ip) => is_non_public_ipv6(ip),
        };
        if non_public {
            Err(UrlPolicyError::PrivateAddress)
        } else {
            Ok(())
        }
    }
}

/// DNS seam. The transport resolves through this once per hop and pins the
/// answers with `reqwest::ClientBuilder::resolve_to_addrs`, so a second lookup
/// cannot change the destination after validation.
#[async_trait]
pub trait HostResolver: Send + Sync {
    async fn resolve(&self, host: &str, port: u16) -> Result<Vec<SocketAddr>, TransportError>;
}

/// Resolver backed by the operating system, used in production.
#[derive(Clone, Copy, Debug, Default)]
pub struct SystemResolver;

#[async_trait]
impl HostResolver for SystemResolver {
    async fn resolve(&self, host: &str, port: u16) -> Result<Vec<SocketAddr>, TransportError> {
        let found = tokio::net::lookup_host((host, port))
            .await
            .map_err(|error| TransportError::Dns(error.to_string()))?
            .collect::<Vec<_>>();
        if found.is_empty() {
            return Err(TransportError::Dns(
                "hostname resolved to no addresses".into(),
            ));
        }
        Ok(found)
    }
}
