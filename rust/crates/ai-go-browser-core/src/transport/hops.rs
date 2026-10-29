//! Hop mechanics shared by the request loop and the tests: redirect
//! resolution, redirect detection and the streamed body cap.

use reqwest::StatusCode;
use url::{Url, form_urlencoded};

use crate::DecodeError;
use crate::destination::DestinationPolicy;
use crate::url_policy::validate_syntax;

use super::TransportError;

/// Resolve and validate a redirect destination before any subsequent DNS
/// lookup or request.
pub(super) fn resolve_redirect(
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

/// Append one streamed chunk up to a hard cap, keeping any prefix that fits.
pub(super) fn append_capped(body: &mut Vec<u8>, chunk: &[u8], limit: usize) {
    let room = limit.saturating_sub(body.len());
    body.extend_from_slice(&chunk[..chunk.len().min(room)]);
}

/// Go `url.Values.Encode`: sorted keys, values in insertion order.
pub fn encode_form(fields: &[(String, String)]) -> String {
    let mut grouped: std::collections::BTreeMap<&str, Vec<&str>> = std::collections::BTreeMap::new();
    for (name, value) in fields {
        grouped.entry(name).or_default().push(value);
    }
    let mut serializer = form_urlencoded::Serializer::new(String::new());
    for (name, values) in grouped {
        for value in values {
            serializer.append_pair(name, value);
        }
    }
    serializer.finish()
}

/// The redirect statuses the request loop follows.
pub(super) fn is_redirect(status: StatusCode) -> bool {
    matches!(status.as_u16(), 301 | 302 | 303 | 307 | 308)
}

/// Map the charset layer's errors onto the transport error surface.
pub(super) fn decode_error(error: DecodeError) -> TransportError {
    match error {
        DecodeError::UnsupportedCharset(label) => TransportError::InvalidCharset(label),
        DecodeError::EncodedTooLarge { limit } => TransportError::DecodeLimit {
            limit,
            decoded: false,
        },
        DecodeError::DecodedTooLarge { limit } => TransportError::DecodeLimit {
            limit,
            decoded: true,
        },
    }
}
