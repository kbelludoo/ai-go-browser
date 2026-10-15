import React, { useState } from 'react';
import { Action, Resource } from '../types/browser';
import { Zap, ShieldCheck, AlertCircle, Play, Info, Check, Search, Filter } from 'lucide-react';

interface ActionsTableProps {
  actions: Action[];
  onInspect: (ref: string) => void;
  onRunAction: (ref: string, confirm: boolean) => Promise<Resource | void>;
  selectedAction: Action | null;
  lastResource: Resource | null;
}

export const ActionsTable: React.FC<ActionsTableProps> = ({
  actions,
  onInspect,
  onRunAction,
  selectedAction,
  lastResource,
}) => {
  const [filterKind, setFilterKind] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [runningRef, setRunningRef] = useState<string | null>(null);

  const filtered = actions.filter((act) => {
    if (filterKind !== 'all' && act.kind !== filterKind) return false;
    if (searchTerm) {
      const q = searchTerm.toLowerCase();
      return (
        act.label.toLowerCase().includes(q) ||
        act.ref.toLowerCase().includes(q) ||
        (act.url && act.url.toLowerCase().includes(q)) ||
        act.source.toLowerCase().includes(q)
      );
    }
    return true;
  });

  const handleRun = async (act: Action) => {
    setRunningRef(act.ref);
    try {
      await onRunAction(act.ref, true);
    } catch (err: any) {
      console.error(err);
    } finally {
      setRunningRef(null);
    }
  };

  return (
    <div className="space-y-4">
      {/* Action Discovery Overview & Filters */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
              <Zap className="w-5 h-5 text-cyan-400" />
              <span>Resolução Estática e Conservadora de Ações (v0.5)</span>
            </h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Descobre endpoints de formulários, links, HTMX e literais JS sem carregar DOM de terceiros ou disparar requisições prematuras.
            </p>
          </div>

          <div className="flex items-center gap-2 text-xs font-mono">
            <span className="px-2.5 py-1 rounded bg-slate-950 border border-slate-800 text-slate-300">
              Total: <strong className="text-cyan-400">{actions.length}</strong> ações
            </span>
          </div>
        </div>

        {/* Filter & Search Bar */}
        <div className="flex flex-wrap items-center gap-3 pt-2 border-t border-slate-800/80">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Filtrar por label, ref (a1), url ou fonte..."
              className="w-full pl-9 pr-3 py-1.5 bg-slate-950 border border-slate-700/80 rounded-lg text-slate-200 text-xs focus:outline-none focus:border-cyan-500"
            />
          </div>

          <div className="flex items-center gap-1.5 text-xs">
            <Filter className="w-3.5 h-3.5 text-slate-400" />
            <span className="text-slate-400">Tipo:</span>
            {['all', 'api', 'form', 'link', 'button'].map((k) => (
              <button
                key={k}
                onClick={() => setFilterKind(k)}
                className={`px-2.5 py-1 rounded text-xs font-mono capitalize transition-colors ${
                  filterKind === k
                    ? 'bg-cyan-600 text-white font-medium'
                    : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
                }`}
              >
                {k === 'all' ? 'Todos' : k}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Discovered Actions Grid / Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-lg">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="bg-slate-950/80 text-slate-400 font-mono border-b border-slate-800 uppercase tracking-wider text-[11px]">
                <th className="py-3 px-4">Ref</th>
                <th className="py-3 px-4">Ação / Rótulo</th>
                <th className="py-3 px-4">Tipo</th>
                <th className="py-3 px-4">Método</th>
                <th className="py-3 px-4">Alvo / URL</th>
                <th className="py-3 px-4">Confiança</th>
                <th className="py-3 px-4">Segurança</th>
                <th className="py-3 px-4 text-right">Executar</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/80 font-sans">
              {filtered.map((act) => {
                const isSelected = selectedAction?.ref === act.ref;
                return (
                  <tr
                    key={act.ref}
                    className={`hover:bg-slate-800/60 transition-colors ${
                      isSelected ? 'bg-cyan-950/30 border-l-2 border-cyan-500' : ''
                    }`}
                  >
                    <td className="py-3 px-4 font-mono font-bold text-cyan-400 whitespace-nowrap">
                      {act.ref}
                    </td>

                    <td className="py-3 px-4 font-medium text-slate-200">
                      <div>{act.label}</div>
                      <div className="text-[11px] text-slate-500 font-mono mt-0.5">{act.source}</div>
                    </td>

                    <td className="py-3 px-4 whitespace-nowrap">
                      <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-slate-800 text-slate-300 border border-slate-700">
                        {act.kind}
                      </span>
                    </td>

                    <td className="py-3 px-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-0.5 rounded text-[11px] font-mono font-bold ${
                          act.method === 'GET'
                            ? 'bg-emerald-950 text-emerald-400 border border-emerald-800/60'
                            : act.method === 'POST'
                            ? 'bg-blue-950 text-blue-400 border border-blue-800/60'
                            : 'bg-rose-950 text-rose-400 border border-rose-800/60'
                        }`}
                      >
                        {act.method}
                      </span>
                    </td>

                    <td className="py-3 px-4 font-mono text-slate-300 max-w-xs truncate" title={act.url}>
                      {act.url || '-'}
                    </td>

                    <td className="py-3 px-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-0.5 rounded text-[11px] font-mono ${
                          act.confidence === 'declared'
                            ? 'bg-cyan-950 text-cyan-300 border border-cyan-800'
                            : 'bg-amber-950 text-amber-300 border border-amber-800'
                        }`}
                      >
                        {act.confidence}
                      </span>
                    </td>

                    <td className="py-3 px-4 whitespace-nowrap">
                      {act.requires_confirmation ? (
                        <span className="flex items-center gap-1 text-[11px] text-amber-400 font-mono">
                          <AlertCircle className="w-3.5 h-3.5" />
                          <span>Exige confirmação</span>
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-[11px] text-emerald-400 font-mono">
                          <ShieldCheck className="w-3.5 h-3.5" />
                          <span>Seguro (GET)</span>
                        </span>
                      )}
                    </td>

                    <td className="py-3 px-4 text-right whitespace-nowrap">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => onInspect(act.ref)}
                          title="Inspecionar evidência e detalhes da ação"
                          className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 flex items-center gap-1 text-[11px]"
                        >
                          <Info className="w-3 h-3" />
                          <span>Inspecionar</span>
                        </button>

                        <button
                          onClick={() => handleRun(act)}
                          disabled={runningRef === act.ref || !act.executable}
                          title={act.executable ? 'Executar ação' : 'Ação requer parâmetros extras'}
                          className="px-2.5 py-1 rounded bg-cyan-600 hover:bg-cyan-500 disabled:opacity-40 disabled:hover:bg-cyan-600 text-white font-medium flex items-center gap-1 text-[11px] transition-colors"
                        >
                          <Play className="w-3 h-3" />
                          <span>{runningRef === act.ref ? 'Executando...' : 'Run'}</span>
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {filtered.length === 0 && (
                <tr>
                  <td colSpan={8} className="py-8 text-center text-slate-500 italic">
                    Nenhuma ação encontrada para os filtros atuais.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Action Inspection Drawer */}
      {selectedAction && (
        <div className="bg-slate-900 border border-cyan-800/80 rounded-xl p-4 space-y-3 shadow-xl">
          <div className="flex items-center justify-between border-b border-slate-800 pb-2">
            <h3 className="text-sm font-bold text-slate-100 flex items-center gap-2">
              <span className="font-mono text-cyan-400">[{selectedAction.ref}]</span>
              <span>Auditoria da Ação: {selectedAction.label}</span>
            </h3>
            <span className="text-xs font-mono text-slate-400">Evidência determinística</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs font-mono">
            <div className="space-y-1.5 p-3 rounded bg-slate-950 border border-slate-800">
              <div className="text-slate-400">Fonte no Código:</div>
              <div className="text-slate-200 font-bold">{selectedAction.source}</div>
              <div className="text-slate-400 mt-2">Evidência Extraída:</div>
              <div className="text-cyan-300">{selectedAction.evidence || 'N/A'}</div>
            </div>

            <div className="space-y-1.5 p-3 rounded bg-slate-950 border border-slate-800">
              <div className="text-slate-400">Alvo Reconstruído:</div>
              <div className="text-slate-200 truncate">{selectedAction.preview || selectedAction.url}</div>
              <div className="text-slate-400 mt-2">Garantia de Segurança:</div>
              <div className="text-amber-300">
                {selectedAction.requires_confirmation
                  ? 'Requer confirm: true antes da chamada de rede.'
                  : 'Requisição idempotente e segura para agentes autônomos.'}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Last Executed Resource Payload Viewer */}
      {lastResource && (
        <div className="bg-slate-900 border border-emerald-800/80 rounded-xl p-4 space-y-2.5 shadow-xl">
          <div className="flex items-center justify-between border-b border-slate-800 pb-2">
            <h3 className="text-sm font-bold text-emerald-400 flex items-center gap-2">
              <Check className="w-4 h-4" />
              <span>Resposta da Ação Executada (Resource)</span>
            </h3>
            <span className="text-xs font-mono text-slate-400">
              HTTP {lastResource.status} • {lastResource.content_type}
            </span>
          </div>

          <div className="p-3 rounded bg-slate-950 border border-slate-800 font-mono text-xs text-slate-300 overflow-x-auto">
            <div className="text-slate-500 mb-1">URL: {lastResource.url}</div>
            <pre className="text-emerald-300 whitespace-pre-wrap">
              {JSON.stringify(lastResource.data || lastResource.body, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
};
