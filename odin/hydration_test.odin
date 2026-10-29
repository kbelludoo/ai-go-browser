package main

import "core:testing"
import "core:strings"

@(test)
test_hydration_extraction_without_javascript :: proc(t: ^testing.T) {
	html := "<!doctype html><html><head><title>Next App</title><script type='application/ld+json'>{\"@context\":\"https://schema.org\",\"@type\":\"Product\",\"name\":\"Odin\"}</script><script id='__NEXT_DATA__' type='application/json'>{\"props\":{\"pageProps\":{\"count\":3}}}</script><script id='__NUXT_DATA__' type='application/json'>[null, {\"ok\":true}]</script><script>document.write('Injected script')</script></head><body><h1>Produto</h1></body></html>"
	p := parse_html(html, "https://example.com")
	testing.expect(t, len(p.hydration.json_ld) == 1)
	testing.expect(t, strings.contains(p.hydration.json_ld[0], "\"@type\":\"Product\""))
	testing.expect(t, strings.contains(p.hydration.next_data, "\"count\":3"))
	testing.expect(t, strings.contains(p.hydration.nuxt_data, "\"ok\":true"))
	testing.expect(t, !strings.contains(p.semantic.text, "Injected script"))
}

@(test)
test_hydration_rejects_invalid_json :: proc(t: ^testing.T) {
	p := parse_html("<script type='application/ld+json'>{invalid json}</script>", "https://example.com")
	testing.expect(t, len(p.hydration.json_ld) == 0)
}
