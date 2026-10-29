use super::*;

const URL: &str = "https://example.test/page";

fn page(html: &str) -> Page {
    parse_page(URL, html, 200, 0)
}

#[test]
fn select_defaults_to_the_first_enabled_option() {
    let page = page(
        r#"<select name="plan">
             <option disabled>Pick one</option>
             <option value="pro">Pro</option>
             <option value="team">Team</option>
           </select>"#,
    );
    let field = &page.inputs[0];
    assert_eq!(field.value, "pro");
    assert!(field.options[1].selected);
    assert!(!field.options[0].selected);
}

#[test]
fn sized_select_has_no_implicit_selection() {
    let page = page(r#"<select name="plan" size="4"><option value="pro">Pro</option></select>"#);
    assert_eq!(page.inputs[0].value, "");
    assert!(!page.inputs[0].options[0].selected);
}

#[test]
fn form_attribute_adopts_detached_controls() {
    let page = page(
        r#"<form id="checkout"><input name="email"></form>
           <input form="checkout" name="coupon">"#,
    );
    assert_eq!(page.forms.len(), 1);
    let refs = page.forms[0]
        .fields
        .iter()
        .map(|field| field.reference.as_str())
        .collect::<Vec<_>>();
    assert_eq!(refs, vec!["i1", "i2"]);
}

#[test]
fn hidden_controls_still_participate_in_submission() {
    let page =
        page(r#"<div style="display:none"><input type="hidden" name="token" value="abc"></div>"#);
    assert_eq!(page.inputs.len(), 1);
    assert_eq!(page.inputs[0].value, "abc");
}

#[test]
fn only_the_last_checked_radio_in_a_group_stays_checked() {
    let page = page(
        r#"<input type="radio" name="size" value="s" checked>
           <input type="radio" name="size" value="m" checked>"#,
    );
    assert!(!page.inputs[0].checked);
    assert!(page.inputs[1].checked);
    assert_eq!(page.inputs[0].value, "s");
}

#[test]
fn page_text_keeps_only_the_requested_number_of_runes() {
    let limited = parse_page(URL, "<p>abc def ghi</p>", 200, 5);
    assert_eq!(limited.text, "abc d");
    let full = parse_page(URL, "<p>abc   def</p>", 200, 0);
    assert_eq!(full.text, "abc def");
}
