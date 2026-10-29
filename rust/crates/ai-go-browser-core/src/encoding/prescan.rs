//! Go `charset.prescan`: the `<meta>` declaration scan used for HTML bodies.
//!
//! The rules follow the HTML encoding sniffing algorithm: the first `<meta>`
//! that both declares a usable label and satisfies the pragma requirement wins.
//! Comments and raw-text elements are skipped so a `<meta>` inside a script
//! string is not mistaken for a declaration.

use encoding_rs::{Encoding, UTF_16BE, UTF_16LE, UTF_8};

#[derive(Clone, Copy, Eq, PartialEq)]
enum Pragma {
    Unknown,
    Required,
    NotRequired,
}

/// Return the encoding declared by the first usable `<meta>` element.
pub(super) fn prescan(content: &[u8]) -> Option<&'static Encoding> {
    let mut index = 0;
    while let Some(after_name) = next_start_tag(content, &mut index, b"meta") {
        let (attributes, next) = parse_attributes(content, after_name);
        index = next;

        let mut encoding = None;
        let mut found_pragma = false;
        let mut need_pragma = Pragma::Unknown;
        let mut seen = Vec::new();
        for (name, value) in attributes {
            if seen.contains(&name) {
                continue;
            }
            seen.push(name.clone());
            match name.as_str() {
                "http-equiv" => {
                    if value == "content-type" {
                        found_pragma = true;
                    }
                }
                "content" => {
                    if encoding.is_none()
                        && let Some(label) = meta_element_charset(&value)
                        && let Some(found) = Encoding::for_label(label.as_bytes())
                    {
                        encoding = Some(found);
                        need_pragma = Pragma::Required;
                    }
                }
                "charset" => {
                    encoding = Encoding::for_label(value.as_bytes());
                    need_pragma = Pragma::NotRequired;
                }
                _ => {}
            }
        }

        if need_pragma == Pragma::Unknown || (need_pragma == Pragma::Required && !found_pragma) {
            continue;
        }
        let Some(encoding) = encoding else {
            continue;
        };
        // A UTF-16 label in a meta tag is treated as UTF-8 by the HTML spec.
        if encoding == UTF_16LE || encoding == UTF_16BE {
            return Some(UTF_8);
        }
        return Some(encoding);
    }
    None
}

/// Go `fromMetaElement`: extract `charset` from a `content` attribute.
fn meta_element_charset(value: &str) -> Option<String> {
    let mut rest = value;
    loop {
        let location = rest.find("charset")?;
        rest = rest[location + "charset".len()..].trim_start();
        let Some(after_equals) = rest.strip_prefix('=') else {
            continue;
        };
        rest = after_equals.trim_start();
        if rest.is_empty() {
            return None;
        }
        let quote = rest.chars().next().unwrap_or('\0');
        if quote == '"' || quote == '\'' {
            let body = &rest[1..];
            let end = body.find(quote)?;
            return Some(body[..end].to_owned());
        }
        let end = rest
            .find(|character: char| character == ';' || character.is_ascii_whitespace())
            .unwrap_or(rest.len());
        return Some(rest[..end].to_owned());
    }
}

/// Advance past the next start tag with the requested name and return the
/// offset just after that name.
fn next_start_tag(content: &[u8], index: &mut usize, wanted: &[u8]) -> Option<usize> {
    while *index < content.len() {
        let open = content[*index..].iter().position(|byte| *byte == b'<')? + *index;
        *index = open;
        if content[open..].starts_with(b"<!--") {
            *index = content[open + 4..]
                .windows(3)
                .position(|window| window == b"-->")
                .map(|offset| open + 4 + offset + 3)
                .unwrap_or(content.len());
            continue;
        }
        let name_start = open + 1;
        if name_start >= content.len() {
            return None;
        }
        let mut name_end = name_start;
        while name_end < content.len() && is_tag_name_byte(content[name_end]) {
            name_end += 1;
        }
        if name_end == name_start {
            *index = name_start;
            continue;
        }
        let name = &content[name_start..name_end];
        *index = name_end;
        if name.eq_ignore_ascii_case(wanted) {
            return Some(name_end);
        }
        for raw in [b"script".as_slice(), b"style".as_slice()] {
            if name.eq_ignore_ascii_case(raw) {
                *index = skip_raw_text(content, name_end, raw);
            }
        }
    }
    None
}

fn skip_raw_text(content: &[u8], from: usize, name: &[u8]) -> usize {
    let mut closing = Vec::with_capacity(name.len() + 2);
    closing.extend_from_slice(b"</");
    closing.extend_from_slice(name);
    content[from..]
        .windows(closing.len())
        .position(|window| window.eq_ignore_ascii_case(&closing))
        .map(|offset| from + offset + closing.len())
        .unwrap_or(content.len())
}

fn is_tag_name_byte(byte: u8) -> bool {
    byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_' | b':')
}

fn parse_attributes(content: &[u8], from: usize) -> (Vec<(String, String)>, usize) {
    let mut attributes = Vec::new();
    let mut index = from;
    while index < content.len() {
        while index < content.len() && content[index].is_ascii_whitespace() {
            index += 1;
        }
        if index >= content.len() || content[index] == b'>' {
            return (attributes, index.saturating_add(1));
        }
        if content[index] == b'/' {
            index += 1;
            continue;
        }
        let name_start = index;
        while index < content.len()
            && !content[index].is_ascii_whitespace()
            && !matches!(content[index], b'=' | b'>' | b'/')
        {
            index += 1;
        }
        if index == name_start {
            index += 1;
            continue;
        }
        let name = String::from_utf8_lossy(&content[name_start..index]).to_ascii_lowercase();
        while index < content.len() && content[index].is_ascii_whitespace() {
            index += 1;
        }
        if index >= content.len() || content[index] != b'=' {
            attributes.push((name, String::new()));
            continue;
        }
        index += 1;
        while index < content.len() && content[index].is_ascii_whitespace() {
            index += 1;
        }
        if index < content.len() && matches!(content[index], b'"' | b'\'') {
            let quote = content[index];
            index += 1;
            let value_start = index;
            while index < content.len() && content[index] != quote {
                index += 1;
            }
            let value = String::from_utf8_lossy(&content[value_start..index]).to_ascii_lowercase();
            attributes.push((name, value));
            index += 1;
            continue;
        }
        let value_start = index;
        while index < content.len()
            && !content[index].is_ascii_whitespace()
            && content[index] != b'>'
        {
            index += 1;
        }
        let value = String::from_utf8_lossy(&content[value_start..index]).to_ascii_lowercase();
        attributes.push((name, value));
    }
    (attributes, index)
}
