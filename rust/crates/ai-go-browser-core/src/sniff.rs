//! Media-type sniffing ported from Go `net/http/internal.DetectContentType`.
//!
//! The transport calls this only when the response has no `Content-Type`
//! header, exactly like the Go reference. The signature table and its order are
//! copied from the standard library so the chosen type matches on the same
//! bytes instead of approximating the WHATWG algorithm.

const SNIFF_LEN: usize = 512;

enum Signature {
    Html(&'static [u8]),
    Exact(&'static [u8], &'static str),
    Masked {
        mask: &'static [u8],
        pattern: &'static [u8],
        skip_ws: bool,
        content_type: &'static str,
    },
    Mp4,
    Text,
}

impl Signature {
    fn matches(&self, data: &[u8], first_non_ws: usize) -> Option<&'static str> {
        match self {
            Signature::Html(prefix) => html_match(data, first_non_ws, prefix),
            Signature::Exact(signature, content_type) => {
                data.starts_with(signature).then_some(content_type)
            }
            Signature::Masked {
                mask,
                pattern,
                skip_ws,
                content_type,
            } => {
                let data = if *skip_ws {
                    &data[first_non_ws..]
                } else {
                    data
                };
                if mask.len() != pattern.len() || data.len() < pattern.len() {
                    return None;
                }
                let hit = pattern
                    .iter()
                    .zip(mask.iter())
                    .zip(data.iter())
                    .all(|((pattern, mask), byte)| byte & mask == *pattern);
                hit.then_some(content_type)
            }
            Signature::Mp4 => mp4_match(data),
            Signature::Text => text_match(data, first_non_ws),
        }
    }
}

const SIGNATURES: &[Signature] = &[
    Signature::Html(b"<!DOCTYPE HTML"),
    Signature::Html(b"<HTML"),
    Signature::Html(b"<HEAD"),
    Signature::Html(b"<SCRIPT"),
    Signature::Html(b"<IFRAME"),
    Signature::Html(b"<H1"),
    Signature::Html(b"<DIV"),
    Signature::Html(b"<FONT"),
    Signature::Html(b"<TABLE"),
    Signature::Html(b"<A"),
    Signature::Html(b"<STYLE"),
    Signature::Html(b"<TITLE"),
    Signature::Html(b"<B"),
    Signature::Html(b"<BODY"),
    Signature::Html(b"<BR"),
    Signature::Html(b"<P"),
    Signature::Html(b"<!--"),
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\xFF\xFF",
        pattern: b"<?xml",
        skip_ws: true,
        content_type: "text/xml; charset=utf-8",
    },
    Signature::Exact(b"%PDF-", "application/pdf"),
    Signature::Exact(b"%!PS-Adobe-", "application/postscript"),
    Signature::Masked {
        mask: b"\xFF\xFF\x00\x00",
        pattern: b"\xFE\xFF\x00\x00",
        skip_ws: false,
        content_type: "text/plain; charset=utf-16be",
    },
    Signature::Masked {
        mask: b"\xFF\xFF\x00\x00",
        pattern: b"\xFF\xFE\x00\x00",
        skip_ws: false,
        content_type: "text/plain; charset=utf-16le",
    },
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\x00",
        pattern: b"\xEF\xBB\xBF\x00",
        skip_ws: false,
        content_type: "text/plain; charset=utf-8",
    },
    Signature::Exact(b"\x00\x00\x01\x00", "image/x-icon"),
    Signature::Exact(b"\x00\x00\x02\x00", "image/x-icon"),
    Signature::Exact(b"BM", "image/bmp"),
    Signature::Exact(b"GIF87a", "image/gif"),
    Signature::Exact(b"GIF89a", "image/gif"),
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF\xFF\xFF",
        pattern: b"RIFF\x00\x00\x00\x00WEBPVP",
        skip_ws: false,
        content_type: "image/webp",
    },
    Signature::Exact(b"\x89PNG\x0D\x0A\x1A\x0A", "image/png"),
    Signature::Exact(b"\xFF\xD8\xFF", "image/jpeg"),
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF",
        pattern: b"FORM\x00\x00\x00\x00AIFF",
        skip_ws: false,
        content_type: "audio/aiff",
    },
    Signature::Masked {
        mask: b"\xFF\xFF\xFF",
        pattern: b"ID3",
        skip_ws: false,
        content_type: "audio/mpeg",
    },
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\xFF\xFF",
        pattern: b"OggS\x00",
        skip_ws: false,
        content_type: "application/ogg",
    },
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF",
        pattern: b"MThd\x00\x00\x00\x06",
        skip_ws: false,
        content_type: "audio/midi",
    },
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF",
        pattern: b"RIFF\x00\x00\x00\x00AVI ",
        skip_ws: false,
        content_type: "video/avi",
    },
    Signature::Masked {
        mask: b"\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF",
        pattern: b"RIFF\x00\x00\x00\x00WAVE",
        skip_ws: false,
        content_type: "audio/wave",
    },
    Signature::Mp4,
    Signature::Exact(b"\x1A\x45\xDF\xA3", "video/webm"),
    Signature::Masked {
        mask: b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF\xFF",
        pattern: b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00LP",
        skip_ws: false,
        content_type: "application/vnd.ms-fontobject",
    },
    Signature::Exact(b"\x00\x01\x00\x00", "font/ttf"),
    Signature::Exact(b"OTTO", "font/otf"),
    Signature::Exact(b"ttcf", "font/collection"),
    Signature::Exact(b"wOFF", "font/woff"),
    Signature::Exact(b"wOF2", "font/woff2"),
    Signature::Exact(b"\x1F\x8B\x08", "application/x-gzip"),
    Signature::Exact(b"PK\x03\x04", "application/zip"),
    Signature::Exact(b"Rar!\x1A\x07\x00", "application/x-rar-compressed"),
    Signature::Exact(b"Rar!\x1A\x07\x01\x00", "application/x-rar-compressed"),
    Signature::Exact(b"\x00\x61\x73\x6D", "application/wasm"),
    Signature::Text,
];

/// Go `http.DetectContentType`: always returns a valid MIME type.
pub fn detect_content_type(data: &[u8]) -> String {
    let data = &data[..data.len().min(SNIFF_LEN)];
    let first_non_ws = data.iter().position(|byte| !is_whitespace(*byte));
    let first_non_ws = first_non_ws.unwrap_or(data.len());
    for signature in SIGNATURES {
        if let Some(content_type) = signature.matches(data, first_non_ws) {
            return content_type.to_owned();
        }
    }
    "application/octet-stream".to_owned()
}

fn is_whitespace(byte: u8) -> bool {
    matches!(byte, b'\t' | b'\n' | 0x0c | b'\r' | b' ')
}

fn is_tag_terminating(byte: u8) -> bool {
    matches!(byte, b' ' | b'>')
}

fn html_match(data: &[u8], first_non_ws: usize, prefix: &[u8]) -> Option<&'static str> {
    let data = &data[first_non_ws..];
    if data.len() < prefix.len() + 1 {
        return None;
    }
    let matched = prefix.iter().zip(data.iter()).all(|(expected, actual)| {
        let expected = expected.to_ascii_uppercase();
        let actual = if actual.is_ascii_lowercase() {
            actual.to_ascii_uppercase()
        } else {
            *actual
        };
        expected == actual
    });
    (matched && is_tag_terminating(data[prefix.len()])).then_some("text/html; charset=utf-8")
}

fn mp4_match(data: &[u8]) -> Option<&'static str> {
    if data.len() < 12 {
        return None;
    }
    let box_size = u32::from_be_bytes([data[0], data[1], data[2], data[3]]) as usize;
    if data.len() < box_size || !box_size.is_multiple_of(4) {
        return None;
    }
    if &data[4..8] != b"ftyp" {
        return None;
    }
    let mut start = 8;
    while start < box_size {
        if start == 12 {
            start += 4;
            continue;
        }
        if start + 3 <= box_size && &data[start..start + 3] == b"mp4" {
            return Some("video/mp4");
        }
        start += 4;
    }
    None
}

fn text_match(data: &[u8], first_non_ws: usize) -> Option<&'static str> {
    for byte in &data[first_non_ws..] {
        let binary = matches!(
            *byte,
            0x00..=0x08 | 0x0b | 0x0e..=0x1a | 0x1c..=0x1f
        );
        if binary {
            return None;
        }
    }
    Some("text/plain; charset=utf-8")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_html_after_leading_whitespace_and_any_case() {
        assert_eq!(
            detect_content_type(b"\n\t  <!doctype html><title>x</title>"),
            "text/html; charset=utf-8"
        );
        assert_eq!(
            detect_content_type(b"<title>x</title>"),
            "text/html; charset=utf-8"
        );
    }

    #[test]
    fn requires_a_tag_terminator_after_the_prefix() {
        // `<p>` is itself an HTML signature in Go, so the terminator check is
        // about the byte after the two-character prefix.
        assert_eq!(
            detect_content_type(b"<p>plain</p>"),
            "text/html; charset=utf-8"
        );
        assert_eq!(detect_content_type(b"<p plain"), "text/html; charset=utf-8");
        assert_eq!(
            detect_content_type(b"plain <p>"),
            "text/plain; charset=utf-8"
        );
    }

    #[test]
    fn detects_binary_signatures_before_falling_back_to_text() {
        assert_eq!(detect_content_type(b"\x89PNG\r\n\x1a\n\0\0"), "image/png");
        assert_eq!(detect_content_type(b"PK\x03\x04rest"), "application/zip");
        assert_eq!(
            detect_content_type(b"\x00\x01\x02"),
            "application/octet-stream"
        );
        assert_eq!(detect_content_type(b"%PDF-1.7"), "application/pdf");
    }

    #[test]
    fn empty_input_is_text_like_the_reference() {
        // Go's textSig accepts an empty payload, so DetectContentType never
        // reaches the application/octet-stream fallback for empty input.
        assert_eq!(detect_content_type(b""), "text/plain; charset=utf-8");
    }

    #[test]
    fn only_the_first_512_bytes_are_considered() {
        let mut data = vec![b'a'; 512];
        data.extend_from_slice(b"<html>");
        assert_eq!(detect_content_type(&data), "text/plain; charset=utf-8");
    }
}
