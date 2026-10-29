//! Accessible names, option selection and control values.

use std::collections::HashMap;

use super::text::TextBuilder;
use super::*;

impl Document {
    /// Go `controlLabel`: snapshot label, `aria-label`, `aria-labelledby`,
    /// explicit `label[for]`, wrapping `label`, then `placeholder`.
    pub(super) fn control_label(
        &self,
        index: usize,
        ids: &HashMap<String, usize>,
        labels: &HashMap<String, Vec<usize>>,
    ) -> String {
        let aria = normalize(self.prop(index, "aria-label"));
        if !aria.is_empty() {
            return aria;
        }

        let mut builder = TextBuilder::new(512);
        for id in self.prop(index, "aria-labelledby").split_whitespace() {
            if let Some(source) = ids.get(id) {
                builder.add(&self.inner_text(*source, 512));
            }
        }
        if !builder.text_is_empty() {
            return builder.finish();
        }

        if let Some(id) = self.node(index).attr("id")
            && let Some(targets) = labels.get(id)
        {
            for label in targets {
                builder.add(&self.inner_text(*label, 512));
            }
        }

        let mut parent = self.node(index).parent;
        while let Some(node) = parent {
            if self.tag(node) == "label" && !self.has(node, "for") {
                builder.add(&self.inner_text(node, 512));
                break;
            }
            parent = self.node(node).parent;
        }
        if !builder.text_is_empty() {
            return builder.finish();
        }

        self.prop(index, "placeholder").to_owned()
    }

    pub(super) fn option_nodes(&self, index: usize) -> Vec<usize> {
        let mut options: Vec<usize> = Vec::new();
        let mut stack = vec![index];
        while let Some(current) = stack.pop() {
            if current != index && self.tag(current) == "select" {
                continue;
            }
            if self.tag(current) == "option" {
                options.push(current);
                continue;
            }
            for child in self.node(current).children.iter().rev() {
                stack.push(*child);
            }
        }
        options
    }

    pub(super) fn select_state(&self, index: usize) -> (Vec<Choice>, String) {
        let multiple = self.has(index, "multiple");
        let mut choices: Vec<Choice> = Vec::new();
        let mut selected: Option<usize> = None;
        let mut first_enabled: Option<usize> = None;

        for option in self.option_nodes(index) {
            let mut disabled = self.has(option, "disabled");
            let mut parent = self.node(option).parent;
            while let Some(node) = parent {
                if node == index {
                    break;
                }
                if self.tag(node) == "optgroup" && self.has(node, "disabled") {
                    disabled = true;
                }
                parent = self.node(node).parent;
            }
            let label = {
                let explicit = self.prop(option, "label");
                if explicit.is_empty() {
                    self.inner_text(option, 512)
                } else {
                    explicit.to_owned()
                }
            };
            let is_selected = self.has(option, "selected");
            if first_enabled.is_none() && !disabled {
                first_enabled = Some(choices.len());
            }
            if is_selected && !multiple {
                if let Some(previous) = selected {
                    choices[previous].selected = false;
                }
                selected = Some(choices.len());
            }
            choices.push(Choice {
                value: self.option_value(option),
                label,
                selected: is_selected,
                disabled,
            });
        }

        if !multiple
            && selected.is_none()
            && let Some(first) = first_enabled
        {
            let size = self.prop(index, "size").parse::<i64>().unwrap_or(0);
            if size <= 1 {
                choices[first].selected = true;
            }
        }

        let value = choices
            .iter()
            .find(|choice| choice.selected)
            .map(|choice| choice.value.clone())
            .unwrap_or_default();
        (choices, value)
    }

    pub(super) fn option_value(&self, index: usize) -> String {
        match self.node(index).attr("value") {
            Some(value) => value.to_owned(),
            None => normalize(&self.raw_text(index, 0)),
        }
    }

    pub(super) fn value(&self, index: usize) -> String {
        let tag = self.tag(index);
        if tag == "textarea" || self.editable_control(index) {
            return self.raw_text(index, 0);
        }
        if tag == "select" {
            return self.select_state(index).1;
        }
        if let Some(value) = self.node(index).attr("value") {
            return value.to_owned();
        }
        let kind = self.control_type(index);
        if kind == "checkbox" || kind == "radio" {
            return "on".into();
        }
        String::new()
    }
}

/// Go `formParams`: successful controls in document order, with repeated select
/// values and a named submitter. Disabled and unchecked controls do not
/// contribute.
pub fn form_params(form: &Form, submitter: Option<&Button>) -> Vec<(String, String)> {
    let mut params = Vec::new();
    for field in &form.fields {
        if field.disabled
            || field.name.is_empty()
            || matches!(
                field.kind.as_str(),
                "file" | "button" | "submit" | "reset" | "image"
            )
        {
            continue;
        }
        if matches!(field.kind.as_str(), "checkbox" | "radio") && !field.checked {
            continue;
        }
        let name = normalize_form_newlines(&field.name);
        if field.kind == "select" {
            for choice in &field.options {
                if choice.selected && !choice.disabled {
                    params.push((name.clone(), normalize_form_newlines(&choice.value)));
                }
            }
            continue;
        }
        params.push((name, normalize_form_newlines(&field.value)));
    }
    if let Some(button) = submitter
        && !button.disabled
        && button.kind == "submit"
        && !button.name.is_empty()
    {
        params.push((
            normalize_form_newlines(&button.name),
            normalize_form_newlines(&button.value),
        ));
    }
    params
}

/// URL-encoded HTML submissions normalize lone CR/LF in names and values.
fn normalize_form_newlines(value: &str) -> String {
    if !value.contains(['\r', '\n']) {
        return value.to_owned();
    }
    value
        .replace("\r\n", "\n")
        .replace('\r', "\n")
        .replace('\n', "\r\n")
}
