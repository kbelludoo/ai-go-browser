//! Page assembly: title, links, headings, text and controls.

use std::collections::HashMap;

use super::text::TextBuilder;
use super::*;

impl Document {
    /// Document-order walk over visible nodes, mirroring Go `domWalk`.
    pub(super) fn walk_visible(&self, mut visit: impl FnMut(usize)) {
        let mut stack = vec![self.root];
        while let Some(index) = stack.pop() {
            if self.hidden(index) {
                continue;
            }
            visit(index);
            for child in self.node(index).children.iter().rev() {
                stack.push(*child);
            }
        }
    }

    /// Control walk, mirroring Go `controlWalk`: hidden controls are included
    /// because they still submit, but inert subtrees are skipped.
    pub(super) fn walk_controls(&self, mut visit: impl FnMut(usize)) {
        let mut stack = vec![self.root];
        while let Some(index) = stack.pop() {
            if CONTROL_SKIP_TAGS.contains(&self.tag(index)) {
                continue;
            }
            visit(index);
            for child in self.node(index).children.iter().rev() {
                stack.push(*child);
            }
        }
    }

    pub(super) fn find_title(&self) -> String {
        let mut stack = vec![self.root];
        while let Some(index) = stack.pop() {
            if self.tag(index) == "title" {
                return normalize(&self.raw_text(index, 4096));
            }
            for child in self.node(index).children.iter().rev() {
                stack.push(*child);
            }
        }
        String::new()
    }

    pub(super) fn has_controls(&self) -> bool {
        let mut stack = vec![self.root];
        while let Some(index) = stack.pop() {
            let tag = self.tag(index);
            if tag == "form"
                || tag == "input"
                || tag == "select"
                || tag == "textarea"
                || tag == "button"
                || self.editable_control(index)
                || self.generic_button(index)
            {
                return true;
            }
            for child in self.node(index).children.iter().rev() {
                stack.push(*child);
            }
        }
        false
    }

    pub(super) fn page(&self, text_limit: usize) -> Page {
        let mut page = Page {
            url: self.url.clone(),
            title: self.find_title(),
            headings: Vec::new(),
            links: Vec::new(),
            buttons: Vec::new(),
            inputs: Vec::new(),
            forms: Vec::new(),
            text: String::new(),
        };

        let base = Url::parse(&self.url).ok();
        let base = base.as_ref();

        let mut text = TextBuilder::new(text_limit);
        self.walk_visible(|index| {
            let node = self.node(index);
            if node.tag == "#text" {
                text.add(&node.text);
                return;
            }
            if node.tag == "a" && node.has("href") {
                let href = resolve(base, self.prop(index, "href"));
                if !href.is_empty() {
                    let aria = self.prop(index, "aria-label");
                    let mut label = if aria.is_empty() {
                        self.inner_text(index, 1200)
                    } else {
                        aria.to_owned()
                    };
                    if label.is_empty() {
                        label = "(link)".into();
                    }
                    page.links.push(Link {
                        reference: format!("l{}", page.links.len() + 1),
                        label,
                        url: href,
                    });
                }
            }
            let tag = &node.tag;
            if tag.len() == 2 && tag.starts_with('h') && matches!(tag.as_bytes()[1], b'1'..=b'6') {
                page.headings.push(Heading {
                    level: tag.as_bytes()[1] - b'0',
                    text: self.inner_text(index, 1200),
                });
            }
        });

        if self.has_controls() {
            let (forms, buttons, inputs) = self.collect_forms(base);
            page.forms = forms;
            page.buttons = buttons;
            page.inputs = inputs;
        }

        page.text = text.finish();
        page
    }

    pub(super) fn collect_forms(&self, base: Option<&Url>) -> (Vec<Form>, Vec<Button>, Vec<Field>) {
        let mut ids = HashMap::new();
        let mut markers = HashMap::new();
        let mut labels: HashMap<String, Vec<usize>> = HashMap::new();
        let mut forms: Vec<Form> = Vec::new();
        let mut form_indexes = HashMap::new();

        self.walk_controls(|index| {
            if let Some(id) = self.node(index).attr("id").filter(|id| !id.is_empty()) {
                ids.entry(id.to_owned()).or_insert(index);
            }
            if self.tag(index) == "label"
                && let Some(target) = self
                    .node(index)
                    .attr("for")
                    .filter(|value| !value.is_empty())
            {
                labels.entry(target.to_owned()).or_default().push(index);
            }
            if self.tag(index) != "form" {
                return;
            }
            if let Some(marker) = self
                .node(index)
                .attr("data-aigobrowser")
                .filter(|m| !m.is_empty())
            {
                markers.insert(marker.to_owned(), index);
            }
            let method = self.prop(index, "method").to_ascii_uppercase();
            let method = if method == "POST" || method == "DIALOG" {
                method
            } else {
                "GET".into()
            };
            let enctype = self.prop(index, "enctype").to_ascii_lowercase();
            let enctype = if enctype == "multipart/form-data" || enctype == "text/plain" {
                enctype
            } else {
                "application/x-www-form-urlencoded".into()
            };
            let action = {
                let raw = self.prop(index, "action");
                if raw.is_empty() {
                    self.url.clone()
                } else {
                    let resolved = resolve(base, raw);
                    if resolved.is_empty() {
                        self.url.clone()
                    } else {
                        resolved
                    }
                }
            };
            form_indexes.insert(index, forms.len());
            forms.push(Form {
                reference: format!("f{}", forms.len() + 1),
                action,
                method,
                enctype,
                submit_ref: None,
                submit_label: "Enviar".into(),
                novalidate: self.has(index, "novalidate"),
                fields: Vec::new(),
            });
        });

        let mut buttons: Vec<Button> = Vec::new();
        let mut inputs: Vec<Field> = Vec::new();
        let mut field_count = 0usize;
        let mut radios: HashMap<String, (Option<usize>, usize)> = HashMap::new();

        self.walk_controls(|index| {
            let editable = self.editable_control(index);
            let generic = self.generic_button(index);
            let tag = self.tag(index);
            if tag != "input"
                && tag != "textarea"
                && tag != "select"
                && tag != "button"
                && !editable
                && !generic
            {
                return;
            }
            if (editable || generic) && !self.visible(index) {
                return;
            }

            let owner = self.form_owner(index, &ids, &markers);
            let mut associated = owner.and_then(|owner| form_indexes.get(&owner).copied());
            if editable || generic {
                associated = None;
            }
            let kind = self.control_type(index);

            if tag == "button"
                || kind == "button"
                || kind == "submit"
                || kind == "reset"
                || kind == "image"
            {
                let mut label = self.control_label(index, &ids, &labels);
                if label.is_empty() {
                    label = self.inner_text(index, 256);
                }
                if label.is_empty() {
                    label = self.prop(index, "value").to_owned();
                }
                if label.is_empty() {
                    label = "(button)".into();
                }
                let mut button = Button {
                    reference: format!("b{}", buttons.len() + 1),
                    id: self.prop(index, "id").to_owned(),
                    label,
                    kind: kind.clone(),
                    name: self.prop(index, "name").to_owned(),
                    value: self.prop(index, "value").to_owned(),
                    disabled: self.disabled(index),
                    form_ref: None,
                };
                if let Some(form_index) = associated {
                    button.form_ref = Some(forms[form_index].reference.clone());
                    if kind == "submit"
                        && !button.disabled
                        && forms[form_index].submit_ref.is_none()
                    {
                        forms[form_index].submit_ref = Some(button.reference.clone());
                        forms[form_index].submit_label = button.label.clone();
                    }
                }
                buttons.push(button);
                return;
            }

            field_count += 1;
            let mut field = Field {
                reference: format!("i{field_count}"),
                id: self.prop(index, "id").to_owned(),
                name: self.prop(index, "name").to_owned(),
                kind: kind.clone(),
                value: String::new(),
                label: self.control_label(index, &ids, &labels),
                disabled: self.disabled(index),
                readonly: self.read_only(index),
                checked: (kind == "checkbox" || kind == "radio") && self.has(index, "checked"),
                multiple: tag == "select" && self.has(index, "multiple"),
                required: self.has(index, "required"),
                pattern: self.prop(index, "pattern").to_owned(),
                options: Vec::new(),
            };
            if tag == "select" {
                let (options, value) = self.select_state(index);
                field.options = options;
                field.value = value;
            } else {
                field.value = self.value(index);
            }

            if kind == "radio" && field.checked && !field.name.is_empty() {
                let group = match associated {
                    Some(form_index) => {
                        format!("{}\u{0}{}", forms[form_index].reference, field.name)
                    }
                    None => format!("standalone\u{0}{}", field.name),
                };
                if let Some((previous_form, previous_field)) = radios.get(&group).copied() {
                    match previous_form {
                        Some(form_index) => {
                            if let Some(existing) = forms[form_index].fields.get_mut(previous_field)
                            {
                                existing.checked = false;
                            }
                        }
                        None => {
                            if let Some(existing) = inputs.get_mut(previous_field) {
                                existing.checked = false;
                            }
                        }
                    }
                }
                let slot = match associated {
                    Some(form_index) => forms[form_index].fields.len(),
                    None => inputs.len(),
                };
                radios.insert(group, (associated, slot));
            }

            match associated {
                Some(form_index) => forms[form_index].fields.push(field),
                None => inputs.push(field),
            }
        });

        (forms, buttons, inputs)
    }
}
