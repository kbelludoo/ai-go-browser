//! Node-level predicates ported from Go `browser/dom.go`.

use std::collections::HashMap;

use super::*;

impl Document {
    /// Go `Node.hidden`: inert or explicitly hidden subtrees and CSS that a
    /// layout-less tree can still evaluate.
    pub(super) fn hidden(&self, index: usize) -> bool {
        let node = self.node(index);
        if HIDDEN_TAGS.contains(&node.tag.as_str()) {
            return true;
        }
        if node.has("hidden") {
            return true;
        }
        if self.prop(index, "aria-hidden").eq_ignore_ascii_case("true") {
            return true;
        }
        let style = self
            .prop(index, "style")
            .to_ascii_lowercase()
            .replace(' ', "");
        style.contains("display:none") || style.contains("visibility:hidden")
    }

    /// Go `Node.visible`: the node and every ancestor are visible.
    pub(super) fn visible(&self, index: usize) -> bool {
        let mut current = Some(index);
        while let Some(node) = current {
            if self.hidden(node) {
                return false;
            }
            current = self.node(node).parent;
        }
        true
    }

    pub(super) fn content_editable(&self, index: usize) -> bool {
        let mut current = Some(index);
        while let Some(node) = current {
            if let Some(value) = self.node(node).attr("contenteditable") {
                let value = value.to_ascii_lowercase();
                match value.as_str() {
                    "" | "true" | "plaintext-only" => return true,
                    "false" => return false,
                    _ => {}
                }
            }
            current = self.node(node).parent;
        }
        false
    }

    pub(super) fn editable_control(&self, index: usize) -> bool {
        if NON_CONTROL_TAGS.contains(&self.tag(index)) {
            return false;
        }
        if self.has_role(index, "textbox") {
            return true;
        }
        if !self.content_editable(index) {
            return false;
        }
        self.node(index)
            .parent
            .is_none_or(|parent| !self.content_editable(parent))
    }

    pub(super) fn generic_button(&self, index: usize) -> bool {
        if NON_CONTROL_TAGS.contains(&self.tag(index)) {
            return false;
        }
        self.has_role(index, "button") || (self.tag(index) == "a" && !self.has(index, "href"))
    }

    pub(super) fn has_role(&self, index: usize, role: &str) -> bool {
        self.prop(index, "role")
            .split_whitespace()
            .any(|value| value.eq_ignore_ascii_case(role))
    }

    pub(super) fn control_type(&self, index: usize) -> String {
        if self.editable_control(index) {
            return "contenteditable".into();
        }
        if self.generic_button(index) {
            return "button".into();
        }
        let tag = self.tag(index);
        if tag != "input" && tag != "button" {
            return tag.to_owned();
        }
        let kind = self.prop(index, "type").to_ascii_lowercase();
        if tag == "button" {
            return match kind.as_str() {
                "button" | "reset" => kind,
                _ => "submit".into(),
            };
        }
        if INPUT_TYPES.contains(&kind.as_str()) || kind == "button" {
            kind
        } else {
            "text".into()
        }
    }

    pub(super) fn disabled(&self, index: usize) -> bool {
        if self.has(index, "disabled") {
            return true;
        }
        let mut current = Some(index);
        while let Some(node) = current {
            if self
                .prop(node, "aria-disabled")
                .trim()
                .eq_ignore_ascii_case("true")
            {
                return true;
            }
            current = self.node(node).parent;
        }
        let mut parent = self.node(index).parent;
        while let Some(node) = parent {
            if self.tag(node) == "fieldset" && self.has(node, "disabled") {
                let legend = self
                    .node(node)
                    .children
                    .iter()
                    .copied()
                    .find(|child| self.tag(*child) == "legend");
                if legend.is_none()
                    || !legend.is_some_and(|legend| self.is_descendant(index, legend))
                {
                    return true;
                }
            }
            parent = self.node(node).parent;
        }
        false
    }

    pub(super) fn is_descendant(&self, index: usize, ancestor: usize) -> bool {
        let mut current = Some(index);
        while let Some(node) = current {
            if node == ancestor {
                return true;
            }
            current = self.node(node).parent;
        }
        false
    }

    pub(super) fn read_only(&self, index: usize) -> bool {
        if self.editable_control(index) && !self.content_editable(index) {
            return true;
        }
        let mut current = Some(index);
        while let Some(node) = current {
            if self
                .prop(node, "aria-readonly")
                .trim()
                .eq_ignore_ascii_case("true")
            {
                return true;
            }
            current = self.node(node).parent;
        }
        if !self.has(index, "readonly") {
            return false;
        }
        TEXT_LIKE_TYPES.contains(&self.control_type(index).as_str())
    }

    pub(super) fn form_owner(
        &self,
        index: usize,
        ids: &HashMap<String, usize>,
        markers: &HashMap<String, usize>,
    ) -> Option<usize> {
        if let Some(marker) = self.node(index).attr("data-aigobrowser-form") {
            return markers.get(marker).copied();
        }
        if let Some(id) = self.node(index).attr("form") {
            return ids
                .get(id)
                .copied()
                .filter(|owner| self.tag(*owner) == "form");
        }
        let mut parent = self.node(index).parent;
        while let Some(node) = parent {
            if self.tag(node) == "form" {
                return Some(node);
            }
            parent = self.node(node).parent;
        }
        None
    }
}
