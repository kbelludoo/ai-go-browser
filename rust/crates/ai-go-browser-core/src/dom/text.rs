//! Visible text extraction with Go `pageText` semantics.

use super::*;

impl Document {
    /// Visible text with Go `pageText` normalization and a rune limit.
    pub(super) fn inner_text(&self, index: usize, limit: usize) -> String {
        if limit == 0 && self.visible(index) {
            // Unbounded variant: avoid the character accounting entirely.
            let mut builder = TextBuilder::unbounded();
            self.collect_text(index, &mut builder);
            return builder.finish();
        }
        let mut builder = TextBuilder::new(limit);
        self.collect_text(index, &mut builder);
        builder.finish()
    }

    pub(super) fn collect_text(&self, index: usize, builder: &mut TextBuilder) {
        if builder.is_done() || self.hidden(index) {
            return;
        }
        let node = self.node(index);
        if node.tag == "#text" {
            builder.add(&node.text);
            return;
        }
        if node.tag == "img" {
            builder.add(self.prop(index, "alt"));
        }
        for child in &node.children {
            self.collect_text(*child, builder);
            if builder.is_done() {
                return;
            }
        }
    }

    pub(super) fn raw_text(&self, index: usize, limit: usize) -> String {
        let mut out = String::new();
        let mut stack = vec![index];
        while let Some(current) = stack.pop() {
            if limit > 0 && out.len() > limit {
                break;
            }
            let node = self.node(current);
            if node.tag == "#text" {
                out.push_str(&node.text);
            }
            for child in node.children.iter().rev() {
                stack.push(*child);
            }
        }
        out
    }
}

/// Go `pageText`: collapses whitespace, inserts single spaces between runs and
/// stops at a rune limit.
pub(super) struct TextBuilder {
    out: String,
    runes: usize,
    limit: usize,
    truncated: bool,
}

impl TextBuilder {
    pub(super) fn new(limit: usize) -> Self {
        Self {
            out: String::new(),
            runes: 0,
            limit,
            truncated: false,
        }
    }

    pub(super) fn unbounded() -> Self {
        Self::new(0)
    }

    pub(super) fn is_done(&self) -> bool {
        self.truncated
    }

    pub(super) fn text_is_empty(&self) -> bool {
        self.out.is_empty()
    }

    pub(super) fn add(&mut self, input: &str) {
        if self.truncated {
            return;
        }
        let mut pending_space = !self.out.is_empty();
        for character in input.chars() {
            if character.is_whitespace() {
                pending_space = !self.out.is_empty();
                continue;
            }
            if pending_space {
                if !self.write(' ') {
                    return;
                }
                pending_space = false;
            }
            if !self.write(character) {
                return;
            }
        }
    }

    pub(super) fn write(&mut self, character: char) -> bool {
        if self.limit > 0 && self.runes >= self.limit {
            self.truncated = true;
            return false;
        }
        self.out.push(character);
        self.runes += 1;
        true
    }

    pub(super) fn finish(self) -> String {
        self.out
    }
}
