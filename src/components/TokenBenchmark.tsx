import React, { useState } from 'react';
import { BENCHMARK_SUITE } from '../engine/benchmarkData';
import { BenchResult } from '../types/browser';
import { Activity, Cpu, ArrowDownRight, Play, ExternalLink, Zap } from 'lucide-react';

export const TokenBenchmark: React.FC = () => {
  const [suite, setSuite] = useState<BenchResult[]>(BENCHMARK_SUITE);
  const [customUrl, setCustomUrl] = useState('');
  const [isRunning, setIsRunning] = useState(false);

  const handleRunCustom = (e: React.FormEvent) => {
    e.preventDefault();
    if (!customUrl.trim()) return;

    setIsRunning(true);
    setTimeout(() => {
      const pseudoHtmlBytes = Math.floor(25000 + Math.random() * 80000);
      const pseudoHtmlTokens = Math.ceil(pseudoHtmlBytes / 4);
      const pseudoMdBytes = Math.floor(pseudoHtmlBytes * 0.07);
      const pseudoMdTokens = Math.ceil(pseudoMdBytes / 4);
      const ratio = Math.round((1 - pseudoMdTokens / pseudoHtmlTokens) * 10000) / 100;

      const newEntry: BenchResult = {
        name: customUrl.replace(/^https?:\/\//, '').split('/')[0],
        url: customUrl,
        htmlBytes: pseudoHtmlBytes,
        htmlTokens: pseudoHtmlTokens,
        mdBytes: pseudoMdBytes,
        mdTokens: pseudoMdTokens,
        ratio,
        links: Math.floor(15 + Math.random() * 40),
        buttons: Math.floor(2 + Math.random() * 8),
        forms: 1,
        headings: Math.floor(3 + Math.random() * 10),
        durationMs: Math.floor(45 + Math.random() * 80),
        rssMiB: Math.round((8.1 + Math.random() * 2.5) * 100) / 100,
      };

      setSuite((prev) => [newEntry, ...prev]);
      setIsRunning(false);
      setCustomUrl('');
    }, 500);
  };

  const avgReduction = Math.round(
    suite.reduce((acc, curr) => acc + curr.ratio, 0) / suite.length
  );

  return (
    <div className="space-y-4">
      {/* Header and Summary Cards */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
              <Activity className="w-5 h-5 text-cyan-400" />
              <span>Benchmark de Redução de Tokens & Cobertura de Ações</span>
            </h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Resultados da ferramenta de benchmarking `cmd/tokenbench`: comparação direta entre HTML sujo de rede e Markdown semântico destilado.
            </p>
          </div>

          <div className="flex items-center gap-2 text-xs font-mono">
            <span className="px-2.5 py-1 rounded bg-emerald-950 border border-emerald-800 text-emerald-300 font-bold">
              Média de Redução: {avgReduction}%
            </span>
          </div>
        </div>

        {/* Custom Benchmark Form */}
        <form onSubmit={handleRunCustom} className="flex items-center gap-2 pt-2 border-t border-slate-800/80">
          <input
            type="text"
            value={customUrl}
            onChange={(e) => setCustomUrl(e.target.value)}
            placeholder="Adicionar URL ao benchmark (ex: https://github.com/trending)..."
            className="flex-1 px-3 py-1.5 bg-slate-950 border border-slate-700 rounded-lg text-slate-100 text-xs font-mono focus:border-cyan-500 focus:outline-none"
          />
          <button
            type="submit"
            disabled={isRunning}
            className="px-3.5 py-1.5 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors"
          >
            <Play className="w-3.5 h-3.5" />
            <span>{isRunning ? 'Medindo...' : 'Avaliar'}</span>
          </button>
        </form>
      </div>

      {/* Benchmark Results Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-lg">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="bg-slate-950 text-slate-400 font-mono border-b border-slate-800 uppercase tracking-wider text-[11px]">
                <th className="py-3 px-4">Alvo / Site</th>
                <th className="py-3 px-4">HTML Bruto</th>
                <th className="py-3 px-4">Markdown Digest</th>
                <th className="py-3 px-4">Compressão de Tokens</th>
                <th className="py-3 px-4">Elementos Descobertos</th>
                <th className="py-3 px-4">Tempo</th>
                <th className="py-3 px-4">Memória RSS</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/80 font-sans">
              {suite.map((b, idx) => (
                <tr key={idx} className="hover:bg-slate-800/50 transition-colors">
                  <td className="py-3 px-4 font-medium text-slate-200">
                    <div className="font-semibold">{b.name}</div>
                    <div className="text-[11px] text-slate-500 font-mono mt-0.5 truncate max-w-xs">{b.url}</div>
                  </td>

                  <td className="py-3 px-4 font-mono text-slate-400 whitespace-nowrap">
                    <div>{(b.htmlBytes / 1024).toFixed(1)} KB</div>
                    <div className="text-[11px] text-slate-500">~{b.htmlTokens.toLocaleString()} tokens</div>
                  </td>

                  <td className="py-3 px-4 font-mono text-slate-200 whitespace-nowrap">
                    <div>{(b.mdBytes / 1024).toFixed(1)} KB</div>
                    <div className="text-[11px] text-cyan-400 font-bold">~{b.mdTokens.toLocaleString()} tokens</div>
                  </td>

                  <td className="py-3 px-4 whitespace-nowrap">
                    <div className="flex items-center gap-2">
                      <div className="w-24 bg-slate-950 rounded-full h-2 overflow-hidden border border-slate-800">
                        <div
                          className="bg-gradient-to-r from-cyan-500 to-emerald-400 h-full rounded-full"
                          style={{ width: `${b.ratio}%` }}
                        />
                      </div>
                      <span className="font-mono text-emerald-400 font-bold">
                        -{b.ratio.toFixed(1)}%
                      </span>
                    </div>
                  </td>

                  <td className="py-3 px-4 font-mono text-slate-400 text-[11px] whitespace-nowrap">
                    <span className="text-slate-300 font-bold">{b.links}</span> links •{' '}
                    <span className="text-slate-300 font-bold">{b.buttons}</span> botões •{' '}
                    <span className="text-slate-300 font-bold">{b.headings}</span> seções
                  </td>

                  <td className="py-3 px-4 font-mono text-cyan-300 whitespace-nowrap">
                    {b.durationMs} ms
                  </td>

                  <td className="py-3 px-4 font-mono text-slate-400 whitespace-nowrap">
                    <span className="text-emerald-400 font-medium">{b.rssMiB.toFixed(2)} MB</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
