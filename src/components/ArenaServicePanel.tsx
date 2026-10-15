import React, { useState } from 'react';
import { ARENA_MODELS, ArenaServiceBridge } from '../engine/arenaService';
import { ArenaModel, ArenaChatEvaluation } from '../types/browser';
import { ShieldCheck, Cpu, Key, Play, RefreshCw, Send, CheckCircle, ExternalLink } from 'lucide-react';

export const ArenaServicePanel: React.FC = () => {
  const [tokenInfo, setTokenInfo] = useState<{ token: string; expirySeconds: number; score: number }>(
    ArenaServiceBridge.mintRecaptchaToken()
  );
  const [promptInput, setPromptInput] = useState('Qual a melhor abordagem para reduzir consumo de RAM em agentes de IA?');
  const [evalResult, setEvalResult] = useState<ArenaChatEvaluation | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleMintFreshToken = () => {
    const t = ArenaServiceBridge.mintRecaptchaToken();
    setTokenInfo(t);
  };

  const handleRunBattle = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!promptInput.trim() || isLoading) return;

    setIsLoading(true);
    try {
      const res = await ArenaServiceBridge.runBlindEvaluation(promptInput);
      setEvalResult(res);
    } catch (err: any) {
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Header & Service Bridge Status */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
              <ShieldCheck className="w-5 h-5 text-cyan-400" />
              <span>Bridge Arena Service & Emissor de Tokens reCAPTCHA v3</span>
            </h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Reverso da API do LMSYS Chatbot Arena (`cmd/arenaservice`) com minting stealth CDP, mouse jitter e sessão persistente de avaliação.
            </p>
          </div>

          <div className="flex items-center gap-2 text-xs font-mono">
            <span className="px-2.5 py-1 rounded bg-slate-950 border border-slate-800 text-emerald-400 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
              <span>REST Bridge: Online (:8080)</span>
            </span>
          </div>
        </div>

        {/* reCAPTCHA v3 Stealth Mint Diagnostic Box */}
        <div className="p-3 rounded-lg bg-slate-950 border border-slate-800/80 space-y-2 text-xs font-mono">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2 text-slate-300">
              <Key className="w-4 h-4 text-cyan-400" />
              <span className="font-bold">Token reCAPTCHA v3 Ativo (Stealth CDP Minter):</span>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-emerald-400">Score Humano: {tokenInfo.score} (Passou)</span>
              <span className="text-slate-600">•</span>
              <span className="text-slate-400">Validade: {tokenInfo.expirySeconds}s</span>
              <button
                onClick={handleMintFreshToken}
                className="px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 flex items-center gap-1 transition-colors"
              >
                <RefreshCw className="w-3 h-3" />
                <span>Renovar</span>
              </button>
            </div>
          </div>

          <div className="text-slate-500 truncate text-[11px] bg-slate-900 p-2 rounded border border-slate-800">
            {tokenInfo.token}
          </div>
        </div>
      </div>

      {/* Interactive Blind Evaluation Battle Runner */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg space-y-4">
        <h3 className="text-sm font-bold text-slate-100 flex items-center gap-2">
          <Cpu className="w-4 h-4 text-cyan-400" />
          <span>Executar Batalha Cega de Avaliação (Arena Stream)</span>
        </h3>

        <form onSubmit={handleRunBattle} className="flex items-center gap-2">
          <input
            type="text"
            value={promptInput}
            onChange={(e) => setPromptInput(e.target.value)}
            placeholder="Digite sua pergunta de teste para os modelos..."
            className="flex-1 px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg text-slate-100 text-sm focus:border-cyan-500 focus:outline-none"
          />
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white rounded-lg text-sm font-medium flex items-center gap-2 transition-colors"
          >
            <Send className="w-4 h-4" />
            <span>{isLoading ? 'Avaliando...' : 'Enviar ao Arena'}</span>
          </button>
        </form>

        {/* Evaluation Result Cards */}
        {evalResult && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-3 border-t border-slate-800/80">
            <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
              <div className="flex items-center justify-between text-xs font-mono">
                <span className="font-bold text-cyan-400">Modelo A: {evalResult.modelA}</span>
                <span className="text-slate-500">{evalResult.timestamp}</span>
              </div>
              <p className="text-xs text-slate-300 leading-relaxed font-sans">
                {evalResult.responseA}
              </p>
            </div>

            <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
              <div className="flex items-center justify-between text-xs font-mono">
                <span className="font-bold text-blue-400">Modelo B: {evalResult.modelB}</span>
                <span className="text-slate-500">{evalResult.timestamp}</span>
              </div>
              <p className="text-xs text-slate-300 leading-relaxed font-sans">
                {evalResult.responseB}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Model Catalog / ELO Leaderboard Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-lg">
        <div className="px-4 py-3 bg-slate-950 border-b border-slate-800 flex items-center justify-between">
          <span className="text-xs font-bold text-slate-200 uppercase tracking-wider font-mono">
            Catálogo de Modelos do Arena (`/v1/models`)
          </span>
          <span className="text-xs font-mono text-slate-400">OpenAI Compatible REST</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse font-mono">
            <thead>
              <tr className="bg-slate-950/60 text-slate-400 border-b border-slate-800 text-[11px]">
                <th className="py-2.5 px-4">Modelo ID</th>
                <th className="py-2.5 px-4">Nome Público</th>
                <th className="py-2.5 px-4">Organização</th>
                <th className="py-2.5 px-4">Categoria</th>
                <th className="py-2.5 px-4 text-right">Arena Score (ELO)</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/80 text-slate-300">
              {ARENA_MODELS.map((m) => (
                <tr key={m.id} className="hover:bg-slate-800/40">
                  <td className="py-2.5 px-4 text-cyan-400 font-bold">{m.id}</td>
                  <td className="py-2.5 px-4 text-slate-200">{m.publicName}</td>
                  <td className="py-2.5 px-4 text-slate-400">{m.organization}</td>
                  <td className="py-2.5 px-4 text-slate-500">{m.category}</td>
                  <td className="py-2.5 px-4 text-right font-bold text-emerald-400">{m.score}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
