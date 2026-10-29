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

use ai_go_browser_core::{Page, parse_page};
use serde::Deserialize;

#[derive(Debug, Deserialize)]
struct Golden {
    fixture: String,
    url: String,
    title: String,
    #[serde(default)]
    headings: Vec<HeadingGolden>,
    #[serde(default)]
    links: Vec<LinkGolden>,
    #[serde(default)]
    buttons: Vec<ButtonGolden>,
    #[serde(default)]
    inputs: Vec<FieldGolden>,
    #[serde(default)]
    forms: Vec<FormGolden>,
}

#[derive(Debug, Deserialize)]
struct HeadingGolden {
    level: u8,
    text: String,
}

#[derive(Debug, Deserialize)]
struct LinkGolden {
    #[serde(rename = "ref")]
    reference: String,
    label: String,
    url: String,
}

#[derive(Debug, Deserialize)]
struct ButtonGolden {
    #[serde(rename = "ref")]
    reference: String,
    label: String,
    #[serde(rename = "type", default)]
    kind: String,
    #[serde(default)]
    disabled: bool,
    #[serde(default)]
    form_ref: Option<String>,
}

#[derive(Debug, Deserialize)]
struct FieldGolden {
    #[serde(rename = "ref")]
    reference: String,
    #[serde(default)]
    name: String,
    #[serde(rename = "type", default)]
    kind: String,
    #[serde(default)]
    label: Option<String>,
}

#[derive(Debug, Deserialize)]
struct FormGolden {
    #[serde(rename = "ref")]
    reference: String,
    action: String,
    method: String,
    enctype: String,
    #[serde(default)]
    submit_ref: Option<String>,
    #[serde(default)]
    submit_label: Option<String>,
    #[serde(default)]
    fields: Vec<FieldGolden>,
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

    let mut matched = 0_usize;
    let mut total_diffs = 0_usize;
    for path in &goldens {
        let raw = fs::read_to_string(path).expect("golden json");
        let golden: Golden = serde_json::from_str(&raw).expect("golden schema");
        let html = fs::read_to_string(fixtures.join(&golden.fixture)).expect("fixture html");
        let page = parse_page(&golden.url, &html, 200, 0);

        let diffs = compare(&golden, &page);
        if diffs.is_empty() {
            matched += 1;
            println!("{:<16} match", golden.fixture);
            continue;
        }

        total_diffs += diffs.len();
        println!(
            "{:<16} {} diffs{}",
            golden.fixture,
            diffs.len(),
            if diffs.len() > 3 {
                format!(": {}; ...", diffs[..3].join("; "))
            } else {
                format!(": {}", diffs.join("; "))
            }
        );
    }

    println!(
        "\n{matched}/{} fixtures match, {total_diffs} differences",
        goldens.len()
    );
    if matched == goldens.len() {
        ExitCode::SUCCESS
    } else {
        ExitCode::FAILURE
    }
}

fn compare(golden: &Golden, page: &Page) -> Vec<String> {
    let mut diffs = Vec::new();

    if page.title != golden.title {
        diffs.push(format!("title {:?} != {:?}", page.title, golden.title));
    }

    compare_len(
        "headings",
        golden.headings.len(),
        page.headings.len(),
        &mut diffs,
    );
    for (index, expected) in golden.headings.iter().enumerate() {
        if let Some(actual) = page.headings.get(index)
            && (actual.level != expected.level || actual.text != expected.text)
        {
            diffs.push(format!(
                "h{index} {:?}:{} != {:?}:{}",
                actual.text, actual.level, expected.text, expected.level
            ));
        }
    }

    compare_len("links", golden.links.len(), page.links.len(), &mut diffs);
    for (index, expected) in golden.links.iter().enumerate() {
        if let Some(actual) = page.links.get(index)
            && (actual.reference != expected.reference
                || actual.label != expected.label
                || actual.url != expected.url)
        {
            diffs.push(format!(
                "link{index} ({} {:?} {}) != ({} {:?} {})",
                actual.reference,
                actual.label,
                actual.url,
                expected.reference,
                expected.label,
                expected.url
            ));
        }
    }

    compare_len(
        "buttons",
        golden.buttons.len(),
        page.buttons.len(),
        &mut diffs,
    );
    for (index, expected) in golden.buttons.iter().enumerate() {
        if let Some(actual) = page.buttons.get(index)
            && (actual.reference != expected.reference
                || actual.label != expected.label
                || actual.kind != expected.kind
                || actual.disabled != expected.disabled
                || actual.form_ref != expected.form_ref)
        {
            diffs.push(format!(
                "button{index} ({} {:?} {} disabled={}) != ({} {:?} {} disabled={})",
                actual.reference,
                actual.label,
                actual.kind,
                actual.disabled,
                expected.reference,
                expected.label,
                expected.kind,
                expected.disabled
            ));
        }
    }

    compare_len("inputs", golden.inputs.len(), page.inputs.len(), &mut diffs);
    for (index, expected) in golden.inputs.iter().enumerate() {
        if let Some(actual) = page.inputs.get(index)
            && (actual.reference != expected.reference
                || actual.name != expected.name
                || actual.kind != expected.kind
                || present(&actual.label) != expected.label.as_deref())
        {
            diffs.push(format!(
                "input{index} ({} {:?} {}) != ({} {:?} {})",
                actual.reference,
                actual.label,
                actual.kind,
                expected.reference,
                expected.label.as_deref().unwrap_or(""),
                expected.kind
            ));
        }
    }

    compare_len("forms", golden.forms.len(), page.forms.len(), &mut diffs);
    for (index, expected) in golden.forms.iter().enumerate() {
        let Some(actual) = page.forms.get(index) else {
            continue;
        };
        if actual.reference != expected.reference
            || actual.action != expected.action
            || actual.method != expected.method
            || actual.enctype != expected.enctype
            || actual.submit_ref != expected.submit_ref
            || present(&actual.submit_label) != expected.submit_label.as_deref()
        {
            diffs.push(format!(
                "form{index} ({} {} {} submit={:?} {:?}) != ({} {} {} submit={:?} {:?})",
                actual.reference,
                actual.method,
                actual.action,
                actual.submit_ref,
                actual.submit_label,
                expected.reference,
                expected.method,
                expected.action,
                expected.submit_ref,
                expected.submit_label
            ));
        }
        let expected_refs = expected
            .fields
            .iter()
            .map(|field| field.reference.as_str())
            .collect::<Vec<_>>();
        let actual_refs = actual
            .fields
            .iter()
            .map(|field| field.reference.as_str())
            .collect::<Vec<_>>();
        if actual_refs != expected_refs {
            diffs.push(format!(
                "form{index} fields {actual_refs:?} != {expected_refs:?}"
            ));
        }
    }

    diffs
}

fn compare_len(label: &str, expected: usize, actual: usize, diffs: &mut Vec<String>) {
    if expected != actual {
        diffs.push(format!("{label} {actual} != {expected}"));
    }
}

/// Go marks `label` and `submit_label` as `omitempty`, so an empty Rust string
/// is the same observation as an absent key in the golden JSON.
fn present(value: &str) -> Option<&str> {
    if value.is_empty() { None } else { Some(value) }
}
