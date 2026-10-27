package main

import "core:testing"

@(test)
test_static_button_action_inference :: proc(t: ^testing.T) {
	raw := "<form method='POST' action='/buy'><button>Comprar</button><button formaction='/preview' formmethod='GET'>Prévia</button></form><button hx-get='/api/items'>Carregar</button><button hx-post='/api/send'>Enviar</button><button data-url='/api/unknown'>Dados</button><button onclick=\"fetch('/api/no')\">JS dinâmico</button><button disabled hx-get='/blocked'>Desabilitado</button>"
	p := parse_html(raw, "https://example.com/shop")
	testing.expect(t, len(p.buttons) == 7)
	testing.expect(t, p.buttons[0].action == "https://example.com/buy")
	testing.expect(t, p.buttons[0].method == "post")
	testing.expect(t, p.buttons[0].source == "form")
	testing.expect(t, p.buttons[1].action == "https://example.com/preview")
	testing.expect(t, p.buttons[1].method == "get")
	testing.expect(t, p.buttons[2].action == "https://example.com/api/items")
	testing.expect(t, p.buttons[2].method == "get")
	testing.expect(t, p.buttons[3].method == "post")
	testing.expect(t, p.buttons[4].source == "data-url")
	testing.expect(t, p.buttons[4].method == "") // no invented method
	testing.expect(t, p.buttons[5].action == "") // scripts are not evaluated
	testing.expect(t, p.buttons[6].disabled)
}

@(test)
test_reject_button_javascript_actions :: proc(t: ^testing.T) {
	p := parse_html("<button hx-get='javascript:alert(1)'>Unsafe</button>", "https://example.com")
	testing.expect(t, len(p.buttons) == 1)
	testing.expect(t, p.buttons[0].action == "")
	testing.expect(t, p.buttons[0].source == "")
}
