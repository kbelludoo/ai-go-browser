use crate::dom::{Field, parse_page};
use crate::{ActionRef, SemanticElement, SemanticSnapshot};

/// Extract the agent-facing semantic snapshot for a static HTML document.
///
/// The snapshot is a flat view over the same [`Page`] model the Go reference
/// produces, so reference ids (`l*`, `b*`, `i*`) and labels match the golden
/// fixtures in `rust/parity`.
pub fn parse_semantic_html(url: impl Into<String>, html: &str) -> SemanticSnapshot {
    let url = url.into();
    let page = parse_page(&url, html, 200, 0);

    let mut elements = Vec::new();
    for link in &page.links {
        let mut element = SemanticElement::new("a");
        element.name = Some(link.label.clone());
        element.href = Some(link.url.clone());
        element.action = ActionRef::new(link.reference.clone(), link.label.clone());
        elements.push(element);
    }
    for button in &page.buttons {
        elements.push(button_element(button));
    }
    for field in page
        .inputs
        .iter()
        .chain(page.forms.iter().flat_map(|form| form.fields.iter()))
    {
        elements.push(field_element(field));
    }

    SemanticSnapshot {
        url,
        title: page.title,
        elements,
    }
}

fn button_element(button: &crate::dom::Button) -> SemanticElement {
    let mut element = SemanticElement::new("button");
    element.name = Some(button.label.clone());
    element.text = Some(button.label.clone());
    element.action = ActionRef::new(button.reference.clone(), button.label.clone());
    element
}

fn field_element(field: &Field) -> SemanticElement {
    let tag = match field.kind.as_str() {
        "textarea" => "textarea",
        "select" => "select",
        "contenteditable" => "div",
        _ => "input",
    };
    let mut element = SemanticElement::new(tag);
    if !field.label.is_empty() {
        element.name = Some(field.label.clone());
        element.action = ActionRef::new(field.reference.clone(), field.label.clone());
    }
    element.text = if field.value.is_empty() {
        None
    } else {
        Some(field.value.clone())
    };
    element
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
        assert_eq!(
            snapshot.elements[0].accessible_name(),
            Some("Documentation")
        );
        assert_eq!(snapshot.elements[1].action.as_ref().unwrap().id, "b1");
        assert_eq!(snapshot.elements[2].action.as_ref().unwrap().id, "i1");
    }

    #[test]
    fn uses_explicit_label_for_form_fields() {
        let snapshot = parse_semantic_html(
            "https://example.com",
            r#"<label for="email">Work email</label><input id="email" type="email">"#,
        );
        assert_eq!(snapshot.elements[0].accessible_name(), Some("Work email"));
    }

    #[test]
    fn resolves_aria_labelledby_in_token_order() {
        let snapshot = parse_semantic_html(
            "https://example.com",
            r#"<span id="prefix">Billing</span><span id="field">address</span><input aria-labelledby="prefix field">"#,
        );
        assert_eq!(
            snapshot.elements[0].accessible_name(),
            Some("Billing address")
        );
    }

    #[test]
    fn aria_label_takes_precedence_over_labelledby_and_text() {
        let snapshot = parse_semantic_html(
            "https://example.com",
            r#"<span id="name">Other name</span><button aria-labelledby="name" aria-label="Save">Submit</button>"#,
        );
        assert_eq!(snapshot.elements[0].accessible_name(), Some("Save"));
    }

    #[test]
    fn empty_button_label_uses_the_reference_placeholder() {
        let snapshot = parse_semantic_html(
            "https://example.com",
            "<button>  </button><button>  Go   now </button>",
        );
        assert_eq!(snapshot.elements.len(), 2);
        assert_eq!(snapshot.elements[0].accessible_name(), Some("(button)"));
        assert_eq!(snapshot.elements[1].accessible_name(), Some("Go now"));
    }
}
