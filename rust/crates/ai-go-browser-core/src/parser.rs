use scraper::{Html, Selector};

use crate::{ActionRef, SemanticElement, SemanticSnapshot};

/// Extract a deterministic, deliberately small set of actionable HTML elements.
/// This is a first parser slice, not a full browser accessibility tree.
pub fn parse_semantic_html(url: impl Into<String>, html: &str) -> SemanticSnapshot {
    let document = Html::parse_document(html);
    let title_selector = Selector::parse("title").expect("static selector");
    let title = document.select(&title_selector).next()
        .map(|e| e.text().collect::<String>().trim().to_owned()).unwrap_or_default();

    let selector = Selector::parse("a[href],button,input,textarea,select,[role='button'],[role='link']")
        .expect("static selector");
    let mut elements = Vec::new();
    let mut button_index = 0usize;
    let mut link_index = 0usize;
    let mut field_index = 0usize;

    for node in document.select(&selector) {
        let tag = node.value().name().to_ascii_lowercase();
        let role = node.value().attr("role").map(str::to_owned);
        let text = normalize_text(node.text().collect::<Vec<_>>().join(" ").as_str());
        let aria = node.value().attr("aria-label").map(str::trim).filter(|s| !s.is_empty());
        let title_attr = node.value().attr("title").map(str::trim).filter(|s| !s.is_empty());
        let value = node.value().attr("value").map(str::trim).filter(|s| !s.is_empty());
        let name = aria.or(title_attr).or(value).map(str::to_owned).or_else(|| text.clone());
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
    fn collapses_whitespace_and_ignores_empty_action_labels() {
        let snapshot = parse_semantic_html("https://example.com", "<button>  </button><button>  Go   now </button>");
        assert_eq!(snapshot.elements.len(), 2);
        assert!(snapshot.elements[0].action.is_none());
        assert_eq!(snapshot.elements[1].accessible_name(), Some("Go now"));
    }
}
