#![forbid(unsafe_code)]

mod action;
mod actions;
mod browser;
mod destination;
mod dom;
mod encoding;
mod memory;
mod parser;
mod semantic;
mod session;
mod sniff;
mod transport;
mod url_policy;

pub use action::ActionRef;
pub use actions::BrowserAction;
pub use browser::{Browser, BrowserError, BrowserOutput, HttpSnapshotBrowser};
pub use destination::{DestinationPolicy, HostResolver, PublicInternet, SystemResolver};
pub use dom::{Button, Choice, Field, Form, Heading, Link, Page, form_params, parse_page};
pub use encoding::{DecodeError, DecodedBody, decode};
pub use memory::{MemoryEntry, MemoryStore};
pub use parser::parse_semantic_html;
pub use semantic::{SemanticElement, SemanticSnapshot};
pub use session::Session;
pub use transport::{
    FetchedPage, HttpResponse, MAX_BODY, MAX_REDIRECTS, MAX_RESOURCE_BYTES, SafeHttpTransport,
    TransportError, encode_form,
};
pub use url_policy::{UrlPolicyError, validate_syntax, validate_url};

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn action_ref_rejects_blank_id_and_label() {
        assert!(ActionRef::new(" ", "Submit").is_none());
        assert!(ActionRef::new("b1", " ").is_none());
    }
}
