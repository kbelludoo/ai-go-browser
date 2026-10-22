package browser

import (
	"strings"
	"testing"
)

func TestExtractHydrationAndButtonAction(t *testing.T) {
	sampleHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Awesome AI Bot Store</title>
    <!-- Schema.org Product Metadata -->
    <script type="application/ld+json">
    {
        "@context": "https://schema.org/",
        "@type": "Product",
        "name": "Autonomous Go Navigator",
        "image": "https://example.com/bot.png",
        "description": "Ultra-lightweight Go web agent with zero Chromium footprint",
        "sku": "GO-AGENT-007",
        "offers": {
            "@type": "Offer",
            "priceCurrency": "USD",
            "price": "99.00",
            "availability": "https://schema.org/InStock"
        }
    }
    </script>

    <!-- Next.js Hydration State -->
    <script id="__NEXT_DATA__" type="application/json">
    {
        "props": {
            "pageProps": {
                "initialProduct": {
                    "id": "prod_9981",
                    "stock": 42,
                    "secretApiRoute": "/api/v2/checkout/express"
                }
            }
        }
    }
    </script>
</head>
<body>
    <h1>Autonomous Go Navigator</h1>
    <p>Get the best autonomous navigation engine.</p>

    <!-- Form with dynamic button and hidden CSRF token -->
    <form action="/cart/add" method="POST">
        <input type="hidden" name="csrf_token" value="sec_token_xyz999">
        <input type="hidden" name="product_id" value="prod_9981">
        <button type="submit"
                id="btn-buy"
                data-action="add-to-cart"
                data-sku="GO-AGENT-007"
                data-price="99.00"
                data-endpoint="/api/v2/cart/instant"
                onclick="trackPurchase('GO-AGENT-007')">
            Comprar Agora
        </button>
    </form>
</body>
</html>
`

	page := Parse([]byte(sampleHTML), "https://store.example.com", 200, len(sampleHTML), 100000)

	// 1. Verify JSON-LD Schema.org Extraction (Zero-JS)
	if page.Hydration == nil {
		t.Fatalf("expected hydration payload, got nil")
	}
	if len(page.Hydration.JSONLD) == 0 {
		t.Fatalf("expected JSON-LD entities, got 0")
	}
	prod := page.Hydration.JSONLD[0]
	if prod["name"] != "Autonomous Go Navigator" || prod["sku"] != "GO-AGENT-007" {
		t.Errorf("unexpected product entity: %+v", prod)
	}

	// 2. Verify Next.js State Extraction (Zero-JS)
	if page.Hydration.NextData == nil {
		t.Fatalf("expected Next.js state, got nil")
	}
	initialProd, ok := page.Hydration.NextData["initialProduct"].(map[string]any)
	if !ok || initialProd["id"] != "prod_9981" || initialProd["secretApiRoute"] != "/api/v2/checkout/express" {
		t.Errorf("unexpected Next.js pageProps: %+v", page.Hydration.NextData)
	}

	// 3. Verify Button Action Metadata Extraction (Zero-JS)
	if len(page.Buttons) == 0 {
		t.Fatalf("expected at least 1 button, got 0")
	}
	btn := page.Buttons[0]
	if btn.Action == nil {
		t.Fatalf("expected button action metadata, got nil")
	}

	// Verify action deduced without clicking or running JS
	if !strings.HasSuffix(btn.Action.FormAction, "/cart/add") || btn.Action.FormMethod != "POST" {
		t.Errorf("expected form action POST /cart/add, got %s %s", btn.Action.FormMethod, btn.Action.FormAction)
	}
	if btn.Action.HiddenInputs["csrf_token"] != "sec_token_xyz999" {
		t.Errorf("expected csrf_token to be captured from form, got %v", btn.Action.HiddenInputs)
	}
	if btn.Action.DataAttrs["data-sku"] != "GO-AGENT-007" || btn.Action.DataAttrs["data-price"] != "99.00" {
		t.Errorf("expected data attributes to be captured, got %v", btn.Action.DataAttrs)
	}

	summary := btn.Action.ActionSummary
	if !strings.Contains(summary, "POST https://store.example.com/cart/add") || !strings.Contains(summary, "sku=GO-AGENT-007") {
		t.Errorf("unexpected ActionSummary: %s", summary)
	}
}

// BenchmarkZeroJSHydrationExtraction proves that static extraction takes microseconds with 0 VM overhead.
func BenchmarkZeroJSHydrationExtraction(b *testing.B) {
	htmlData := `
<!DOCTYPE html><html><head>
<script type="application/ld+json">{"@type": "Product", "name": "Item", "offers": {"price": "19.99"}}</script>
<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"data":[1,2,3,4,5]}}}</script>
</head><body>
<form action="/api/submit" method="POST">
<button type="submit" data-action="submit" data-id="100">Send</button>
</form>
</body></html>`

	raw := []byte(htmlData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := Parse(raw, "https://example.com", 200, len(raw), 100000)
		if p.Hydration == nil {
			b.Fatal("missing hydration")
		}
	}
}
