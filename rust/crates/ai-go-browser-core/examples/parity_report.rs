//! Differential report between the Go reference parser and the Rust port.
//!
//! `rust/parity/gen` writes golden snapshots with the Go implementation. This
//! example replays the same fixtures through the Rust parser and prints where
//! the two disagree, so parity work is measured instead of assumed.
//!
//! ```text
//! cargo run --example parity_report
//! ```
//!
//! The example exits non-zero while any fixture differs, which makes it useful
//! locally and in a manual comparison job, but it is not part of `cargo test`.

#![forbid(unsafe_code)]

use std::{fs, path::PathBuf, process::ExitCode};

use ai_go_browser_core::{SemanticSnapshot, parse_semantic_html};
use serde::Deserialize;

#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct Golden {
    fixture: String,
    url: String,
    title: String,
    headings: Vec<Heading>,
    links: Vec<Link>,
    buttons: Vec<Button>,
    inputs: Vec<Input>,
}

#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct Heading {
    level: u32,
    text: String,
}

#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct Link {
    #[serde(rename = "ref")]
    reference: String,
    label: String,
    url: String,
}

#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct Button {
    #[serde(rename = "ref")]
    reference: String,
    label: String,
}

#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct Input {
    name: String,
    label: Option<String>,
}

fn main() -> ExitCode {
    let root = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../parity");
    let fixtures = root.join("fixtures");
    let golden_dir = root.join("golden");

    let mut goldens = match fs::read_dir(&golden_dir) {
        Ok(entries) => entries
            .filter_map(Result::ok)
            .map(|entry| entry.path())
            .filter(|path| path.extension().is_some_and(|ext| ext == "json"))
            .collect::<Vec<_>>(),
        Err(error) => {
            eprintln!("read {}: {error}", golden_dir.display());
            return ExitCode::FAILURE;
        }
    };
    goldens.sort();

    if goldens.is_empty() {
        eprintln!(
            "no goldens in {}; run `go run ./rust/parity/gen`",
            golden_dir.display()
        );
        return ExitCode::FAILURE;
    }

    let mut mismatched = 0_usize;
    for path in &goldens {
        let raw = fs::read_to_string(path).expect("golden json");
        let golden: Golden = serde_json::from_str(&raw).expect("golden schema");
        let html_path = fixtures.join(&golden.fixture);
        let html = fs::read_to_string(&html_path).expect("fixture html");
        let snapshot = parse_semantic_html(golden.url.clone(), &html);

        let report = compare(&golden, &snapshot);
        if !report.is_empty() {
            mismatched += 1;
            println!("{}: {report}", golden.fixture);
        } else {
            println!("{}: match", golden.fixture);
        }
    }

    println!(
        "\n{}/{} fixtures match",
        goldens.len() - mismatched,
        goldens.len()
    );
    if mismatched == 0 {
        ExitCode::SUCCESS
    } else {
        ExitCode::FAILURE
    }
}

fn compare(golden: &Golden, snapshot: &SemanticSnapshot) -> String {
    let mut diffs = Vec::new();

    if snapshot.title != golden.title {
        diffs.push(format!("title {:?} != {:?}", snapshot.title, golden.title));
    }

    let links = collect(snapshot, 'l');
    let buttons = collect(snapshot, 'b');
    let fields = collect(snapshot, 'f');

    compare_count("links", golden.links.len(), links.len(), &mut diffs);
    compare_count("buttons", golden.buttons.len(), buttons.len(), &mut diffs);
    compare_count("fields", golden.inputs.len(), fields.len(), &mut diffs);

    for (index, expected) in golden.links.iter().enumerate() {
        if let Some(actual) = links.get(index) {
            if actual.0 != expected.label {
                diffs.push(format!(
                    "link{} label {:?} != {:?}",
                    index + 1,
                    actual.0,
                    expected.label
                ));
            }
            if actual.1 != Some(expected.url.as_str()) {
                diffs.push(format!(
                    "link{} url {:?} != {:?}",
                    index + 1,
                    actual.1,
                    expected.url
                ));
            }
        }
    }

    for (index, expected) in golden.buttons.iter().enumerate() {
        if let Some(actual) = buttons.get(index)
            && actual.0 != expected.label
        {
            diffs.push(format!(
                "button{} label {:?} != {:?}",
                index + 1,
                actual.0,
                expected.label
            ));
        }
    }

    for (index, expected) in golden.inputs.iter().enumerate() {
        if let Some(actual) = fields.get(index)
            && actual.0 != expected.label.as_deref().unwrap_or_default()
        {
            diffs.push(format!(
                "field{} label {:?} != {:?}",
                index + 1,
                actual.0,
                expected.label
            ));
        }
    }

    if !golden.headings.is_empty() {
        diffs.push(format!(
            "headings not implemented (go: {})",
            golden
                .headings
                .iter()
                .map(|heading| format!("h{}:{}", heading.level, heading.text))
                .collect::<Vec<_>>()
                .join(", ")
        ));
    }

    diffs.join("; ")
}

fn compare_count(label: &str, expected: usize, actual: usize, diffs: &mut Vec<String>) {
    if expected != actual {
        diffs.push(format!("{label} {actual} != {expected}"));
    }
}

fn collect(snapshot: &SemanticSnapshot, prefix: char) -> Vec<(&str, Option<&str>)> {
    snapshot
        .elements
        .iter()
        .filter(|element| {
            element
                .action
                .as_ref()
                .is_some_and(|action| action.id.starts_with(prefix))
        })
        .map(|element| {
            (
                element.accessible_name().unwrap_or(""),
                element.href.as_deref(),
            )
        })
        .collect()
}
