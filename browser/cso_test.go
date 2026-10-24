package browser

import (
	"testing"
)

func TestCSOTokenReduction(t *testing.T) {
	// Sample real-world web page markdown with links, headers, boilerplate, and tables
	sampleMD := `# Portal de Serviços Públicos - Governo Digital

[Pular para o conteúdo principal](#main)
[Menu de Acessibilidade](/acessibilidade)
[Política de Privacidade e Cookies](/privacidade?utm_source=header&utm_medium=banner)

## Serviços Mais Acessados

* [Emitir Certidão de Regularidade Fiscal](https://receita.economia.gov.br/servicos/certidao?ano=2026&origem=portal) - Emissão gratuita e imediata para pessoas físicas e jurídicas.
* [Consultar Situação Cadastral de CPF](https://receita.economia.gov.br/cpf/consulta?status=ativo) - Verifique pendências cadastrais na base da Receita.
* [Agendamento de Atendimento Presencial](https://receita.economia.gov.br/agendamento) - Marque data e horário nas agências regionais.

| Serviço | Prazo | Custo |
|---|---|---|
| Certidão Negativa | Imediato | Gratuito |
| Alteração Cadastral | Até 48h | Gratuito |
| Regularização de Débitos | Até 5 dias | Conforme cálculo |

Todos os direitos reservados. Governo Federal © 2026.
[Voltar ao topo](#top)
`

	tokensMD := EstimateTokens(sampleMD)
	bytesMD := len(sampleMD)

	cso := MarkdownToCSO(sampleMD, 10000)
	tokensCSO := EstimateTokens(cso)
	bytesCSO := len(cso)

	reductionPct := (1.0 - float64(tokensCSO)/float64(tokensMD)) * 100

	t.Logf("📊 [RESULTADO EMPÍRICO REAL]")
	t.Logf("   Markdown: %d bytes | Estimativa de Tokens: %d", bytesMD, tokensMD)
	t.Logf("   CSO:      %d bytes | Estimativa de Tokens: %d", bytesCSO, tokensCSO)
	t.Logf("   ✨ Redução adicional de tokens alcançada pelo CSO: %.2f%%", reductionPct)
	t.Logf("   --- Conteúdo CSO Gerado ---\n%s\n---------------------------", cso)

	if reductionPct < 30.0 {
		t.Fatalf("esperava reducao de tokens >= 30%%, obteve %.2f%%", reductionPct)
	}
}
