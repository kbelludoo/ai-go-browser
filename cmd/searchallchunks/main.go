package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	chunks := []string{
		"1_qx_g6zeg7_0.js", "3__ofeua7_zyl.js", "3btkt--fh555f.js", "3qzmlu00gsb_e.js", "1zovzp2st7wa1.js",
		"2ayblcy6gm81v.js", "3rji3z109tjn3.js", "0w060liy5v9fx.js", "1b-0ttdah3ci-.js", "18ylitmwgezh3.js",
		"26hfqqpg3cr4n.js", "3kep5f-0ykbl4.js", "3edshwmdh3ay9.js", "3k-k7yv1dt1p_.js", "0k0lbycqn0i0a.js",
		"113acmyrswe1l.js", "28vem-jel0hka.js", "01sd_vphrxyqf.js", "0cfollrbhsyfm.js", "1un2z8b1wpwc3.js",
		"0cc5v2x25a2s4.js", "2bcmdb-_9bylw.js", "1zn5ilzbq8fd9.js", "2dqy23hij1vds.js", "01nf7dtxyi-xm.js",
		"10j_n_24-it2h.js", "0wjxwuj0bj9x7.js", "3ug37nyd4wzw1.js", "1xa2paqtzs_47.js", "0q3a2-ayfr8ms.js",
		"3ka-dadvvcrgc.js", "0mf_pc4xd63tf.js", "1g76ddcswkf2g.js", "41alpg4hsifqr.js", "2kl74_o40r6wd.js",
		"0bxvdpgnwrowg.js", "2k0-k5wwdblwa.js", "09fi_u5do687m.js", "32a_dgkycyory.js", "2ane-4b86-60w.js",
		"2bzvow9g6km_v.js", "0yb3_szurluaw.js", "19_96o_0duy-g.js", "2ij1cdcd4st84.js", "0cz1d0mv5g_q7.js",
		"0_d4m8e70c2rw.js", "2ovd9ml5m0q6-.js", "01dtb792wrq6b.js", "0f_w8ucqarieh.js",
	}

	for _, c := range chunks {
		url := "https://arena.ai/_next/static/chunks/" + c + "?dpl=dpl_5utUdh1kn7oYYXe1bxQeq4wahtbE"
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		txt := string(body)

		if strings.Contains(txt, "Built for complex tasks") {
			fmt.Printf("MATCH in %s:\n", c)
			idx := strings.Index(txt, "Built for complex tasks")
			start := max(0, idx-300)
			end := min(len(txt), idx+300)
			fmt.Println(txt[start:end])
			return
		}
	}
	fmt.Println("Not found in chunks")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
