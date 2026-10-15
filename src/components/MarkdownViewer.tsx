import React, { useState } from 'react';
import { Page } from '../types/browser';
import { FileText, Copy, Check, Hash, Sparkles } from 'lucide-react';

interface MarkdownViewerProps {
  page: Page | null;
}

export const MarkdownViewer: React.FC<MarkdownViewerProps> = ({ page }) => {
  const [viewMode, setViewMode] = useState<'markdown' | 'text'>('markdown');
  const [copied, setCopied] = useState(false);

  if (!page) {
    return (
      <div className="p-8 text-center text-slate-500">
        Nenhuma página carregada para extração de Markdown.
      </div>
    );
  }

  const content = viewMode === 'markdown' ? (page.markdown || page.text) : page.text;
  const rawBytes = page.bytes || content.length * 3;
  const rawTokens = Math.ceil(rawBytes / 4);
  const mdTokens = Math.ceil(content.length / 4);
  const savings = rawTokens > 0 ? Math.round((1 - mdTokens / rawTokens) * 1000) / 10 : 0;

  const copyContent = () => {
    navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-4">
      {/* Metrics Banner */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-cyan-400" />
              <span>Digest Semântico de Baixo Custo para LLMs</span>
            </h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Converte HTML poluído de dezenas de kilobytes em Markdown limpo e conciso, economizando até 95% de tokens de contexto.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => setViewMode('markdown')}
              className={`px-3 py-1 rounded text-xs font-mono font-medium transition-colors ${
                viewMode === 'markdown'
                  ? 'bg-cyan-600 text-white'
                  : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              Markdown
            </button>
            <button
              onClick={() => setViewMode('text')}
              className={`px-3 py-1 rounded text-xs font-mono font-medium transition-colors ${
                viewMode === 'text'
                  ? 'bg-cyan-600 text-white'
                  : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              Texto Puro
            </button>
            <button
              onClick={copyContent}
              className="px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded text-xs font-mono flex items-center gap-1.5 transition-colors border border-slate-700"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied ? 'Copiado!' : 'Copiar'}</span>
            </button>
          </div>
        </div>

        {/* Token Savings Metric Bar */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-2 border-t border-slate-800/80 text-xs font-mono">
          <div className="p-2.5 rounded bg-slate-950 border border-slate-800/80">
            <div className="text-slate-500 text-[11px]">Tokens HTML Bruto</div>
            <div className="text-slate-300 font-bold text-sm mt-0.5">~{rawTokens.toLocaleString()}</div>
          </div>
          <div className="p-2.5 rounded bg-slate-950 border border-slate-800/80">
            <div className="text-slate-500 text-[11px]">Tokens Digest AI</div>
            <div className="text-cyan-400 font-bold text-sm mt-0.5">~{mdTokens.toLocaleString()}</div>
          </div>
          <div className="p-2.5 rounded bg-slate-950 border border-slate-800/80">
            <div className="text-slate-500 text-[11px]">Economia de Contexto</div>
            <div className="text-emerald-400 font-bold text-sm mt-0.5">{savings}% menor</div>
          </div>
          <div className="p-2.5 rounded bg-slate-950 border border-slate-800/80">
            <div className="text-slate-500 text-[11px]">Tamanho Caracteres</div>
            <div className="text-slate-300 font-bold text-sm mt-0.5">{content.length} chars</div>
          </div>
        </div>
      </div>

      {/* Content Editor / Viewer */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-lg">
        <div className="bg-slate-950 px-4 py-2 border-b border-slate-800 flex items-center justify-between text-xs font-mono text-slate-400">
          <span>{viewMode === 'markdown' ? 'digest.md' : 'content.txt'}</span>
          <span>{content.split('\n').length} linhas</span>
        </div>

        <div className="p-4 bg-slate-950 font-mono text-xs text-slate-200 leading-relaxed overflow-x-auto max-h-[500px]">
          <pre className="whitespace-pre-wrap font-mono">{content}</pre>
        </div>
      </div>
    </div>
  );
};
