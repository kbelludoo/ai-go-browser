import React, { useState } from 'react';
import { Globe, ArrowRight, RefreshCw, Tag, AlertTriangle, Layers, ExternalLink, Play } from 'lucide-react';
import { Page, Command } from '../types/browser';
import { SAMPLE_PAGES, SamplePageDefinition } from '../engine/samplePages';

interface PageViewerProps {
  page: Page | null;
  currentHTML: string;
  onExecute: (cmd: Command) => void;
  onSelectSample: (sample: SamplePageDefinition) => void;
  activeSampleId?: string;
}

export const PageViewer: React.FC<PageViewerProps> = ({
  page,
  currentHTML,
  onExecute,
  onSelectSample,
  activeSampleId,
}) => {
  const [urlInput, setUrlInput] = useState(page?.url || 'http://127.0.0.1:33137/');
  const [withJS, setWithJS] = useState(false);
  const [showRefBadges, setShowRefBadges] = useState(true);
  const [formInputValues, setFormInputValues] = useState<Record<string, string>>({ q: 'pesquisa semântica' });

  const handleNavigate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!urlInput.trim()) return;
    onExecute({ cmd: 'open', url: urlInput, js: withJS });
  };

  const handleInputChange = (fieldRef: string, val: string) => {
    setFormInputValues((prev) => ({ ...prev, [fieldRef]: val }));
    onExecute({ cmd: 'fill', ref: fieldRef, value: val });
  };

  if (!page) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-slate-400">
        <Globe className="w-12 h-12 text-slate-600 mb-3 animate-pulse" />
        <p className="text-base font-medium">Nenhuma página aberta no motor.</p>
        <p className="text-xs text-slate-500 mt-1">Selecione um cenário de teste acima ou digite uma URL.</p>
      </div>
    );
  }

  const stack = page.stack;

  return (
    <div className="space-y-4">
      {/* Navigation & Address Bar */}
      <div className="bg-slate-900 p-3 rounded-xl border border-slate-800 shadow-md space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          {/* Preset Scenario Selector */}
          <div className="flex items-center gap-1.5 overflow-x-auto text-xs pb-1 sm:pb-0">
            <span className="text-slate-400 font-medium whitespace-nowrap">Cenários:</span>
            {SAMPLE_PAGES.map((s) => (
              <button
                key={s.id}
                onClick={() => {
                  setUrlInput(s.url);
                  onSelectSample(s);
                }}
                className={`px-2.5 py-1 rounded text-xs font-medium whitespace-nowrap transition-colors ${
                  activeSampleId === s.id
                    ? 'bg-cyan-600 text-white font-semibold'
                    : 'bg-slate-800 hover:bg-slate-700 text-slate-300'
                }`}
              >
                {s.name.split(' (')[0]}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-2">
            <label className="flex items-center gap-1.5 text-xs text-slate-300 cursor-pointer">
              <input
                type="checkbox"
                checked={withJS}
                onChange={(e) => setWithJS(e.target.checked)}
                className="rounded border-slate-700 text-cyan-500 focus:ring-cyan-500 bg-slate-950"
              />
              <span>Ativar JS / Headless</span>
            </label>

            <button
              onClick={() => setShowRefBadges(!showRefBadges)}
              className={`px-2 py-1 rounded text-xs font-mono border transition-colors ${
                showRefBadges
                  ? 'bg-cyan-950/80 text-cyan-300 border-cyan-800'
                  : 'bg-slate-800 text-slate-400 border-slate-700'
              }`}
            >
              Tags de Ref: {showRefBadges ? 'VISÍVEIS' : 'OCULTAS'}
            </button>
          </div>
        </div>

        {/* Address Input */}
        <form onSubmit={handleNavigate} className="flex items-center gap-2">
          <div className="relative flex-1">
            <Globe className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={urlInput}
              onChange={(e) => setUrlInput(e.target.value)}
              placeholder="Digite a URL (ex: https://example.com ou http://127.0.0.1:33137/)"
              className="w-full pl-9 pr-4 py-2 bg-slate-950 border border-slate-700 rounded-lg text-slate-100 text-sm font-mono focus:outline-none focus:border-cyan-500"
            />
          </div>
          <button
            type="submit"
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white font-medium rounded-lg text-sm flex items-center gap-1.5 transition-colors"
          >
            <span>Carregar</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </form>

        {/* Diagnostics & Tech Stack Header */}
        {stack && (
          <div className="pt-2 border-t border-slate-800/80 flex flex-wrap items-center justify-between gap-2 text-xs">
            <div className="flex flex-wrap items-center gap-2">
              <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-200 border border-slate-700 font-mono">
                {stack.framework}
              </span>
              <span
                className={`px-2 py-0.5 rounded border font-mono ${
                  stack.renderMode.includes('CSR')
                    ? 'bg-amber-950/60 text-amber-300 border-amber-800/60'
                    : 'bg-emerald-950/60 text-emerald-300 border-emerald-800/60'
                }`}
              >
                {stack.renderMode}
              </span>
              <span className="text-slate-400 font-mono">
                Encoding: <strong className="text-slate-200">{stack.encoding.toUpperCase()}</strong>
              </span>
              <span className="text-slate-400 font-mono">
                Rendimento Semântico: <strong className="text-cyan-400">{stack.contentYield}%</strong>
              </span>
            </div>

            <div className="text-slate-500 font-mono text-[11px]">
              {page.bytes} bytes brutos • {page.text.length} chars de texto
            </div>
          </div>
        )}

        {/* Notice Warning if CSR or Low Yield */}
        {stack?.notice && (
          <div className="p-2.5 rounded bg-amber-950/40 border border-amber-800/50 flex items-start gap-2 text-xs text-amber-200">
            <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold">{stack.notice}</p>
              <button
                onClick={() => onExecute({ cmd: 'render', url: page.url })}
                className="mt-1 text-cyan-400 underline hover:text-cyan-300 flex items-center gap-1 font-mono text-[11px]"
              >
                <Play className="w-3 h-3" />
                <span>Acionar Camada 3: Motor Headless Efêmero (`render`)</span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Rendered Sandbox Viewport */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Main Rendered DOM Canvas */}
        <div className="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg space-y-4">
          <div className="border-b border-slate-800 pb-3 flex items-center justify-between">
            <div>
              <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                <span>{page.title}</span>
                <span className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
                  HTTP {page.status}
                </span>
              </h2>
              {page.description && (
                <p className="text-xs text-slate-400 mt-1">{page.description}</p>
              )}
            </div>
          </div>

          {/* Interactive Visual DOM Sandbox */}
          <div className="bg-slate-950 p-6 rounded-lg border border-slate-800/80 space-y-5 text-slate-200 min-h-[300px]">
            {/* Dynamic Buttons */}
            {page.buttons.length > 0 && (
              <div className="space-y-2">
                <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                  Botões Interativos:
                </div>
                <div className="flex flex-wrap gap-2.5">
                  {page.buttons.map((b) => (
                    <div key={b.ref} className="relative inline-flex items-center">
                      <button
                        onClick={() => onExecute({ cmd: 'press', ref: b.ref })}
                        className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-lg font-medium text-sm border border-slate-700 hover:border-cyan-500 shadow-sm transition-all active:scale-95 flex items-center gap-2"
                      >
                        <span>{b.label}</span>
                        {showRefBadges && (
                          <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-cyan-900 text-cyan-300 font-bold border border-cyan-700">
                            [{b.ref}]
                          </span>
                        )}
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Dynamic Forms */}
            {page.forms.length > 0 && (
              <div className="space-y-3 pt-3 border-t border-slate-800/60">
                <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
                  <span>Formulários ({page.forms.length}):</span>
                </div>
                {page.forms.map((f) => (
                  <form
                    key={f.ref}
                    onSubmit={(e) => {
                      e.preventDefault();
                      onExecute({ cmd: 'submit', ref: f.ref, confirm: true });
                    }}
                    className="p-4 rounded-lg bg-slate-900/90 border border-slate-800 space-y-3"
                  >
                    <div className="flex items-center justify-between text-xs text-slate-400 font-mono">
                      <span>Ação: {f.action}</span>
                      <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">
                        {f.method} {showRefBadges && `[${f.ref}]`}
                      </span>
                    </div>

                    <div className="space-y-2">
                      {f.fields.map((field, idx) => {
                        const fieldRef = `i${idx + 1}`;
                        if (field.type === 'hidden') return null;
                        return (
                          <div key={field.name} className="space-y-1">
                            <label className="text-xs font-medium text-slate-300 flex items-center justify-between">
                              <span>Campo: {field.name}</span>
                              {showRefBadges && (
                                <span className="text-[10px] font-mono text-cyan-400">[{fieldRef}]</span>
                              )}
                            </label>
                            <input
                              type={field.type}
                              value={formInputValues[field.name] ?? field.value ?? ''}
                              onChange={(e) => handleInputChange(field.name, e.target.value)}
                              placeholder={`Preencha ${field.name}...`}
                              className="w-full px-3 py-1.5 bg-slate-950 border border-slate-700 rounded text-slate-100 text-sm font-mono focus:border-cyan-500 focus:outline-none"
                            />
                          </div>
                        );
                      })}
                    </div>

                    <button
                      type="submit"
                      className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded font-medium text-sm transition-colors flex items-center gap-2"
                    >
                      <span>{f.submit_label || 'Enviar'}</span>
                      {showRefBadges && (
                        <span className="text-[10px] font-mono bg-cyan-900 px-1 rounded text-cyan-200">
                          [submit]
                        </span>
                      )}
                    </button>
                  </form>
                ))}
              </div>
            )}

            {/* Extracted Clean Text Preview */}
            <div className="pt-3 border-t border-slate-800/60 space-y-2">
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                Conteúdo Semântico Normalizado:
              </div>
              <div className="p-4 rounded bg-slate-900/60 text-sm text-slate-300 leading-relaxed font-sans max-h-48 overflow-y-auto whitespace-pre-wrap">
                {page.text || 'Nenhum texto visível extraído.'}
              </div>
            </div>
          </div>
        </div>

        {/* Sidebar: Navigation Links & Headings Tree */}
        <div className="space-y-4">
          {/* Discovered Links */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
            <h3 className="text-sm font-semibold text-slate-200 flex items-center justify-between">
              <span>Links Navegáveis ({page.links.length})</span>
              <span className="text-xs font-mono text-slate-400">click &lt;ref&gt;</span>
            </h3>

            <div className="space-y-1.5 max-h-60 overflow-y-auto pr-1">
              {page.links.map((link) => (
                <button
                  key={link.ref}
                  onClick={() => onExecute({ cmd: 'click', ref: link.ref })}
                  className="w-full text-left p-2 rounded bg-slate-950/70 hover:bg-slate-800 border border-slate-800/80 hover:border-cyan-500/50 transition-colors flex items-start justify-between gap-2 text-xs"
                >
                  <div className="truncate">
                    <div className="font-medium text-slate-200 truncate">{link.label}</div>
                    <div className="text-slate-500 font-mono text-[11px] truncate">{link.url}</div>
                  </div>
                  {showRefBadges && (
                    <span className="shrink-0 font-mono text-[10px] px-1.5 py-0.5 rounded bg-cyan-950 text-cyan-400 border border-cyan-800">
                      [{link.ref}]
                    </span>
                  )}
                </button>
              ))}
              {page.links.length === 0 && (
                <p className="text-xs text-slate-500 italic">Nenhum link com href na página.</p>
              )}
            </div>
          </div>

          {/* Section Headings Hierarchy */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
            <h3 className="text-sm font-semibold text-slate-200 flex items-center gap-2">
              <Layers className="w-4 h-4 text-cyan-400" />
              <span>Hierarquia de Seções ({page.headings.length})</span>
            </h3>

            <div className="space-y-1 max-h-48 overflow-y-auto text-xs">
              {page.headings.map((h, i) => (
                <div
                  key={i}
                  className="flex items-center gap-2 py-1 px-2 rounded hover:bg-slate-800 text-slate-300 font-mono"
                  style={{ paddingLeft: `${(h.level - 1) * 12 + 8}px` }}
                >
                  <span className="text-cyan-400 font-bold">H{h.level}</span>
                  <span className="truncate">{h.text}</span>
                </div>
              ))}
              {page.headings.length === 0 && (
                <p className="text-xs text-slate-500 italic">Nenhum cabeçalho H1-H6 detectado.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
