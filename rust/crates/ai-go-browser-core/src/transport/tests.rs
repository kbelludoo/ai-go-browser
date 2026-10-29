use super::*;
use reqwest::StatusCode;

#[tokio::test]
async fn rejects_private_literal_before_opening_a_connection() {
    for url in [
        "http://127.0.0.1:1/",
        "http://10.0.0.1:1/",
        "http://169.254.169.254/latest/meta-data/",
        "http://[::1]:1/",
    ] {
        let error = SafeHttpTransport::default().get(url).await.unwrap_err();
        assert_eq!(
            error,
            TransportError::Policy(UrlPolicyError::PrivateAddress),
            "{url}"
        );
    }
}

#[tokio::test]
async fn rejects_localhost_before_dns_resolution() {
    let error = SafeHttpTransport::default()
        .get("http://localhost:1/")
        .await
        .unwrap_err();
    assert_eq!(error, TransportError::Policy(UrlPolicyError::LocalHost));
}

#[tokio::test]
async fn rejects_non_http_schemes_before_network_access() {
    let error = SafeHttpTransport::default()
        .get("file:///etc/passwd")
        .await
        .unwrap_err();
    assert_eq!(
        error,
        TransportError::Policy(UrlPolicyError::UnsupportedScheme)
    );
}

#[test]
fn parses_response_body_into_semantic_snapshot() {
    let response = HttpResponse::new(
        "https://example.com/page",
        200,
        Some("text/html; charset=utf-8".into()),
        b"<title>Demo</title><button>Continue</button>".to_vec(),
    );
    let page = FetchedPage::from_response(response).unwrap();
    assert_eq!(page.snapshot.url, "https://example.com/page");
    assert_eq!(page.snapshot.title, "Demo");
    assert_eq!(page.snapshot.actionable().count(), 1);
}

#[test]
fn decodes_declared_legacy_charsets_before_parsing() {
    let response = HttpResponse::new(
        "https://example.com/",
        200,
        Some("text/html; charset=iso-8859-1".into()),
        b"<title>Caf\xe9</title>".to_vec(),
    );
    let page = FetchedPage::from_response(response).unwrap();
    assert_eq!(page.decoded.encoding, "windows-1252");
    assert_eq!(page.snapshot.title, "Café");
}

#[test]
fn honors_a_utf16_bom_instead_of_rejecting_the_body() {
    let mut body = vec![0xFF, 0xFE];
    for unit in "<title>Wide</title>".encode_utf16() {
        body.extend_from_slice(&unit.to_le_bytes());
    }
    let response = HttpResponse::new("https://example.com/", 200, None, body);
    let page = FetchedPage::from_response(response).unwrap();
    assert_eq!(page.decoded.encoding, "utf-16le");
    assert_eq!(page.snapshot.title, "Wide");
}

#[test]
fn redirect_resolution_rejects_private_destinations_before_fetching() {
    let current = Url::parse("https://example.com/start").unwrap();
    for location in [
        "http://127.0.0.1/admin",
        "http://localhost/admin",
        "file:///etc/passwd",
    ] {
        let error = resolve_redirect(&PublicInternet, &current, location).unwrap_err();
        assert!(
            matches!(error, TransportError::Policy(_)),
            "expected policy rejection for {location}, got {error:?}"
        );
    }
}

#[test]
fn redirect_resolution_supports_relative_locations() {
    let current = Url::parse("https://example.com/a/page").unwrap();
    let target = resolve_redirect(&PublicInternet, &current, "../next").unwrap();
    assert_eq!(target.as_str(), "https://example.com/next");
}

#[test]
fn streamed_body_cap_keeps_the_prefix_that_fits() {
    let mut body = Vec::new();
    append_capped(&mut body, b"abc", 5);
    append_capped(&mut body, b"def", 5);
    assert_eq!(body, b"abcde");
}

#[test]
fn streamed_body_cap_does_not_append_past_the_limit() {
    let mut body = b"abc".to_vec();
    append_capped(&mut body, b"def", 5);
    assert_eq!(body, b"abcde");
}

#[test]
fn default_limits_match_the_reference() {
    let transport = SafeHttpTransport::default();
    assert_eq!(transport.timeout, Duration::from_secs(20));
    assert_eq!(transport.max_response_bytes, 5 << 20);
    assert_eq!(transport.max_redirects, 15);
}

#[test]
fn recognizes_only_supported_redirect_statuses() {
    assert!(is_redirect(StatusCode::MOVED_PERMANENTLY));
    assert!(is_redirect(StatusCode::FOUND));
    assert!(is_redirect(StatusCode::TEMPORARY_REDIRECT));
    assert!(!is_redirect(StatusCode::OK));
    assert!(!is_redirect(StatusCode::BAD_REQUEST));
}

#[test]
fn form_encoding_sorts_keys_and_keeps_repeated_values() {
    let fields = vec![
        ("b".to_owned(), "two words".to_owned()),
        ("a".to_owned(), "café".to_owned()),
        ("b".to_owned(), "second".to_owned()),
        ("empty".to_owned(), String::new()),
    ];
    assert_eq!(
        encode_form(&fields),
        "a=caf%C3%A9&b=two+words&b=second&empty="
    );
}

#[test]
fn request_headers_are_the_go_set() {
    assert_eq!(headers::REQUEST_HEADERS.len(), 11);
    let names = headers::REQUEST_HEADERS.map(|(name, _)| name);
    for expected in [
        "user-agent",
        "accept",
        "accept-language",
        "sec-ch-ua",
        "sec-fetch-dest",
        "sec-fetch-mode",
        "sec-fetch-site",
        "sec-fetch-user",
        "upgrade-insecure-requests",
    ] {
        assert!(names.contains(&expected), "{expected} missing");
    }
    assert!(
        !names.contains(&"accept-encoding"),
        "the Go RoundTripper never advertises gzip"
    );
}
