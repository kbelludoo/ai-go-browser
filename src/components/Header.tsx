import React from 'react';
import { Cpu, Globe, Layers, ShieldCheck, Zap, Activity } from 'lucide-react';
import { Memory } from '../types/browser';

interface HeaderProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  activeLayer: 1 | 2 | 3;
  setActiveLayer: (layer: 1 | 2 | 3) => void;
  memory: Memory;
  durationMs: number;
}

export const Header: React.FC<HeaderProps> = ({
  activeTab,
  setActiveTab,
  activeLayer,
  setActiveLayer,
  memory,
  durationMs,
}) => {
  return (
    <header className="border-b border-slate-800 bg-slate-900/90 backdrop-blur sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 py-2.5 flex flex-wrap items-center justify-between gap-3">
        {/* Logo and Architecture Subtitle */}
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-cyan-500 to-blue-600 flex items-center justify-center shadow-lg shadow-cyan-500/20 text-white font-black tracking-wider">
            AI
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-slate-100 tracking-tight text-base">
                AI Go Browser
              </span>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-cyan-950/80 text-cyan-400 border border-cyan-800/50">
                v0.7 Universal
              </span>
            </div>
            <p className="text-xs text-slate-400 hidden sm:block">
              Navegador Híbrido Semântico para Agentes de IA
            </p>
          </div>
        </div>

        {/* 3-Layer Architecture Switch */}
        <div className="flex items-center gap-1 bg-slate-950 p-1 rounded-lg border border-slate-800 text-xs">
          <button
            onClick={() => setActiveLayer(1)}
            title="Camada 1: Motor Go Puro para 80%+ da Web (<25 MB RAM)"
            className={`px-2.5 py-1 rounded font-medium flex items-center gap-1.5 transition-colors ${
              activeLayer === 1
                ? 'bg-cyan-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Zap className="w-3.5 h-3.5" />
            <span>Go Puro (L1)</span>
          </button>
          <button
            onClick={() => setActiveLayer(2)}
            title="Camada 2: Extrator de Dados Estruturados & Hydration (Next/Nuxt/JSON-LD)"
            className={`px-2.5 py-1 rounded font-medium flex items-center gap-1.5 transition-colors ${
              activeLayer === 2
                ? 'bg-cyan-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Layers className="w-3.5 h-3.5" />
            <span>Estruturado (L2)</span>
          </button>
          <button
            onClick={() => setActiveLayer(3)}
            title="Camada 3: Motor Headless Efêmero sob Demanda para SPAs"
            className={`px-2.5 py-1 rounded font-medium flex items-center gap-1.5 transition-colors ${
              activeLayer === 3
                ? 'bg-cyan-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Globe className="w-3.5 h-3.5" />
            <span>Headless JS (L3)</span>
          </button>
        </div>

        {/* Live Memory & Speed Telemetry */}
        <div className="flex items-center gap-3 text-xs font-mono">
          <div className="flex items-center gap-1.5 bg-slate-950/80 px-2.5 py-1 rounded border border-slate-800 text-emerald-400">
            <Cpu className="w-3.5 h-3.5 text-emerald-500" />
            <span>RSS: {memory.rss_mib.toFixed(2)} MB</span>
            <span className="text-slate-600">|</span>
            <span className="text-slate-400">Heap: {memory.heap_mib.toFixed(2)} MB</span>
          </div>

          <div className="hidden md:flex items-center gap-1.5 bg-slate-950/80 px-2.5 py-1 rounded border border-slate-800 text-cyan-400">
            <Activity className="w-3.5 h-3.5 text-cyan-500" />
            <span>{durationMs} ms</span>
          </div>
        </div>
      </div>

      {/* Main Tab Navigation */}
      <div className="max-w-7xl mx-auto px-4 flex overflow-x-auto gap-2 border-t border-slate-800/60 pt-1 text-xs">
        {[
          { id: 'browser', label: 'Console / Agente IA', icon: Globe },
          { id: 'realsites', label: '🧪 Teste em Sites Reais', icon: Activity },
          { id: 'actions', label: 'Ações Descobertas (discover)', icon: Zap },
          { id: 'dom', label: 'Inspetor DOM & Dados (L2)', icon: Layers },
          { id: 'markdown', label: 'Digest Markdown LLM', icon: Cpu },
          { id: 'benchmark', label: 'Benchmark Tokens (tokenbench)', icon: Activity },
          { id: 'arena', label: 'Arena Service & Anti-Bot', icon: ShieldCheck },
        ].map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-1.5 px-3 py-2 border-b-2 font-medium transition-colors whitespace-nowrap ${
                isActive
                  ? 'border-cyan-500 text-cyan-400 bg-cyan-950/20'
                  : 'border-transparent text-slate-400 hover:text-slate-200 hover:border-slate-700'
              }`}
            >
              <Icon className="w-3.5 h-3.5" />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>
    </header>
  );
};
