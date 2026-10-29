use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

use url::Url;

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum UrlPolicyError {
    MissingScheme,
    UnsupportedScheme,
    MissingHost,
    Credentials,
    LocalHost,
    PrivateAddress,
}

/// Validate URL syntax and obvious non-public destinations.
///
/// This is only the first layer of SSRF protection. A network adapter must
/// still resolve DNS, reject non-public answers, connect to a validated/pinned
/// address, and revalidate every redirect target.
pub fn validate_url(input: &str) -> Result<(), UrlPolicyError> {
    if !input.contains(':') {
        return Err(UrlPolicyError::MissingScheme);
    }

    let parsed = Url::parse(input).map_err(|_| {
        if input.split_once("://").is_none() {
            UrlPolicyError::MissingScheme
        } else {
            UrlPolicyError::MissingHost
        }
    })?;

    if !matches!(parsed.scheme(), "http" | "https") {
        return Err(UrlPolicyError::UnsupportedScheme);
    }
    if parsed.username() != "" || parsed.password().is_some() {
        return Err(UrlPolicyError::Credentials);
    }

    let host = parsed.host().ok_or(UrlPolicyError::MissingHost)?;
    match host {
        url::Host::Domain(domain) => {
            let normalized = domain.trim_end_matches('.').to_ascii_lowercase();
            if normalized == "localhost"
                || normalized.ends_with(".localhost")
                || normalized == "localhost.localdomain"
                || normalized.ends_with(".local")
            {
                return Err(UrlPolicyError::LocalHost);
            }
        }
        url::Host::Ipv4(address) => {
            if is_non_public_ipv4(address) {
                return Err(UrlPolicyError::PrivateAddress);
            }
        }
        url::Host::Ipv6(address) => {
            if is_non_public_ipv6(address) {
                return Err(UrlPolicyError::PrivateAddress);
            }
        }
    }

    Ok(())
}

fn is_non_public_ipv4(ip: Ipv4Addr) -> bool {
    let [a, b, _, _] = ip.octets();
    ip.is_private()
        || ip.is_loopback()
        || ip.is_link_local()
        || ip.is_unspecified()
        || ip.is_broadcast()
        || ip.is_multicast()
        // Shared address space (RFC 6598), benchmarking, documentation,
        // protocol assignments, and other special-use ranges.
        || (a == 100 && (64..=127).contains(&b))
        || (a == 198 && (b == 18 || b == 19))
        || (a == 192 && b == 0)
        || (a == 192 && b == 0 && ip.octets()[2] == 2)
        || (a == 192 && b == 0 && ip.octets()[2] == 0)
        || (a == 198 && b == 51 && ip.octets()[2] == 100)
        || (a == 203 && b == 0 && ip.octets()[2] == 113)
        || a >= 240
}

fn is_non_public_ipv6(ip: Ipv6Addr) -> bool {
    let octets = ip.octets();
    let is_ipv4_mapped = octets[..10].iter().all(|byte| *byte == 0)
        && octets[10] == 0xff
        && octets[11] == 0xff;

    ip.is_loopback()
        || ip.is_unspecified()
        || ip.is_unique_local()
        || ip.is_unicast_link_local()
        || ip.is_multicast()
        || is_ipv4_mapped
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_public_http() {
        assert_eq!(validate_url("https://example.com/path"), Ok(()));
    }

    #[test]
    fn rejects_file_scheme() {
        assert_eq!(
            validate_url("file:///etc/passwd"),
            Err(UrlPolicyError::UnsupportedScheme)
        );
    }

    #[test]
    fn rejects_missing_scheme() {
        assert_eq!(validate_url("example.com"), Err(UrlPolicyError::MissingScheme));
    }

    #[test]
    fn rejects_localhost_variants() {
        for url in [
            "http://localhost:8080",
            "http://sub.localhost/",
            "http://localhost.localdomain/",
            "http://printer.local/",
        ] {
            assert_eq!(validate_url(url), Err(UrlPolicyError::LocalHost), "{url}");
        }
    }

    #[test]
    fn rejects_loopback_and_private_ips() {
        for url in [
            "http://127.0.0.1",
            "http://10.1.2.3",
            "http://169.254.1.1",
            "http://100.64.0.1",
            "http://192.0.2.1",
            "http://198.18.0.1",
            "http://224.0.0.1",
        ] {
            assert_eq!(validate_url(url), Err(UrlPolicyError::PrivateAddress), "{url}");
        }
    }

    #[test]
    fn rejects_ipv6_local_and_mapped_addresses() {
        for url in [
            "http://[::1]/",
            "http://[fc00::1]/",
            "http://[fe80::1]/",
            "http://[::ffff:127.0.0.1]/",
        ] {
            assert_eq!(validate_url(url), Err(UrlPolicyError::PrivateAddress), "{url}");
        }
    }

    #[test]
    fn rejects_userinfo() {
        assert_eq!(
            validate_url("https://user:pass@example.com"),
            Err(UrlPolicyError::Credentials)
        );
    }

    #[test]
    fn rejects_non_http_schemes() {
        assert_eq!(
            validate_url("ftp://example.com"),
            Err(UrlPolicyError::UnsupportedScheme)
        );
    }
}
