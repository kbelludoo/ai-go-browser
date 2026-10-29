use std::net::IpAddr;

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum UrlPolicyError { MissingScheme, UnsupportedScheme, MissingHost, Credentials, LocalHost, PrivateAddress }

/// Syntactic URL gate only. DNS resolution, address pinning, and redirect
/// revalidation must be implemented by the transport adapter.
pub fn validate_url(input: &str) -> Result<(), UrlPolicyError> {
    let (scheme, rest) = input.split_once("://").ok_or(UrlPolicyError::MissingScheme)?;
    if !matches!(scheme.to_ascii_lowercase().as_str(), "http" | "https") {
        return Err(UrlPolicyError::UnsupportedScheme);
    }
    let authority = rest.split(['/', '?', '#']).next().unwrap_or("");
    if authority.is_empty() { return Err(UrlPolicyError::MissingHost); }
    if authority.contains('@') { return Err(UrlPolicyError::Credentials); }
    let host_port = authority.rsplit_once(':').filter(|(_, p)| p.parse::<u16>().is_ok())
        .map(|(h, _)| h).unwrap_or(authority);
    let host = host_port.trim_matches(['[', ']']).trim_end_matches('.');
    let normalized = host.to_ascii_lowercase();
    if normalized == "localhost" || normalized.ends_with(".localhost") || normalized == "localhost.localdomain" {
        return Err(UrlPolicyError::LocalHost);
    }
    if let Ok(ip) = host.parse::<IpAddr>() {
        let private = match ip {
            IpAddr::V4(v) => v.is_private() || v.is_loopback() || v.is_link_local() || v.is_unspecified() || v.is_broadcast(),
            IpAddr::V6(v) => v.is_loopback() || v.is_unspecified() || v.is_unique_local() || v.is_unicast_link_local(),
        };
        if private { return Err(UrlPolicyError::PrivateAddress); }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
 use super::*;
 #[test] fn accepts_public_http() { assert_eq!(validate_url("https://example.com/path"), Ok(())); }
 #[test] fn rejects_file_scheme() { assert_eq!(validate_url("file:///etc/passwd"), Err(UrlPolicyError::UnsupportedScheme)); }
 #[test] fn rejects_localhost() { assert_eq!(validate_url("http://localhost:8080"), Err(UrlPolicyError::LocalHost)); }
 #[test] fn rejects_loopback_ip() { assert_eq!(validate_url("http://127.0.0.1"), Err(UrlPolicyError::PrivateAddress)); }
 #[test] fn rejects_userinfo() { assert_eq!(validate_url("https://user:pass@example.com"), Err(UrlPolicyError::Credentials)); }
}
