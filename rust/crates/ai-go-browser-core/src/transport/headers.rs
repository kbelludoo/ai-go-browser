//! Request headers copied verbatim from Go `browser.request`.
//!
//! These values are part of the wire contract, not tunables: the parity
//! harness records them on the server side and compares the Rust request with
//! the Go one, so they live in one place and are not derived from a library
//! default.

pub const USER_AGENT: &str = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36";
pub const ACCEPT: &str = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7";
pub const ACCEPT_LANGUAGE: &str = "en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7";
pub const SEC_CH_UA: &str = r#""Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24""#;
pub const SEC_CH_UA_MOBILE: &str = "?0";
pub const SEC_CH_UA_PLATFORM: &str = r#""Windows""#;
pub const SEC_FETCH_DEST: &str = "document";
pub const SEC_FETCH_MODE: &str = "navigate";
pub const SEC_FETCH_SITE: &str = "none";
pub const SEC_FETCH_USER: &str = "?1";
pub const UPGRADE_INSECURE_REQUESTS: &str = "1";

pub const REQUEST_HEADERS: [(&str, &str); 11] = [
    ("user-agent", USER_AGENT),
    ("accept", ACCEPT),
    ("accept-language", ACCEPT_LANGUAGE),
    ("sec-ch-ua", SEC_CH_UA),
    ("sec-ch-ua-mobile", SEC_CH_UA_MOBILE),
    ("sec-ch-ua-platform", SEC_CH_UA_PLATFORM),
    ("sec-fetch-dest", SEC_FETCH_DEST),
    ("sec-fetch-mode", SEC_FETCH_MODE),
    ("sec-fetch-site", SEC_FETCH_SITE),
    ("sec-fetch-user", SEC_FETCH_USER),
    ("upgrade-insecure-requests", UPGRADE_INSECURE_REQUESTS),
];

/// Go sets this only for `POST`; it never advertises `Accept-Encoding`.
pub const FORM_CONTENT_TYPE: &str = "application/x-www-form-urlencoded";
