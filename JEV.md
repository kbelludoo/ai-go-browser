# Jev no AI Go Browser

Proposta de integração opcional, baseada na documentação oficial consultada em 10/10/2026. A integração ainda não está implementada nem foi feita chamada autenticada.

Jev recebe um estado e perguntas tipadas e devolve decisões estruturadas. Ele pode servir como seletor semântico das ações encontradas pelo navegador. A TypeSafe esclarece que ele não gera código nem texto livre. [Documentação](https://docs.typesafe.ai/introduction/coding-agents).

## Primeiro recurso proposto

Um comando `suggest` receberia o objetivo do agente e devolveria uma sugestão de ação, sem executá-la:

```json
{"cmd":"suggest","value":"Abrir a documentação de instalação"}
```

O Go coletaria as ações de `Discover()`, filtraria as permitidas e montaria um estado curto com título, trecho relevante e labels. Jev escolheria uma das referências existentes, ou `none`. O código validaria a referência contra a mesma lista, mantendo separadas a confiança numérica do modelo e `Action.Confidence`, que representa a origem da evidência (`declared`/`inferred`).

Para outra etapa, `Noul` poderia avaliar se a página contém o resultado pedido. `Score` poderia ordenar candidatos por relevância. O primeiro piloto deve medir acerto, abstenção, latência e tokens em tarefas locais com resultado conhecido antes de automatizar execução.

## Contrato HTTP

O endpoint oficial é `POST https://api.typesafe.ai/v1/systemone`, autenticado por Bearer. Go pode chamá-lo com `net/http` e `encoding/json`, sem adicionar Node ou Python. Uma pergunta `Choice` aceita até 255 opções. [Referência da API](https://docs.typesafe.ai/api).

Exemplo ilustrativo, com referências de uma página de teste:

```json
{
  "model": "jev-latest",
  "state": {
    "goal": "Abrir instruções de instalação",
    "page_title": "Documentação",
    "actions": [
      {"ref": "a1", "label": "Instalação", "method": "GET"},
      {"ref": "a2", "label": "Histórico de versões", "method": "GET"}
    ]
  },
  "questions": {
    "next_action": {
      "type": "choice",
      "instructions": "Escolha a ação que abre as instruções pedidas. Se nenhuma atender ao objetivo, escolha none.",
      "criteria": {
        "a1": "Abrir a seção Instalação",
        "a2": "Abrir o Histórico de versões",
        "none": "Nenhuma ação atende ao objetivo"
      }
    }
  }
}
```

A resposta contém `answers.next_action.choice`, `confidence` e `probabilities`. A confiança deriva da distribuição de opções; um valor alto não garante correção nem concede autorização para ações. [Confiança](https://docs.typesafe.ai/confidence).

## Desempenho e configuração

A inferência hospedada permite usar Jev sem carregar o modelo na RAM local. Toda chamada acrescenta comunicação com a API; o ganho potencial seria reduzir decisões enviadas a um LLM maior, e deve ser medido. A integração não reduz diretamente memória do DOM ou do Chromium.

O cliente usaria `TYPESAFE_API_KEY` no ambiente, endpoint oficial fixo, timeout e limites para corpo de requisição/resposta. A chave não iria nos snapshots ou logs. O estado enviado conteria apenas os dados necessários à decisão, omitindo valores de campos ocultos/senhas e outros segredos.

As regras atuais de hosts, `Executable`, `RequiresConfirmation` e `AllowPost` continuariam sendo verificadas pelo código. Em caso de falha da API, referência inválida ou resultado incerto, o navegador seguiria disponível para comandos explícitos.

Jev tem limitações conhecidas com contexto irrelevante e conteúdo adversarial; perguntas curtas e seleção entre ações já identificadas são um encaixe melhor que pedir planejamento amplo. [Limitações oficiais](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
