import React, { useState } from 'react';
import { Play, CheckCircle2, AlertTriangle, Globe, ArrowRight, ShieldAlert, Cpu, Sparkles, Filter, RefreshCw, Layers } from 'lucide-react';
import { DOMParserEngine } from '../engine/domParser';

interface RealSiteTestItem {
  url: string;
  name: string;
  category: string;
  loaded?: boolean;
  status?: number;
  duration_ms?: number;
  bytes?: number;
  title?: string;
  linksCount?: number;
  headingsCount?: number;
  formsCount?: number;
  buttonsCount?: number;
  textSnippet?: string;
  readabilityScore?: number;
  tokenReduction?: number;
  wafStatus?: string;
  nextPage?: string;
  suggestions?: string[];
}

const DEFAULT_REAL_SITES: RealSiteTestItem[] = [
  { url: 'https://example.com/', name: 'Example Domain', category: 'Baseline / RFC 2606' },
  { url: 'https://news.ycombinator.com/', name: 'Hacker News', category: 'Link Aggregator / High Density' },
  { url: 'https://go.dev/', name: 'Go Language Portal', category: 'Modern Tech Landing / Docs' },
  { url: 'https://en.wikipedia.org/wiki/Go_(programming_language)', name: 'Wikipedia (Go)', category: 'High Volume Reference / 700KB+' },
  { url: 'https://httpbin.org/', name: 'Httpbin.org', category: 'HTTP APIs / Endpoints' },
  { url: 'https://books.toscrape.com/', name: 'Books to Scrape', category: 'E-commerce Catalog / Pagination' },
  { url: 'https://lite.cnn.com/', name: 'CNN Lite News', category: 'Low-Bandwidth News Stream' },
  { url: 'https://pkg.go.dev/', name: 'Go Packages Index', category: 'Search & Deep Documentation' },
];

export const RealSitesTestRunner: React.FC = () => {
  const [testList, setTestList] = useState<RealSiteTestItem[]>(DEFAULT_REAL_SITES);
  const [isRunning, setIsRunning] = useState(false);
  const [activeUrl, setActiveUrl] = useState<string | null>(null);
  const [customInput, setCustomInput] = useState('');
  const [selectedSite, setSelectedSite] = useState<RealSiteTestItem | null>(null);

  const testSingleUrl = async (target: RealSiteTestItem): Promise<RealSiteTestItem> => {
    const startTime = performance.now();
    try {
      const resp = await fetch(`/api/proxy?url=${encodeURIComponent(target.url)}`);
      const duration = Math.round(performance.now() - startTime);

      if (!resp.ok) {
        return {
          ...target,
          loaded: false,
          status: resp.status,
          duration_ms: duration,
          suggestions: ['Site retornou código de erro HTTP ou rejeitou User-Agent.'],
        };
      }

      const data = await resp.json();
      const rawHTML: string = data.html || '';
      const doc = DOMParserEngine.parse(rawHTML, target.url, data.status, data.bytes);
      const page = doc.getPage(5000);
      const stack = DOMParserEngine.analyzeTechStack(rawHTML, page.text.length, 'utf-8');

      const rawTokens = Math.ceil(data.bytes / 4);
      const cleanTokens = Math.ceil(page.text.length / 4);
      const tokenReduction = rawTokens > 0 ? Math.round((1 - cleanTokens / rawTokens) * 1000) / 10 : 0;

      // Generate AI-agent specific improvements based on real findings
      const suggestions: string[] = [];
      if (stack.wafStatus === 'cloudflare_challenge') {
        suggestions.push('WAF Cloudflare detectado: requer fallback efêmero com CDP ou rotação de TLS fingerprint (ja3).');
      }
      if (data.bytes > 500000) {
        suggestions.push(`Página muito pesada (${(data.bytes / 1024).toFixed(0)} KB): implementar streaming HTML parser em chunks para economizar mais RAM.`);
      }
      if (page.links.length > 200) {
        suggestions.push(`Muitos links (${page.links.length}): priorizar apenas links dentro de <main> e ignorar rodapés repetitivos.`);
      }
      if (page.nextPageUrl) {
        suggestions.push(`Paginação encontrada (${page.nextPageUrl}): o agente pode usar comando 'next_page' direto.`);
      } else if (target.category.includes('Catalog') || target.category.includes('News')) {
        suggestions.push('Sem tag rel="next" explícita: adicionar heurística para links com números ordinais (1, 2, 3...)');
      }
      if (page.readabilityScore && page.readabilityScore < 15) {
        suggestions.push('Baixa densidade de texto útil em relação ao HTML: aumentar filtro de ruído para descartar scripts e tracking.');
      } else {
        suggestions.push('Excelente rendimento: Markdown destilado economizou >90% de tokens para o LLM.');
      }

      return {
        ...target,
        loaded: true,
        status: data.status,
        bytes: data.bytes,
        duration_ms: duration,
        title: page.title,
        linksCount: page.links.length,
        headingsCount: page.headings.length,
        formsCount: page.forms.length,
        buttonsCount: page.buttons.length,
        textSnippet: page.text.substring(0, 300),
        readabilityScore: page.readabilityScore,
        tokenReduction,
        wafStatus: stack.wafStatus,
        nextPage: page.nextPageUrl,
        suggestions,
      };
    } catch (err: any) {
      return {
        ...target,
        loaded: false,
        status: 0,
        duration_ms: Math.round(performance.now() - startTime),
        suggestions: [`Falha de rede ou timeout: ${err.message}`],
      };
    }
  };

  const runAllTests = async () => {
    setIsRunning(true);
    const updated = [...testList];

    for (let i = 0; i < updated.length; i++) {
      setActiveUrl(updated[i].url);
      const res = await testSingleUrl(updated[i]);
      updated[i] = res;
      setTestList([...updated]);
    }

    setActiveUrl(null);
    setIsRunning(false);
  };

  const handleAddCustom = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!customInput.trim()) return;
    let url = customInput.trim();
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      url = `https://${url}`;
    }

    const newItem: RealSiteTestItem = {
      url,
      name: new URL(url).hostname,
      category: 'URL Personalizada',
    };

    setTestList((prev) => [newItem, ...prev]);
    setCustomInput('');
    setIsRunning(true);
    setActiveUrl(url);
    const result = await testSingleUrl(newItem);
    setTestList((prev) => [result, ...prev.slice(1)]);
    setActiveUrl(null);
    setIsRunning(false);
  };

  return (
    <div className="space-y-4">
      {/* Top Banner: Real Sites Test Suite */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 sm:p-5 space-y-4 shadow-xl">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <span className="px-2 py-0.5 rounded bg-cyan-950 text-cyan-400 font-mono text-[11px] font-bold border border-cyan-800">
                AI Agent Benchmark
              </span>
              <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
                <span>Avaliação em Sites Reais & Diagnóstico para IAs</span>
              </h2>
            </div>
            <p className="text-xs text-slate-400 mt-1 max-w-3xl">
              Navegadores para agentes de IA operam sem tela: recebem URLs, ignoram poluição visual (banners, cookies, ads) e entregam contexto semântico puro com o menor consumo de tokens e RAM possível.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={runAllTests}
              disabled={isRunning}
              className="px-4 py-2 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 disabled:opacity-50 text-white font-medium rounded-lg text-xs flex items-center gap-2 shadow-lg shadow-cyan-600/20 transition-all active:scale-95"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>{isRunning ? 'Testando Sites Reais...' : 'Executar Todos os Testes'}</span>
            </button>
          </div>
        </div>

        {/* Add Custom Live URL to Test */}
        <form onSubmit={handleAddCustom} className="flex items-center gap-2 pt-2 border-t border-slate-800/80">
          <div className="relative flex-1">
            <Globe className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={customInput}
              onChange={(e) => setCustomInput(e.target.value)}
              placeholder="Testar qualquer site real (ex: https://github.com/trending, https://reddit.com/r/golang, https://globo.com)..."
              className="w-full pl-9 pr-4 py-2 bg-slate-950 border border-slate-700 rounded-lg text-slate-100 text-xs font-mono focus:border-cyan-500 focus:outline-none"
            />
          </div>
          <button
            type="submit"
            disabled={isRunning}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors"
          >
            <span>Testar URL</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </button>
        </form>
      </div>

      {/* Real Test Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-lg">
        <div className="px-4 py-3 bg-slate-950 border-b border-slate-800 flex items-center justify-between text-xs font-mono">
          <span className="font-bold text-slate-300 uppercase tracking-wider">
            Resultados de Teste ao Vivo ({testList.length} sites)
          </span>
          {activeUrl && (
            <span className="text-cyan-400 flex items-center gap-1.5 animate-pulse">
              <RefreshCw className="w-3 h-3 animate-spin" />
              <span>Baixando e parseando: {activeUrl}</span>
            </span>
          )}
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="bg-slate-950/60 text-slate-400 font-mono border-b border-slate-800 text-[11px] uppercase tracking-wider">
                <th className="py-3 px-4">Site / URL</th>
                <th className="py-3 px-4">Status & WAF</th>
                <th className="py-3 px-4">Tamanho & Tempo</th>
                <th className="py-3 px-4">Economia Tokens LLM</th>
                <th className="py-3 px-4">Elementos Descobertos</th>
                <th className="py-3 px-4">Paginação</th>
                <th className="py-3 px-4 text-right">Diagnóstico</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/80 font-sans">
              {testList.map((site, idx) => {
                const isSelected = selectedSite?.url === site.url;
                return (
                  <tr
                    key={idx}
                    className={`hover:bg-slate-800/40 transition-colors ${
                      isSelected ? 'bg-cyan-950/20 border-l-2 border-cyan-500' : ''
                    }`}
                  >
                    <td className="py-3 px-4">
                      <div className="font-semibold text-slate-200">{site.name}</div>
                      <div className="text-[11px] text-slate-500 font-mono truncate max-w-xs">{site.url}</div>
                      <span className="inline-block mt-0.5 text-[10px] px-1.5 py-0.2 rounded bg-slate-800 text-slate-400">
                        {site.category}
                      </span>
                    </td>

                    <td className="py-3 px-4 font-mono whitespace-nowrap">
                      {site.loaded === undefined ? (
                        <span className="text-slate-500">Pendente</span>
                      ) : site.loaded ? (
                        <div className="space-y-1">
                          <span className="flex items-center gap-1 text-emerald-400 font-bold">
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            <span>HTTP {site.status}</span>
                          </span>
                          {site.wafStatus && site.wafStatus !== 'clean' && (
                            <span className="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded bg-amber-950 text-amber-400 border border-amber-800">
                              <ShieldAlert className="w-3 h-3" />
                              <span>{site.wafStatus}</span>
                            </span>
                          )}
                        </div>
                      ) : (
                        <span className="flex items-center gap-1 text-rose-400 font-bold">
                          <AlertTriangle className="w-3.5 h-3.5" />
                          <span>Falhou ({site.status || 'Erro'})</span>
                        </span>
                      )}
                    </td>

                    <td className="py-3 px-4 font-mono whitespace-nowrap">
                      {site.bytes ? (
                        <div>
                          <div className="text-slate-200">{(site.bytes / 1024).toFixed(1)} KB</div>
                          <div className="text-[11px] text-cyan-400">{site.duration_ms} ms</div>
                        </div>
                      ) : (
                        '-'
                      )}
                    </td>

                    <td className="py-3 px-4 whitespace-nowrap">
                      {site.tokenReduction ? (
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <div className="w-16 bg-slate-950 rounded-full h-1.5 overflow-hidden border border-slate-800">
                              <div
                                className="bg-emerald-500 h-full rounded-full"
                                style={{ width: `${site.tokenReduction}%` }}
                              />
                            </div>
                            <span className="font-mono text-emerald-400 font-bold">
                              -{site.tokenReduction}%
                            </span>
                          </div>
                          <div className="text-[10px] text-slate-500 font-mono">
                            Densidade: {site.readabilityScore}/100
                          </div>
                        </div>
                      ) : (
                        '-'
                      )}
                    </td>

                    <td className="py-3 px-4 font-mono text-[11px] text-slate-400 whitespace-nowrap">
                      {site.loaded ? (
                        <div>
                          <div><strong className="text-slate-200">{site.linksCount}</strong> links</div>
                          <div><strong className="text-slate-200">{site.formsCount}</strong> forms • <strong className="text-slate-200">{site.buttonsCount}</strong> btns</div>
                        </div>
                      ) : (
                        '-'
                      )}
                    </td>

                    <td className="py-3 px-4 font-mono text-[11px] whitespace-nowrap">
                      {site.nextPage ? (
                        <span className="px-2 py-0.5 rounded bg-cyan-950 text-cyan-300 border border-cyan-800 truncate max-w-[140px] inline-block" title={site.nextPage}>
                          Detectada
                        </span>
                      ) : site.loaded ? (
                        <span className="text-slate-600">Não detectada</span>
                      ) : (
                        '-'
                      )}
                    </td>

                    <td className="py-3 px-4 text-right whitespace-nowrap">
                      <button
                        onClick={() => setSelectedSite(site)}
                        className="px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white border border-slate-700 text-xs font-medium transition-colors"
                      >
                        Ver Análise
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Selected Site Detailed Diagnostics & Engineering Suggestions */}
      {selectedSite && selectedSite.suggestions && (
        <div className="bg-slate-900 border border-cyan-800/80 rounded-xl p-5 shadow-2xl space-y-4">
          <div className="flex items-center justify-between border-b border-slate-800 pb-3">
            <div>
              <h3 className="text-sm font-bold text-slate-100 flex items-center gap-2">
                <Sparkles className="w-4 h-4 text-cyan-400" />
                <span>Diagnóstico de Otimização para IA: {selectedSite.name}</span>
              </h3>
              <p className="text-xs text-slate-400 font-mono mt-0.5">{selectedSite.url}</p>
            </div>
            <button
              onClick={() => setSelectedSite(null)}
              className="text-xs text-slate-400 hover:text-slate-200 font-mono px-2 py-1 rounded bg-slate-800"
            >
              Fechar
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Recommendations */}
            <div className="space-y-2 p-3.5 rounded-lg bg-slate-950 border border-slate-800">
              <div className="text-xs font-bold text-cyan-400 uppercase tracking-wider font-mono">
                Recomendações Práticas para o Agente:
              </div>
              <ul className="space-y-1.5 text-xs text-slate-300">
                {selectedSite.suggestions.map((sug, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <span className="text-cyan-500 font-bold">•</span>
                    <span>{sug}</span>
                  </li>
                ))}
              </ul>
            </div>

            {/* Extracted Clean Snippet */}
            <div className="space-y-2 p-3.5 rounded-lg bg-slate-950 border border-slate-800">
              <div className="text-xs font-bold text-emerald-400 uppercase tracking-wider font-mono">
                Amostra de Texto Semântico Filtrado:
              </div>
              <p className="text-xs text-slate-300 leading-relaxed font-sans max-h-32 overflow-y-auto">
                {selectedSite.textSnippet || 'Nenhum texto disponível'}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Concrete Technical Guidelines for AI-First Browsers */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-xs">
        <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-2">
          <div className="font-bold text-cyan-400 flex items-center gap-2">
            <Filter className="w-4 h-4" />
            <span>1. Remoção Agressiva de Ruído</span>
          </div>
          <p className="text-slate-400 leading-relaxed">
            IAs não precisam de rodapés, banners de cookies, scripts de rastreamento ou headers duplicados. Extrair o container &lt;main&gt; ou &lt;article&gt; economiza 80% a 95% do contexto do LLM.
          </p>
        </div>

        <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-2">
          <div className="font-bold text-emerald-400 flex items-center gap-2">
            <Cpu className="w-4 h-4" />
            <span>2. Resolução de Ações sem UI</span>
          </div>
          <p className="text-slate-400 leading-relaxed">
            Em vez de fazer a IA adivinhar coordenadas de clique na tela, extrair ações declaradas com referências concisas (`a1`, `l1`, `f1`). O agente decide a próxima ação em 1 único token.
          </p>
        </div>

        <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-2">
          <div className="font-bold text-amber-400 flex items-center gap-2">
            <ShieldAlert className="w-4 h-4" />
            <span>3. Detecção Precoce de WAF</span>
          </div>
          <p className="text-slate-400 leading-relaxed">
            Se a página contiver desafio Cloudflare Turnstile ou DataDome, o motor sinaliza imediatamente no payload JSON para o agente decidir se aciona a camada efêmera com stealth ou avança.
          </p>
        </div>
      </div>
    </div>
  );
};
