//! Charset handling ported from Go `browser/encoding.go`.
//!
//! The precedence is BOM, then the `Content-Type` charset parameter, then -
//! for HTML only - the `<meta>` prescan and the UTF-8 heuristic, finally
//! Windows-1252. Non-HTML payloads without an explicit charset are UTF-8, and
//! an unknown label on a non-HTML payload is an error.

use encoding_rs::{Encoding, UTF_8, UTF_16BE, UTF_16LE, WINDOWS_1252};

use crate::sniff::detect_content_type;

mod prescan;

use prescan::prescan;

/// A response body converted to UTF-8, with the encoding chosen by the
/// reference rules and the number of leading BOM bytes that were removed.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DecodedBody {
    pub text: String,
    pub encoding: String,
    pub bom_bytes: usize,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum DecodeError {
    EncodedTooLarge { limit: usize },
    DecodedTooLarge { limit: usize },
    UnsupportedCharset(String),
}

/// Go `responseMediaType`.
pub fn media_type(header: &str) -> String {
    header
        .split(';')
        .next()
        .unwrap_or("")
        .trim()
        .to_ascii_lowercase()
}

/// Go `htmlMediaType`.
pub fn html_media_type(media: &str) -> bool {
    media == "text/html" || media == "application/xhtml+xml"
}

/// Go `sniffResponseMediaType`: decode a preview, then detect its type.
pub fn sniff_media_type(content: &[u8]) -> String {
    let preview = &content[..content.len().min(1024)];
    let decoded = match decode("application/octet-stream", preview, 4096) {
        Ok(decoded) => decoded.text.into_bytes(),
        Err(_) => preview.to_vec(),
    };
    media_type(&detect_content_type(&decoded))
}

/// Go `DecodeHTTPBodyLimit`.
pub fn decode(
    content_type: &str,
    raw: &[u8],
    max_bytes: usize,
) -> Result<DecodedBody, DecodeError> {
    if raw.len() > max_bytes {
        return Err(DecodeError::EncodedTooLarge { limit: max_bytes });
    }
    let (encoding, name, bom_bytes) = body_encoding(content_type, raw)?;
    let payload = &raw[bom_bytes..];
    if name == "utf-8"
        && let Ok(text) = std::str::from_utf8(payload)
    {
        return Ok(DecodedBody {
            text: text.to_owned(),
            encoding: name,
            bom_bytes,
        });
    }
    // Undeclared ASCII HTML defaults to Windows-1252, but every byte already
    // represents the same UTF-8 text.
    if name == "windows-1252" && payload.is_ascii() {
        return Ok(DecodedBody {
            text: String::from_utf8(payload.to_vec()).unwrap_or_default(),
            encoding: name,
            bom_bytes,
        });
    }
    let (text, _) = encoding.decode_without_bom_handling(payload);
    if text.len() > max_bytes {
        return Err(DecodeError::DecodedTooLarge { limit: max_bytes });
    }
    Ok(DecodedBody {
        text: text.into_owned(),
        encoding: name,
        bom_bytes,
    })
}

fn body_encoding(
    content_type: &str,
    raw: &[u8],
) -> Result<(&'static Encoding, String, usize), DecodeError> {
    for (bom, encoding) in [
        (b"\xEF\xBB\xBF".as_slice(), UTF_8),
        (b"\xFF\xFE".as_slice(), UTF_16LE),
        (b"\xFE\xFF".as_slice(), UTF_16BE),
    ] {
        if raw.starts_with(bom) {
            return Ok((encoding, canonical_name(encoding), bom.len()));
        }
    }

    let mut media = media_type(content_type);
    if media.is_empty() {
        media = media_type(&detect_content_type(raw));
    }
    let html = html_media_type(&media);
    if let Some(label) = charset_parameter(content_type) {
        if let Some(encoding) = Encoding::for_label(label.as_bytes()) {
            return Ok((encoding, canonical_name(encoding), 0));
        }
        if !html {
            return Err(DecodeError::UnsupportedCharset(label));
        }
    }
    if html {
        let encoding = determine_encoding(raw, content_type);
        return Ok((encoding, canonical_name(encoding), 0));
    }
    Ok((UTF_8, canonical_name(UTF_8), 0))
}

fn canonical_name(encoding: &'static Encoding) -> String {
    encoding.name().to_ascii_lowercase()
}

/// Go `charset.DetermineEncoding`.
fn determine_encoding(raw: &[u8], content_type: &str) -> &'static Encoding {
    let head = &raw[..raw.len().min(1024)];
    if let Some((encoding, _)) = Encoding::for_bom(head) {
        return encoding;
    }
    if let Some(label) = charset_parameter(content_type)
        && let Some(encoding) = Encoding::for_label(label.as_bytes())
    {
        return encoding;
    }
    if let Some(encoding) = prescan(head) {
        return encoding;
    }
    // The HTML spec tries UTF-8 before the legacy fallback. A partial rune at
    // the end of the preview is dropped first so it cannot veto the guess.
    let trimmed = trim_partial_rune(head);
    if trimmed.iter().any(|byte| *byte >= 0x80) && std::str::from_utf8(trimmed).is_ok() {
        return UTF_8;
    }
    WINDOWS_1252
}

fn trim_partial_rune(head: &[u8]) -> &[u8] {
    for index in (head.len().saturating_sub(3)..head.len()).rev() {
        let byte = head[index];
        if byte < 0x80 {
            break;
        }
        if is_rune_start(byte) {
            return &head[..index];
        }
    }
    head
}

fn is_rune_start(byte: u8) -> bool {
    byte & 0xC0 != 0x80
}

fn charset_parameter(content_type: &str) -> Option<String> {
    for part in content_type.split(';').skip(1) {
        let Some((name, value)) = part.split_once('=') else {
            continue;
        };
        if name.trim().eq_ignore_ascii_case("charset") {
            let value = value.trim().trim_matches('"').trim();
            if !value.is_empty() {
                return Some(value.to_owned());
            }
        }
    }
    None
}


#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn decodes_a_declared_latin1_body_as_windows_1252() {
        let raw = b"<title>Caf\xe9</title>";
        let decoded = decode("text/html; charset=iso-8859-1", raw, 1024).unwrap();
        assert_eq!(decoded.encoding, "windows-1252");
        assert_eq!(decoded.text, "<title>Café</title>");
    }

    #[test]
    fn meta_charset_wins_over_the_legacy_default() {
        let raw = b"<meta charset=\"windows-1252\"><title>Caf\xe9</title>";
        let decoded = decode("text/html", raw, 1024).unwrap();
        assert_eq!(decoded.encoding, "windows-1252");
        assert_eq!(
            decoded.text,
            "<meta charset=\"windows-1252\"><title>Café</title>"
        );
    }

    #[test]
    fn meta_http_equiv_requires_the_pragma() {
        // UTF-8 bytes read as Windows-1252 would turn "é" into "Ã©".
        let raw = "<meta content=\"text/html; charset=iso-8859-1\"><p>café</p>".as_bytes();
        let without = decode("text/html", raw, 1024).unwrap();
        assert_eq!(without.encoding, "utf-8");
        assert!(without.text.contains("café"));

        let raw = "<meta http-equiv=\"Content-Type\" content=\"text/html; charset=iso-8859-1\"><p>café</p>".as_bytes();
        let with = decode("text/html", raw, 1024).unwrap();
        assert_eq!(with.encoding, "windows-1252");
        assert!(with.text.contains("cafÃ©"));
    }

    #[test]
    fn strips_boms_and_uses_their_encoding() {
        let utf8 = decode("text/html", b"\xEF\xBB\xBF<title>ok</title>", 1024).unwrap();
        assert_eq!(utf8.bom_bytes, 3);
        assert_eq!(utf8.text, "<title>ok</title>");

        let mut raw = vec![0xFF, 0xFE];
        for unit in "<title>ok</title>".encode_utf16() {
            raw.extend_from_slice(&unit.to_le_bytes());
        }
        let decoded = decode("text/html", &raw, 1024).unwrap();
        assert_eq!(decoded.encoding, "utf-16le");
        assert_eq!(decoded.text, "<title>ok</title>");
    }

    #[test]
    fn undeclared_valid_utf8_is_not_treated_as_windows_1252() {
        let raw = "<p>crème brûlée</p>".as_bytes();
        let decoded = decode("text/html", raw, 1024).unwrap();
        assert_eq!(decoded.encoding, "utf-8");
        assert_eq!(decoded.text, "<p>crème brûlée</p>");
    }

    #[test]
    fn undeclared_ascii_defaults_to_windows_1252() {
        let decoded = decode("text/html", b"<p>plain</p>", 1024).unwrap();
        assert_eq!(decoded.encoding, "windows-1252");
        assert_eq!(decoded.text, "<p>plain</p>");
    }

    #[test]
    fn unknown_charset_is_an_error_only_for_non_html() {
        let plain = decode("text/plain; charset=made-up", b"body", 1024).unwrap_err();
        assert_eq!(plain, DecodeError::UnsupportedCharset("made-up".into()));
        let html = decode("text/html; charset=made-up", b"<p>ok</p>", 1024).unwrap();
        assert_eq!(html.encoding, "windows-1252");
    }

    #[test]
    fn non_html_without_charset_is_utf8() {
        let decoded = decode("application/json", "{\"a\":\"é\"}".as_bytes(), 1024).unwrap();
        assert_eq!(decoded.encoding, "utf-8");
        assert_eq!(decoded.text, "{\"a\":\"é\"}");
    }

    #[test]
    fn encoded_and_decoded_limits_are_enforced() {
        assert_eq!(
            decode("text/html", b"abcdef", 5).unwrap_err(),
            DecodeError::EncodedTooLarge { limit: 5 }
        );
        let raw = b"\xe9\xe9\xe9";
        assert_eq!(
            decode("text/html; charset=iso-8859-1", raw, 5).unwrap_err(),
            DecodeError::DecodedTooLarge { limit: 5 }
        );
    }

    #[test]
    fn media_type_and_sniffing_follow_the_reference() {
        assert_eq!(media_type("Text/HTML; Charset=UTF-8"), "text/html");
        assert!(html_media_type("application/xhtml+xml"));
        assert!(!html_media_type("application/xml"));
        assert_eq!(
            sniff_media_type(b"<!doctype html><title>x</title>"),
            "text/html"
        );
        assert_eq!(sniff_media_type(b"plain text"), "text/plain");
    }

    #[test]
    fn meta_inside_a_comment_or_script_is_ignored() {
        let raw = b"<!-- <meta charset=\"utf-16\"> --><meta charset=\"utf-8\"><p>ok</p>";
        let decoded = decode("text/html", raw, 1024).unwrap();
        assert_eq!(decoded.encoding, "utf-8");
    }
}
