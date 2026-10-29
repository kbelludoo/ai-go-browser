//! Golden types and the comparison rules applied to each scenario.

use std::collections::BTreeMap;

use serde::Deserialize;

#[derive(Deserialize)]
pub struct Golden {
    pub scenarios: Vec<Scenario>,
}

#[derive(Deserialize)]
pub struct Scenario {
    pub name: String,
    #[serde(default)]
    pub steps: Vec<Step>,
    pub requests: Vec<RequestRecord>,
    pub outcome: Expected,
}

#[derive(Deserialize)]
pub struct Step {
    pub cmd: String,
    #[serde(default)]
    pub url: String,
    #[serde(rename = "ref", default)]
    reference: String,
}

impl Step {
    pub fn reference(&self) -> &str {
        if self.reference.is_empty() {
            &self.url
        } else {
            &self.reference
        }
    }
}

#[derive(Deserialize, Eq, PartialEq)]
pub struct RequestRecord {
    pub method: String,
    pub path: String,
    pub query: String,
    pub headers: BTreeMap<String, String>,
    #[serde(default)]
    pub body: String,
}

#[derive(Default, Deserialize)]
pub struct Expected {
    ok: bool,
    #[serde(default)]
    error: String,
    #[serde(default)]
    url: String,
    #[serde(default)]
    status: u16,
    #[serde(default)]
    title: String,
    #[serde(default)]
    text: String,
    #[serde(default)]
    text_sha256: String,
    #[serde(default)]
    text_len: usize,
    #[serde(default)]
    bytes: usize,
    #[serde(default)]
    truncated: bool,
    #[serde(default)]
    links: usize,
    #[serde(default)]
    buttons: usize,
    #[serde(default)]
    inputs: usize,
    #[serde(default)]
    forms: usize,
}

/// What the Rust transport produced for one scenario.
#[derive(Default)]
pub struct Observed {
    pub ok: bool,
    pub error: String,
    pub url: String,
    pub status: u16,
    pub title: String,
    pub text: String,
    pub text_sha256: String,
    pub text_len: usize,
    pub bytes: usize,
    pub truncated: bool,
    pub links: usize,
    pub buttons: usize,
    pub inputs: usize,
    pub forms: usize,
}

pub fn compare_outcome(expected: &Expected, actual: &Observed, diffs: &mut Vec<String>) {
    if expected.ok != actual.ok {
        diffs.push(format!("ok {} != {}", actual.ok, expected.ok));
        return;
    }
    if !expected.ok {
        // The reference embeds the request URL in its error text, so compare
        // the failure class instead of the whole string.
        if expected.error.contains("many redirects") && actual.error != "many redirects" {
            diffs.push(format!("error {:?} != many redirects", actual.error));
        }
        if expected.error.contains("response exceeds")
            && !actual.error.starts_with("response exceeds")
        {
            diffs.push(format!("error {:?} != response exceeds", actual.error));
        }
        return;
    }
    compare("url", &expected.url, &actual.url, diffs);
    compare("status", &expected.status, &actual.status, diffs);
    compare("title", &expected.title, &actual.title, diffs);
    compare("text", &expected.text, &truncate(&actual.text, 2000), diffs);
    compare(
        "text_sha256",
        &expected.text_sha256,
        &actual.text_sha256,
        diffs,
    );
    compare("text_len", &expected.text_len, &actual.text_len, diffs);
    compare("bytes", &expected.bytes, &actual.bytes, diffs);
    compare("truncated", &expected.truncated, &actual.truncated, diffs);
    compare("links", &expected.links, &actual.links, diffs);
    compare("buttons", &expected.buttons, &actual.buttons, diffs);
    compare("inputs", &expected.inputs, &actual.inputs, diffs);
    compare("forms", &expected.forms, &actual.forms, diffs);
}

pub fn compare_requests(
    expected: &[RequestRecord],
    actual: &[RequestRecord],
    diffs: &mut Vec<String>,
) {
    if expected.len() != actual.len() {
        diffs.push(format!("requests {} != {}", actual.len(), expected.len()));
        return;
    }
    for (index, (expected, actual)) in expected.iter().zip(actual).enumerate() {
        if expected.method != actual.method
            || expected.path != actual.path
            || expected.query != actual.query
            || expected.body != actual.body
        {
            diffs.push(format!(
                "request {index}: {} {}?{} body={:?} != {} {}?{} body={:?}",
                actual.method,
                actual.path,
                actual.query,
                actual.body,
                expected.method,
                expected.path,
                expected.query,
                expected.body
            ));
        }
        for (name, value) in &expected.headers {
            if actual.headers.get(name) != Some(value) {
                diffs.push(format!(
                    "request {index} header {name}: {:?} != {:?}",
                    actual.headers.get(name),
                    Some(value)
                ));
            }
        }
        for name in actual.headers.keys() {
            if !expected.headers.contains_key(name) {
                diffs.push(format!("request {index} unexpected header {name}"));
            }
        }
    }
}

fn compare<T: PartialEq + std::fmt::Debug>(
    label: &str,
    expected: &T,
    actual: &T,
    diffs: &mut Vec<String>,
) {
    if expected != actual {
        diffs.push(format!("{label} {actual:?} != {expected:?}"));
    }
}

fn truncate(value: &str, limit: usize) -> String {
    value.chars().take(limit).collect()
}
