//! Replay of one golden scenario through the Rust transport.

use ai_go_browser_core::{
    HttpResponse, MAX_BODY, Page, SafeHttpTransport, TransportError, decode, form_params,
    parse_page,
};
use reqwest::Method;
use sha2::{Digest, Sha256};

use crate::golden::{Observed, Scenario};

pub async fn replay(
    transport: &SafeHttpTransport,
    base: &str,
    scenario: &Scenario,
) -> Observed {
    let mut page: Option<Page> = None;
    let mut response: Option<HttpResponse> = None;
    for step in &scenario.steps {
        match step.cmd.as_str() {
            "open" => match transport.get(&absolute(base, &step.url)).await {
                Ok(fetched) => {
                    response = Some(fetched.clone());
                    page = decode_page(&fetched);
                }
                Err(error) => return failed(error),
            },
            "submit" => {
                let Some(current) = page.as_ref() else {
                    return failed(TransportError::Request(
                        "submit without an open page".into(),
                    ));
                };
                let Some(form) = current
                    .forms
                    .iter()
                    .find(|form| form.reference == step.reference())
                else {
                    return failed(TransportError::Request(format!(
                        "form {} not found",
                        step.reference()
                    )));
                };
                let submitter = form.submit_ref.as_ref().and_then(|reference| {
                    current.buttons.iter().find(|b| &b.reference == reference)
                });
                let fields = form_params(form, submitter);
                let result = if form.method.eq_ignore_ascii_case("post") {
                    transport.post_form(&form.action, &fields).await
                } else {
                    transport
                        .request(Method::GET, &form.action, Some(&fields))
                        .await
                };
                match result {
                    Ok(fetched) => {
                        response = Some(fetched.clone());
                        page = decode_page(&fetched);
                    }
                    Err(error) => return failed(error),
                }
            }
            other => return failed(TransportError::Request(format!("unknown step {other}"))),
        }
    }

    let Some(page) = page else {
        return failed(TransportError::Request("scenario produced no page".into()));
    };
    let Some(response) = response else {
        return failed(TransportError::Request(
            "scenario produced no response".into(),
        ));
    };
    let digest = Sha256::digest(page.text.as_bytes());
    Observed {
        ok: true,
        error: String::new(),
        url: response
            .final_url
            .strip_prefix(base)
            .unwrap_or(&response.final_url)
            .to_owned(),
        status: response.status,
        title: page.title.clone(),
        text: page.text.clone(),
        text_sha256: hex(&digest),
        text_len: page.text.chars().count(),
        bytes: response.wire_bytes,
        truncated: response.truncated,
        links: page.links.len(),
        buttons: page.buttons.len(),
        inputs: page.inputs.len(),
        forms: page.forms.len(),
    }
}

fn decode_page(response: &HttpResponse) -> Option<Page> {
    let content_type = response.content_type.clone().unwrap_or_default();
    let decoded = decode(&content_type, &response.body, MAX_BODY).ok()?;
    Some(parse_page(
        &response.final_url,
        &decoded.text,
        response.status,
        0,
    ))
}

fn failed(error: TransportError) -> Observed {
    Observed {
        error: describe(&error),
        ..Observed::default()
    }
}

fn describe(error: &TransportError) -> String {
    match error {
        TransportError::TooManyRedirects => "many redirects".into(),
        TransportError::ResponseTooLarge { limit } => format!("response exceeds {limit} bytes"),
        TransportError::Policy(policy) => format!("policy: {policy:?}"),
        TransportError::InvalidRedirect => "invalid redirect".into(),
        TransportError::InvalidCharset(label) => format!("unsupported response charset {label:?}"),
        TransportError::DecodeLimit { limit, decoded } => {
            if *decoded {
                format!("decoded UTF-8 response exceeds {limit} bytes")
            } else {
                format!("encoded response exceeds {limit} bytes")
            }
        }
        TransportError::Dns(message) | TransportError::Request(message) => message.clone(),
    }
}

fn absolute(base: &str, target: &str) -> String {
    if target.starts_with("http") {
        target.to_owned()
    } else {
        format!("{base}{target}")
    }
}

fn hex(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}
