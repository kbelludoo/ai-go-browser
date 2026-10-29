//! Static DOM model and page extraction.
//!
//! This is a Rust port of the Go reference parser (`browser/dom.go` +
//! `browser/parser.go`). It keeps the observable rules - visibility, control
//! typing, accessible-name precedence, reference numbering, form ownership and
//! text normalization - because the port is compared against Go goldens in
//! `rust/parity`. When a rule looks odd, it is because the Go implementation
//! behaves that way and parity is measured, not assumed.

use std::collections::BTreeMap;

use ego_tree::NodeId;
use scraper::{Html, Node as ScraperNode};
use url::Url;

/// A heading extracted from the visible document.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Heading {
    pub level: u8,
    pub text: String,
}

/// A navigable link (`a[href]`).
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Link {
    pub reference: String,
    pub label: String,
    pub url: String,
}

/// A button-like control: `button`, submit/reset/image inputs, `role=button`,
/// and anchors without an `href`, matching the Go control walk.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Button {
    pub reference: String,
    pub id: String,
    pub label: String,
    pub kind: String,
    pub name: String,
    pub value: String,
    pub disabled: bool,
    pub form_ref: Option<String>,
}

/// An `<option>` inside a select control.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Choice {
    pub value: String,
    pub label: String,
    pub selected: bool,
    pub disabled: bool,
}

/// A form field, either standalone (`i*`) or owned by a form.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Field {
    pub reference: String,
    pub id: String,
    pub name: String,
    pub kind: String,
    pub value: String,
    pub label: String,
    pub disabled: bool,
    pub readonly: bool,
    pub checked: bool,
    pub multiple: bool,
    pub required: bool,
    pub pattern: String,
    pub options: Vec<Choice>,
}

/// A `<form>` with its owned fields and submit control.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Form {
    pub reference: String,
    pub action: String,
    pub method: String,
    pub enctype: String,
    pub submit_ref: Option<String>,
    pub submit_label: String,
    pub novalidate: bool,
    pub fields: Vec<Field>,
}

/// The agent-facing static page model.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Page {
    pub url: String,
    pub title: String,
    pub headings: Vec<Heading>,
    pub links: Vec<Link>,
    pub buttons: Vec<Button>,
    pub inputs: Vec<Field>,
    pub forms: Vec<Form>,
    pub text: String,
}

struct Node {
    tag: String,
    attrs: BTreeMap<String, String>,
    text: String,
    parent: Option<usize>,
    children: Vec<usize>,
}

struct Document {
    url: String,
    nodes: Vec<Node>,
    root: usize,
}

const HIDDEN_TAGS: [&str; 7] = [
    "script", "style", "noscript", "template", "svg", "head", "title",
];
const CONTROL_SKIP_TAGS: [&str; 6] = [
    "template", "script", "style", "noscript", "head", "datalist",
];
const NON_CONTROL_TAGS: [&str; 6] = [
    "#text",
    "#document",
    "input",
    "textarea",
    "select",
    "button",
];
const INPUT_TYPES: [&str; 21] = [
    "hidden",
    "text",
    "search",
    "tel",
    "url",
    "email",
    "password",
    "date",
    "month",
    "week",
    "time",
    "datetime-local",
    "number",
    "range",
    "color",
    "checkbox",
    "radio",
    "file",
    "submit",
    "image",
    "reset",
];
const TEXT_LIKE_TYPES: [&str; 11] = [
    "contenteditable",
    "textarea",
    "text",
    "search",
    "tel",
    "url",
    "email",
    "password",
    "date",
    "month",
    "week",
];

impl Node {
    fn attr(&self, name: &str) -> Option<&str> {
        self.attrs.get(name).map(String::as_str)
    }

    fn has(&self, name: &str) -> bool {
        self.attrs.contains_key(name)
    }
}

/// Parse a document into the agent-facing page model.
///
/// `text_limit` bounds `Page::text`; a value of zero keeps all text, matching
/// the Go `pageText` contract.
pub fn parse_page(url: &str, html: &str, status: u16, text_limit: usize) -> Page {
    let _ = status;
    Document::parse(url, html).page(text_limit)
}

impl Document {
    fn parse(url: &str, html: &str) -> Self {
        let parsed = Html::parse_document(html);
        let tree = parsed.tree;
        let mut nodes = Vec::new();
        let root = build(&mut nodes, &tree, tree.root().id(), None);
        Self {
            url: url.to_owned(),
            nodes,
            root,
        }
    }

    fn node(&self, index: usize) -> &Node {
        &self.nodes[index]
    }

    fn prop(&self, index: usize, name: &str) -> &str {
        self.node(index).attr(name).unwrap_or("")
    }

    fn has(&self, index: usize, name: &str) -> bool {
        self.node(index).has(name)
    }

    fn tag(&self, index: usize) -> &str {
        &self.node(index).tag
    }
}

fn build(
    nodes: &mut Vec<Node>,
    tree: &ego_tree::Tree<ScraperNode>,
    id: NodeId,
    parent: Option<usize>,
) -> usize {
    let (tag, attrs, text) = match tree.get(id).expect("node id").value() {
        ScraperNode::Element(element) => (
            element.name().to_owned(),
            element
                .attrs()
                .map(|(name, value)| (name.to_owned(), value.to_owned()))
                .collect::<BTreeMap<_, _>>(),
            String::new(),
        ),
        ScraperNode::Text(text) => ("#text".to_owned(), BTreeMap::new(), text.text.to_string()),
        _ => ("#other".to_owned(), BTreeMap::new(), String::new()),
    };

    let index = nodes.len();
    nodes.push(Node {
        tag,
        attrs,
        text,
        parent,
        children: Vec::new(),
    });

    let children = tree
        .get(id)
        .expect("node id")
        .children()
        .map(|child| child.id())
        .collect::<Vec<_>>();
    for child in children {
        let child_index = build(nodes, tree, child, Some(index));
        nodes[index].children.push(child_index);
    }

    index
}

/// Resolve a reference against the page URL, keeping only HTTP(S).
fn resolve(base: Option<&Url>, reference: &str) -> String {
    let Some(base) = base else {
        return String::new();
    };
    let Ok(joined) = base.join(reference.trim()) else {
        return String::new();
    };
    if joined.scheme() == "http" || joined.scheme() == "https" {
        joined.to_string()
    } else {
        String::new()
    }
}

fn normalize(input: &str) -> String {
    input.split_whitespace().collect::<Vec<_>>().join(" ")
}

mod fields;
mod node;
mod page;
mod text;

pub use fields::form_params;

#[cfg(test)]
mod tests;
