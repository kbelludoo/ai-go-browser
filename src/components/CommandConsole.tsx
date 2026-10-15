import React, { useState } from 'react';
import { Terminal, Send, HelpCircle, CheckCircle, AlertTriangle, ChevronDown, ChevronRight, Copy } from 'lucide-react';
import { Command, Result } from '../types/browser';

interface CommandConsoleProps {
  onExecute: (cmd: Command) => Promise<Result>;
  lastResult: Result | null;
}

export const CommandConsole: React.FC<CommandConsoleProps> = ({ onExecute, lastResult }) => {
  const [inputVal, setInputVal] = useState('discover');
  const [isJsonlMode, setIsJsonlMode] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);
  const [isPending, setIsPending] = useState(false);
  const [history, setHistory] = useState<string[]>(['open http://127.0.0.1:33137/', 'discover']);
  const [historyIdx, setHistoryIdx] = useState(-1);
  const [copied, setCopied] = useState(false);

  const parseToCommand = (line: string): Command => {
    line = line.trim();
    if (line.startsWith('{')) {
      return JSON.parse(line);
    }
    const parts = line.split(/\s+/);
    const cmd = parts[0].toLowerCase();
    switch (cmd) {
      case 'open':
        return {
          cmd: 'open',
          url: parts[1],
          js: parts.length > 2 && (parts[2] === 'js' || parts[2] === '--js'),
        };
      case 'render':
        return { cmd: 'render', url: parts[1] };
      case 'click':
        return { cmd: 'click', ref: parts[1] };
      case 'press':
        return { cmd: 'press', ref: parts[1] };
      case 'fill':
        return { cmd: 'fill', ref: parts[1], value: parts.slice(2).join(' ') };
      case 'submit':
        return { cmd: 'submit', ref: parts[1], confirm: true };
      case 'query':
        return { cmd: 'query', selector: parts.slice(1).join(' ') };
      case 'inspect':
        return { cmd: 'inspect', ref: parts[1] };
      case 'run':
        return { cmd: 'run', ref: parts[1], confirm: true };
      case 'snapshot':
      case 'discover':
      case 'back':
      case 'markdown':
      case 'text':
      case 'help':
      case 'quit':
      case 'exit':
        return { cmd };
      default:
        if (line.startsWith('http://') || line.startsWith('https://')) {
          return { cmd: 'open', url: line };
        }
        return { cmd };
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputVal.trim() || isPending) return;

    setIsPending(true);
    try {
      const cmdObj = parseToCommand(inputVal);
      setHistory((prev) => [inputVal, ...prev.slice(0, 30)]);
      setHistoryIdx(-1);
      await onExecute(cmdObj);
    } catch (err: any) {
      console.error(err);
    } finally {
      setIsPending(false);
    }
  };

  const handleQuickCmd = (cmdText: string) => {
    setInputVal(cmdText);
    const cmdObj = parseToCommand(cmdText);
    onExecute(cmdObj);
  };

  const copyJSON = () => {
    if (!lastResult) return;
    navigator.clipboard.writeText(JSON.stringify(lastResult, null, 2));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-slate-900 border-t border-slate-800 p-3 sm:p-4 shadow-xl">
      <div className="max-w-7xl mx-auto space-y-2.5">
        {/* Terminal Header & Mode Switch */}
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
          <div className="flex items-center gap-2 text-slate-400 font-mono">
            <Terminal className="w-4 h-4 text-cyan-400" />
            <span className="font-semibold text-slate-200">Terminal do Agente IA</span>
            <span className="text-slate-600">•</span>
            <span className="hidden sm:inline">Execução de comandos determinísticos</span>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setIsJsonlMode(!isJsonlMode)}
              className={`px-2 py-0.5 rounded text-[11px] font-mono border transition-colors ${
                isJsonlMode
                  ? 'bg-cyan-950 text-cyan-300 border-cyan-700'
                  : 'bg-slate-800 text-slate-400 border-slate-700 hover:text-slate-200'
              }`}
            >
              Modo JSONL {isJsonlMode ? 'ATIVO' : 'CLI'}
            </button>

            <button
              type="button"
              onClick={() => setIsExpanded(!isExpanded)}
              className="text-slate-400 hover:text-slate-200 flex items-center gap-1 text-[11px]"
            >
              {isExpanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
              <span>{isExpanded ? 'Ocultar JSON' : 'Ver JSON'}</span>
            </button>
          </div>
        </div>

        {/* Quick Command Chips */}
        <div className="flex flex-wrap items-center gap-1.5 text-xs font-mono">
          <span className="text-slate-500 text-[11px]">Atalhos:</span>
          {[
            { label: 'discover', cmd: 'discover' },
            { label: 'inspect a1', cmd: 'inspect a1' },
            { label: 'run a1 (confirm)', cmd: 'run a1' },
            { label: 'press b1 (clique JS)', cmd: 'press b1' },
            { label: 'snapshot', cmd: 'snapshot' },
            { label: 'markdown', cmd: 'markdown' },
            { label: 'query #estado', cmd: 'query #estado' },
            { label: 'help', cmd: 'help' },
          ].map((item) => (
            <button
              key={item.label}
              onClick={() => handleQuickCmd(item.cmd)}
              className="px-2 py-0.5 rounded bg-slate-800/80 hover:bg-slate-700 text-slate-300 hover:text-white border border-slate-700/60 transition-colors"
            >
              {item.label}
            </button>
          ))}
        </div>

        {/* Command Input Form */}
        <form onSubmit={handleSubmit} className="flex items-center gap-2">
          <div className="relative flex-1">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 font-mono text-cyan-400 font-bold text-sm">
              &gt;
            </span>
            <input
              type="text"
              value={inputVal}
              onChange={(e) => setInputVal(e.target.value)}
              placeholder={
                isJsonlMode
                  ? '{"cmd":"open","url":"http://127.0.0.1:33137/","js":true}'
                  : 'Ex: open http://127.0.0.1:33137/ | discover | press b1 | query form'
              }
              className="w-full pl-8 pr-4 py-2 bg-slate-950 border border-slate-700/80 rounded-lg text-slate-100 font-mono text-sm focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 transition-colors"
            />
          </div>
          <button
            type="submit"
            disabled={isPending}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white font-medium rounded-lg text-sm flex items-center gap-1.5 transition-colors shadow-sm"
          >
            <Send className="w-4 h-4" />
            <span className="hidden sm:inline">Executar</span>
          </button>
        </form>

        {/* Telemetry Bar for Last Execution */}
        {lastResult && (
          <div className="flex flex-wrap items-center justify-between gap-2 p-2 rounded bg-slate-950/70 border border-slate-800 text-xs font-mono">
            <div className="flex items-center gap-2">
              {lastResult.ok ? (
                <span className="flex items-center gap-1 text-emerald-400">
                  <CheckCircle className="w-3.5 h-3.5" />
                  <span>OK 200</span>
                </span>
              ) : (
                <span className="flex items-center gap-1 text-red-400">
                  <AlertTriangle className="w-3.5 h-3.5" />
                  <span>Erro: {lastResult.error}</span>
                </span>
              )}
              <span className="text-slate-600">|</span>
              <span className="text-slate-400">Duração: {lastResult.duration_ms} ms</span>
              <span className="text-slate-600">|</span>
              <span className="text-slate-400">RSS: {lastResult.memory.rss_mib.toFixed(2)} MB</span>
            </div>

            {lastResult.js && (
              <span className="text-cyan-400 bg-cyan-950/80 px-2 py-0.5 rounded border border-cyan-800/40 text-[11px]">
                JS Worker: {lastResult.js.duration_ms} ms {lastResult.js.peak_mib ? `(${lastResult.js.peak_mib} MB)` : ''}
              </span>
            )}
          </div>
        )}

        {/* Expanded Raw JSONL Drawer */}
        {isExpanded && lastResult && (
          <div className="relative mt-2 p-3 bg-slate-950 rounded-lg border border-slate-800 text-xs font-mono text-slate-300 max-h-64 overflow-y-auto">
            <button
              onClick={copyJSON}
              className="absolute right-3 top-3 px-2 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded flex items-center gap-1 text-[11px]"
            >
              <Copy className="w-3 h-3" />
              <span>{copied ? 'Copiado!' : 'Copiar'}</span>
            </button>
            <pre className="whitespace-pre-wrap">{JSON.stringify(lastResult, null, 2)}</pre>
          </div>
        )}
      </div>
    </div>
  );
};
