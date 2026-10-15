export interface SamplePageDefinition {
  id: string;
  url: string;
  name: string;
  description: string;
  category: 'demo' | 'benchmark' | 'integration' | 'spa';
  html: string;
  mockApiEndpoints?: Record<string, { status: number; contentType: string; data: any }>;
}

export const SAMPLE_PAGES: SamplePageDefinition[] = [
  {
    id: 'demo-local',
    name: 'AI Browser Interactive Demo (v0.4/v0.5)',
    url: 'http://127.0.0.1:33137/',
    description: 'Original Go test harness with JS stateful counter, dynamic link injection, search form, and fetch API endpoint.',
    category: 'demo',
    html: `<!doctype html>
<html>
<head>
  <title>AI Browser Demo</title>
  <meta name="description" content="Navegação e ações para agente IA universal">
  <link rel="alternate" type="application/json" href="/api/produtos">
  <style>.x {color:red}</style>
</head>
<body>
  <main>
    <p id="estado">Sem JavaScript</p>
    <button id="contar" type="button">Contar</button>
    <button id="produtos" type="button">Ver produtos</button>
    <h1>Página inicial</h1>
    <p>Navegação para agente IA de alta performance.</p>
    <a href="/artigo"><span>Ler</span> artigo</a>
    <form action="/busca" method="GET">
      <label for="q">Busca</label>
      <input id="q" name="q" aria-label="Pesquisa" placeholder="Digite sua pesquisa...">
      <input type="hidden" name="csrf_token" value="secret_session_token_xyz987">
      <button type="submit">Pesquisar</button>
    </form>
    <script>
      // Persistent JS state
      document.getElementById('estado').textContent = 'JavaScript ativo';
      var n = 0;
      document.getElementById('contar').addEventListener('click', function() {
        n++;
        document.getElementById('estado').textContent = 'Cliques: ' + n;
      });
      document.getElementById('produtos').addEventListener('click', function() {
        fetch('/api/produtos');
      });
    </script>
  </main>
</body>
</html>`,
    mockApiEndpoints: {
      '/api/produtos': {
        status: 200,
        contentType: 'application/json',
        data: {
          products: [
            { id: 1, name: 'Caderno Universitário', price: 12.5 },
            { id: 2, name: 'Caneta Esferográfica Gel', price: 2.9 },
            { id: 3, name: 'Mochila Ergonômica', price: 89.9 }
          ],
          api_token: '[redacted]',
          server_timestamp: '2026-10-14T18:00:00Z'
        }
      },
      '/artigo': {
        status: 200,
        contentType: 'text/html',
        data: `<!doctype html><html><head><title>Artigo Local</title></head><body><h1>Artigo de Pesquisa Semântica</h1><p>Funcionou com motor híbrido sem sobrecarga de RAM.</p><a href="/">Voltar</a></body></html>`
      },
      '/busca': {
        status: 200,
        contentType: 'text/html',
        data: `<!doctype html><html><head><title>Resultados de Busca</title></head><body><h1>Resultado da Pesquisa</h1><p>Encontrados 4 resultados para o agente.</p><a href="/">Voltar à Home</a></body></html>`
      }
    }
  },
  {
    id: 'php-catalog',
    name: 'Multi-Language Server & HTMX Actions Suite',
    url: 'https://demo-site.local/products.php',
    description: 'Demonstrates language-independent discovery: PHP route, HTMX hx-get, nonbutton elements, danger confirmation check.',
    category: 'integration',
    html: `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="windows-1252">
  <title>Catálogo de Produtos - PHP/HTMX</title>
  <link rel="alternate" type="application/json" href="/api/catalog.json">
</head>
<body>
  <header>
    <h1>Mercado Integrado PHP</h1>
    <p>Endpoints expostos e ações de formulário e HTMX.</p>
  </header>
  <nav>
    <a href="/products.php?category=eletronicos">Eletrônicos</a>
    <a href="/products.php?category=livros">Livros & Revistas</a>
  </nav>
  <section hx-get="/api/feed.php" aria-controls="feed-container">
    <h3>Feed Dinâmico de Ofertas</h3>
    <div id="feed-container">Clique para carregar feed via HTMX</div>
  </section>
  <div class="card">
    <h4>Item 101: Monitor UltraWide 34</h4>
    <p>Preço: R$ 2.499,00</p>
    <button hx-post="/cart/add" data-url="/cart/add" hx-vals='{"item": 101}'>Adicionar ao Carrinho</button>
    <button class="danger-btn" hx-delete="/admin/products/101">Excluir Produto (Perigoso)</button>
  </div>
  <form action="/checkout.php" method="POST">
    <input type="hidden" name="token" value="auth_bearer_9999">
    <label>Cupom de Desconto:</label>
    <input type="text" name="coupon" value="PROMO2026">
    <button type="submit">Finalizar Pedido</button>
  </form>
  <script>
    // Axios candidate endpoints in client bundle
    function checkStock(id) {
      axios.get('/api/inventory/check?id=' + id);
    }
    fetch('/api/analytics/view');
  </script>
</body>
</html>`,
    mockApiEndpoints: {
      '/api/catalog.json': {
        status: 200,
        contentType: 'application/json',
        data: {
          catalog: 'V2',
          items: [
            { id: 101, name: 'Monitor UltraWide 34', price: 2499.0 },
            { id: 102, name: 'Teclado Mecânico RGB', price: 450.0 }
          ]
        }
      },
      '/api/feed.php': {
        status: 200,
        contentType: 'application/json',
        data: { deals: ['Frete Grátis acima de R$100', '10% de cashback em PIX'] }
      },
      '/api/analytics/view': {
        status: 200,
        contentType: 'application/json',
        data: { recorded: true, session: 'agent_crawler_v0.7' }
      }
    }
  },
  {
    id: 'nextjs-hydration',
    name: 'Next.js App with Hydration Payload & JSON-LD',
    url: 'https://news-portal.example/articles/ai-agent-breakthrough',
    description: 'Layer 2 Structured Data extraction: extracts __NEXT_DATA__ and schema.org JSON-LD without firing heavy browser JS.',
    category: 'spa',
    html: `<!doctype html>
<html>
<head>
  <title>AI Go Browser v0.7 Released: Semantic Web Crawler</title>
  <meta name="description" content="Universal hybrid browser delivers structured semantic data to LLMs under 25MB RAM.">
  <script type="application/ld+json">
  {
    "@context": "https://schema.org",
    "@type": "NewsArticle",
    "headline": "AI Go Browser v0.7 Released: Semantic Web Crawler",
    "datePublished": "2026-10-14T10:00:00Z",
    "author": {
      "@type": "Person",
      "name": "Kbelludoo"
    },
    "publisher": {
      "@type": "Organization",
      "name": "AI Systems Lab"
    }
  }
  </script>
  <script id="__NEXT_DATA__" type="application/json">
  {
    "props": {
      "pageProps": {
        "articleId": "art_2026_09",
        "title": "AI Go Browser v0.7 Released",
        "category": "Artificial Intelligence",
        "readingTimeMinutes": 4,
        "tags": ["Golang", "Headless Browser", "AI Agent", "Token Reduction", "LLM"]
      }
    },
    "page": "/articles/[slug]",
    "query": { "slug": "ai-agent-breakthrough" },
    "buildId": "prod-build-882319"
  }
  </script>
</head>
<body>
  <div id="__next">
    <header>
      <nav>
        <a href="/">Notícias</a>
        <a href="/benchmarks">Benchmarks</a>
      </nav>
    </header>
    <main>
      <h1>AI Go Browser v0.7 Released: Semantic Web Crawler</h1>
      <p>Um navegador headless de alta performance projetado para agentes de IA e automação semântica, mantendo consumo estrito abaixo de 300 MB de RAM.</p>
      <h2>Arquitetura Híbrida de 3 Camadas</h2>
      <p>Combina motor Go puro para 80%+ da web, extrator de dados estruturados com hidratação, e motor headless efêmero sob demanda.</p>
      <form action="/newsletter/subscribe" method="POST">
        <label>Receba novidades:</label>
        <input name="email" type="email" placeholder="seu@email.com">
        <button type="submit">Inscrever</button>
      </form>
    </main>
  </div>
</body>
</html>`
  },
  {
    id: 'tokenbench-wikipedia',
    name: 'Wikipedia (Go Programming Language)',
    url: 'https://en.wikipedia.org/wiki/Go_(programming_language)',
    description: 'High token reduction target: dense HTML with thousands of nodes converted into clean, concise Markdown.',
    category: 'benchmark',
    html: `<!doctype html>
<html>
<head>
  <title>Go (programming language) - Wikipedia</title>
</head>
<body>
  <div id="content">
    <h1 id="firstHeading">Go (programming language)</h1>
    <div id="bodyContent">
      <p><b>Go</b> is a statically typed, compiled high-level programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson.</p>
      <p>It is syntactically similar to C, but also has memory safety, garbage collection, structural typing, and CSP-style concurrency.</p>
      <h2>Key Features</h2>
      <p>Go was designed to improve programming productivity in an era of multicore, networked machines and large codebases.</p>
      <ul>
        <li>Static typing and run-time efficiency (like C)</li>
        <li>Readability and usability (like Python)</li>
        <li>High-performance concurrency mechanisms (goroutines and channels)</li>
      </ul>
      <h2>Adoption in AI Agent Infrastructure</h2>
      <p>Modern autonomous agents utilize lightweight Go runtimes to navigate web applications with minimal RAM footprint and sub-millisecond execution loops.</p>
      <a href="https://go.dev">Official Go Website</a>
      <a href="https://golang.org/doc/">Go Documentation</a>
    </div>
  </div>
</body>
</html>`
  },
  {
    id: 'tokenbench-hackernews',
    name: 'Hacker News (Y Combinator)',
    url: 'https://news.ycombinator.com',
    description: 'Benchmark target with numerous vote links, discussion threads, and submissions.',
    category: 'benchmark',
    html: `<!doctype html>
<html>
<head><title>Hacker News</title></head>
<body>
  <center>
    <table border="0" cellpadding="0" cellspacing="0" width="85%" bgcolor="#f6f6ef">
      <tr>
        <td bgcolor="#ff6600">
          <table border="0" cellpadding="0" cellspacing="2">
            <tr>
              <td><a href="https://news.ycombinator.com"><b class="hnname">Hacker News</b></a></td>
              <td><a href="newest">new</a> | <a href="front">past</a> | <a href="newcomments">comments</a> | <a href="ask">ask</a> | <a href="show">show</a> | <a href="jobs">jobs</a> | <a href="submit">submit</a></td>
            </tr>
          </table>
        </td>
      </tr>
      <tr>
        <td>
          <ol>
            <li>
              <a href="https://github.com/kbelludoo/ai-go-browser">AI Go Browser v0.7: Universal Hybrid Browser for AI Agents</a>
              <span class="subtext">142 points by kbelludoo 3 hours ago | <a href="item?id=101">48 comments</a></span>
            </li>
            <li>
              <a href="https://lmsys.org">LMSYS Chatbot Arena Leaderboard Updates</a>
              <span class="subtext">280 points by ml_researcher 5 hours ago | <a href="item?id=102">95 comments</a></span>
            </li>
            <li>
              <a href="https://developer.mozilla.org">Modern Web Performance in 2026</a>
              <span class="subtext">95 points by dev_guru 7 hours ago | <a href="item?id=103">31 comments</a></span>
            </li>
          </ol>
        </td>
      </tr>
    </table>
  </center>
</body>
</html>`
  }
];
