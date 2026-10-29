use crate::ActionRef;

/// Stable commands that an agent can issue against a semantic browser snapshot.
/// Values are intentionally explicit so adapters can validate targets before acting.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum BrowserAction {
    Click { target: ActionRef },
    Fill { target: ActionRef, value: String },
    SelectOption { target: ActionRef, value: String },
    Submit { target: ActionRef },
    Back,
    Forward,
    Reload,
    WaitForText { text: String },
    ExtractText,
}

impl BrowserAction {
    pub fn target(&self) -> Option<&ActionRef> {
        match self {
            Self::Click { target }
            | Self::Fill { target, .. }
            | Self::SelectOption { target, .. }
            | Self::Submit { target } => Some(target),
            Self::Back | Self::Forward | Self::Reload | Self::WaitForText { .. } | Self::ExtractText => None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn target_is_exposed_only_for_element_actions() {
        let target = ActionRef::new("b2", "Save").unwrap();
        assert_eq!(BrowserAction::Click { target: target.clone() }.target(), Some(&target));
        assert!(BrowserAction::Back.target().is_none());
        assert!(BrowserAction::ExtractText.target().is_none());
    }
}
