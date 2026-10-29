use std::collections::HashMap;

use scraper::{Html, Selector};

use crate::{ActionRef, SemanticElement, SemanticSnapshot};

/// Extract a deterministic, deliberately small set of actionable HTML elements.
/// This is a first parser slice, not a full browser accessibility tree.
pub fn parse_semantic_html(url: impl Into<String>, html: &str) -> SemanticSnapshot {
    let document = Html::parse_document(html);
    let title_selector = Selector::parse("title").expect("static selector");
    let title = document.select(&title_selector).next()
        .map(|e| e.text().collect::<String>().trim().to_owned()).unwrap_or_default();

    let mut id_names = HashMap::<String, String>::new();
    let id_selector = Selector::parse("[id]").expect("static selector");
    for node in document.select(&id_selector) {
        if let Some(id) = node.value().attr("id") {
            let aria = node.value().attr("aria-label").map(str::trim).filter(|s| !s.is_empty());
            let text = normalize_text(&node.text().collect::<Vec<_>>().join(" "));
            if let Some(name) = aria.map(str::to_owned).or(text) {
                id_names.insert(id.to_owned(), name);
            }
        }
    }

    let mut label_names = HashMap::<String, String>::new();
    let label_selector = Selector::parse("label[for]").expect("static selector");
    for label in document.select(&label_selector) {
        if let (Some(target), Some(name)) = (
            label.value().attr("for"),
            normalize_text(&label.text().collect::<Vec<_>>().join(" ")),
        ) {
            label_names.entry(target.to_owned()).or_insert(name);
        }
    }

    let selector = Selector::parse("a[href],button,input,textarea,select,[role='button'],[role='link']")
        .expect("static selector");
    let mut elements = Vec::new();
    let mut button_index = 0usize;
    let mut link_index = 0usize;
    let mut field_index = 0usize;

    for node in document.select(&selector) {
        let tag = node.value().name().to_ascii_lowercase();
        let role = node.value().attr("role").map(str::to_owned);
        let text = normalize_text(&node.text().collect::<Vec<_>>().join(" "));
        let aria = node.value().attr("aria-label").map(str::trim).filter(|s| !s.is_empty());
        let labelled_by = node.value().attr("aria-labelledby").map(|ids| {
            ids.split_whitespace().filter_map(|id| id_names.get(id).map(String::as_str))
                .collect::<Vec<_>>().join(" ")
        }).filter(|s| !s.trim().is_empty());
        let label_for = node.value().attr("id").and_then(|id| label_names.get(id).map(String::as_str));
        let title_attr = node.value().attr("title").map(str::trim).filter(|s| !s.is_empty());
        let value = node.value().attr("value").map(str::trim).filter(|s| !s.is_empty());
        let name = aria.map(str::to_owned)
            .or(labelled_by)
            .or_else(|| label_for.map(str::to_owned))
            .or_else(|| title_attr.map(str::to_owned))
            .or_else(|| value.map(str::to_owned))
            .or_else(|| text.clone());
        let href = node.value().attr("href").map(str::to_owned);
        let is_link = tag == "a" || role.as_deref() == Some("link");
        let is_field = matches!(tag.as_str(), "input" | "textarea" | "select");
        let action = if is_field {
            field_index += 1;
            name.as_deref().and_then(|label| ActionRef::new(format!("f{field_index}"), label))
        } else if is_link {
            link_index += 1;
            name.as_deref().and_then(|label| ActionRef::new(format!("l{link_index}"), label))
        } else {
            button_index += 1;
            name.as_deref().and_then(|label| ActionRef::new(format!("b{button_index}"), label))
        };

        let mut element = SemanticElement::new(tag);
        element.role = role;
        element.name = name;
        element.text = text;
        element.href = href;
        element.action = action;
        elements.push(element);
    }

    SemanticSnapshot { url: url.into(), title, elements }
}

fn normalize_text(input: &str) -> Option<String> {
    let value = input.split_whitespace().collect::<Vec<_>>().join(" ");
    if value.is_empty() { None } else { Some(value) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extracts_title_links_buttons_and_fields() {
        let html = r#"<html><head><title>Example page</title></head><body>
          <a href="/docs" aria-label="Documentation">docs</a>
          <button>Save changes</button><input aria-label="Email" type="email">
        </body></html>"#;
        let snapshot = parse_semantic_html("https://example.com", html);
        assert_eq!(snapshot.title, "Example page");
        assert_eq!(snapshot.elements.len(), 3);
        assert_eq!(snapshot.elements[0].action.as_ref().unwrap().id, "l1");
        assert_eq!(snapshot.elements[0].accessible_name(), Some("Documentation"));
        assert_eq!(snapshot.elements[1].action.as_ref().unwrap().id, "b1");
        assert_eq!(snapshot.elements[2].action.as_ref().unwrap().id, "f1");
    }

    #[test]
    fn uses_explicit_label_for_form_fields() {
        let snapshot = parse_semantic_html("https://example.com",
            r#"<label for="email">Work email</label><input id="email" type="email">"#);
        assert_eq!(snapshot.elements[0].accessible_name(), Some("Work email"));
    }

    #[test]
    fn resolves_aria_labelledby_in_token_order() {
        let snapshot = parse_semantic_html("https://example.com",
            r#"<span id="prefix">Billing</span><span id="field">address</span><input aria-labelledby="prefix field">"#);
        assert_eq!(snapshot.elements[0].accessible_name(), Some("Billing address"));
    }

    #[test]
    fn aria_label_takes_precedence_over_labelledby_and_text() {
        let snapshot = parse_semantic_html("https://example.com",
            r#"<span id="name">Other name</span><button aria-labelledby="name" aria-label="Save">Submit</button>"#);
        assert_eq!(snapshot.elements[0].accessible_name(), Some("Save"));
    }

    #[test]
    fn collapses_whitespace_and_ignores_empty_action_labels() {
        let snapshot = parse_semantic_html("https://example.com", "<button>  </button><button>  Go   now </button>");
        assert_eq!(snapshot.elements.len(), 2);
        assert!(snapshot.elements[0].action.is_none());
        assert_eq!(snapshot.elements[1].accessible_name(), Some("Go now"));
    }
}
