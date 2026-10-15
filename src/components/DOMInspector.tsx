import React, { useState } from 'react';
import { Page, ElementMatch } from '../types/browser';
import { Layers, Search, Code, CheckCircle, Database } from 'lucide-react';

interface DOMInspectorProps {
  page: Page | null;
  onQuery: (selector: string) => Promise<ElementMatch[]>;
  lastMatches: ElementMatch[] | null;
}

export const DOMInspector: React.FC<DOMInspectorProps> = ({ page, onQuery, lastMatches }) => {
  const [selectorInput, setSelectorInput] = useState('button, form input');
  const [queryResults, setQueryResults] = useState<ElementMatch[] | null>(lastMatches);
  const [isQuerying, setIsQuerying] = useState(false);
  const [activeMetaTab, setActiveMetaTab] = useState<'next' | 'jsonld' | 'og'>('next');

  const handleRunQuery = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectorInput.trim()) return;
    setIsQuerying(true);
    try {
      const res = await onQuery(selectorInput);
      setQueryResults(res);
    } catch (err: any) {
      console.error(err);
    } finally {
      setIsQuerying(false);
    }
  };

  const metadata = page?.metadata || {};
  const hasNextData = Boolean(metadata['next_data']);
  const hasJsonLd = Boolean(metadata['json_ld']);
  const hasOg = Boolean(metadata['open_graph']);

  return (
    <div className="space-y-4">
      {/* Selector Query Section */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-bold text-slate-100 flex items-center gap-2">
              <Search className="w-4 h-4 text-cyan-400" />
              <span>Consulta CSS Reduzida (`query &lt;selector&gt;`)</span>
            </h3>
            <p className="text-xs text-slate-400 mt-0.5">
              Retorna apenas nós correspondentes sem inundar a janela de contexto do LLM com o snapshot inteiro da página.
            </p>
          </div>

          <div className="flex gap-1.5 text-xs font-mono">
            {['button', '#estado', 'form input', 'h1, h2', 'a[href]'].map((sample) => (
              <button
                key={sample}
                onClick={() => {
                  setSelectorInput(sample);
                  onQuery(sample).then(setQueryResults);
                }}
                className="px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition-colors"
              >
                {sample}
              </button>
            ))}
          </div>
        </div>

        <form onSubmit={handleRunQuery} className="flex items-center gap-2">
          <div className="relative flex-1">
            <Code className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={selectorInput}
              onChange={(e) => setSelectorInput(e.target.value)}
              placeholder="Digite o seletor CSS (ex: form input[name=q], #main, button)..."
              className="w-full pl-9 pr-4 py-2 bg-slate-950 border border-slate-700 rounded-lg text-slate-100 text-xs font-mono focus:border-cyan-500 focus:outline-none"
            />
          </div>
          <button
            type="submit"
            disabled={isQuerying}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white font-medium rounded-lg text-xs transition-colors"
          >
            {isQuerying ? 'Consultando...' : 'Buscar Nós'}
          </button>
        </form>

        {/* Query Matches Table */}
        {queryResults && (
          <div className="pt-2 border-t border-slate-800/80 space-y-2">
            <div className="text-xs font-mono text-slate-400 flex items-center justify-between">
              <span>{queryResults.length} nós encontrados (limite máx: 32):</span>
            </div>

            <div className="bg-slate-950 rounded-lg border border-slate-800 overflow-x-auto max-h-60">
              <table className="w-full text-left text-xs font-mono">
                <thead className="bg-slate-900 text-slate-400 border-b border-slate-800 text-[11px]">
                  <tr>
                    <th className="py-2 px-3">Tag</th>
                    <th className="py-2 px-3">ID</th>
                    <th className="py-2 px-3">Texto / Valor</th>
                    <th className="py-2 px-3">Atributos Relevantes</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60 text-slate-300">
                  {queryResults.map((m, idx) => (
                    <tr key={idx} className="hover:bg-slate-900/40">
                      <td className="py-2 px-3 text-cyan-400 font-bold">&lt;{m.tag}&gt;</td>
                      <td className="py-2 px-3 text-slate-400">{m.id ? `#${m.id}` : '-'}</td>
                      <td className="py-2 px-3 text-slate-200 truncate max-w-xs">{m.text || m.value || '-'}</td>
                      <td className="py-2 px-3 text-slate-500 truncate max-w-xs">
                        {m.href ? `href="${m.href}" ` : ''}
                        {m.name ? `name="${m.name}" ` : ''}
                        {m.role ? `role="${m.role}"` : ''}
                      </td>
                    </tr>
                  ))}
                  {queryResults.length === 0 && (
                    <tr>
                      <td colSpan={4} className="py-4 text-center text-slate-500 italic">
                        Nenhum elemento encontrado para o seletor.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Layer 2: Structured Data & Hydration Payloads */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-800 pb-3">
          <div>
            <h3 className="text-sm font-bold text-slate-100 flex items-center gap-2">
              <Database className="w-4 h-4 text-cyan-400" />
              <span>Camada 2: Extração de Dados Estruturados & Hydration</span>
            </h3>
            <p className="text-xs text-slate-400 mt-0.5">
              Extrai esquemas schema.org, __NEXT_DATA__ e OpenGraph de SPAs sem necessidade de executar engine pesada de layout.
            </p>
          </div>

          <div className="flex gap-1.5 text-xs font-mono">
            <button
              onClick={() => setActiveMetaTab('next')}
              className={`px-2.5 py-1 rounded transition-colors ${
                activeMetaTab === 'next'
                  ? 'bg-cyan-600 text-white font-semibold'
                  : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              __NEXT_DATA__ {hasNextData && '●'}
            </button>
            <button
              onClick={() => setActiveMetaTab('jsonld')}
              className={`px-2.5 py-1 rounded transition-colors ${
                activeMetaTab === 'jsonld'
                  ? 'bg-cyan-600 text-white font-semibold'
                  : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              JSON-LD {hasJsonLd && '●'}
            </button>
            <button
              onClick={() => setActiveMetaTab('og')}
              className={`px-2.5 py-1 rounded transition-colors ${
                activeMetaTab === 'og'
                  ? 'bg-cyan-600 text-white font-semibold'
                  : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              OpenGraph {hasOg && '●'}
            </button>
          </div>
        </div>

        {/* Payload Display */}
        <div className="bg-slate-950 p-4 rounded-lg border border-slate-800 font-mono text-xs max-h-72 overflow-y-auto">
          {activeMetaTab === 'next' && (
            hasNextData ? (
              <pre className="text-cyan-300 whitespace-pre-wrap">
                {JSON.stringify(metadata['next_data'], null, 2)}
              </pre>
            ) : (
              <p className="text-slate-500 italic">
                Nenhum script de hidratação `&lt;script id="__NEXT_DATA__"&gt;` encontrado nesta página.
                (Carregue o cenário "Next.js App with Hydration Payload" para testar).
              </p>
            )
          )}

          {activeMetaTab === 'jsonld' && (
            hasJsonLd ? (
              <pre className="text-emerald-300 whitespace-pre-wrap">
                {JSON.stringify(metadata['json_ld'], null, 2)}
              </pre>
            ) : (
              <p className="text-slate-500 italic">
                Nenhum script `&lt;script type="application/ld+json"&gt;` encontrado.
              </p>
            )
          )}

          {activeMetaTab === 'og' && (
            hasOg ? (
              <pre className="text-amber-300 whitespace-pre-wrap">
                {JSON.stringify(metadata['open_graph'], null, 2)}
              </pre>
            ) : (
              <p className="text-slate-500 italic">
                Nenhuma tag meta OpenGraph `og:*` encontrada na página.
              </p>
            )
          )}
        </div>
      </div>
    </div>
  );
};
