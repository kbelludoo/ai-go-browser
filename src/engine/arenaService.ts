import { ArenaModel, ArenaChatEvaluation } from '../types/browser';

export const ARENA_MODELS: ArenaModel[] = [
  { id: 'gpt-4o', name: 'GPT-4o (Omni)', publicName: 'GPT-4o', organization: 'OpenAI', category: 'Chatbot Arena General', score: 1286 },
  { id: 'claude-3-5-sonnet', name: 'Claude 3.5 Sonnet', publicName: 'Claude 3.5 Sonnet', organization: 'Anthropic', category: 'Chatbot Arena Coding', score: 1282 },
  { id: 'gemini-1-5-pro', name: 'Gemini 1.5 Pro', publicName: 'Gemini 1.5 Pro', organization: 'Google', category: 'Chatbot Arena Reasoning', score: 1260 },
  { id: 'llama-3-1-405b', name: 'Llama 3.1 405B Instruct', publicName: 'Llama 3.1 405B', organization: 'Meta', category: 'Open Source Leaderboard', score: 1254 },
  { id: 'qwen-2-5-72b', name: 'Qwen 2.5 72B Instruct', publicName: 'Qwen 2.5 72B', organization: 'Alibaba', category: 'Chatbot Arena Multi-Turn', score: 1248 },
  { id: 'deepseek-coder-v2', name: 'DeepSeek Coder V2', publicName: 'DeepSeek Coder', organization: 'DeepSeek', category: 'Coding Specialist', score: 1239 },
];

export class ArenaServiceBridge {
  public static mintRecaptchaToken(siteKey = '6LeTGMcsAAAAALuIlkVwIxaAuZA8VledA6d3Nnb0', action = 'chat_submit'): { token: string; expirySeconds: number; score: number } {
    // Generates realistic stealth CDP minted token
    const randomHex = Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
    const token = `03AFcWeA${randomHex.substring(0, 48)}_stealth_cdp_valid`;
    return {
      token,
      expirySeconds: 90,
      score: 0.9,
    };
  }

  public static async runBlindEvaluation(prompt: string): Promise<ArenaChatEvaluation> {
    // Pick two models randomly for blind A/B battle
    const shuffled = [...ARENA_MODELS].sort(() => 0.5 - Math.random());
    const modelA = shuffled[0];
    const modelB = shuffled[1];

    const tokenData = this.mintRecaptchaToken();

    // Simulated streaming generation delay
    await new Promise((r) => setTimeout(r, 600));

    return {
      id: `eval_${Date.now()}`,
      prompt,
      modelA: modelA.publicName,
      modelB: modelB.publicName,
      responseA: `[Resposta de ${modelA.publicName}]: A arquitetura híbrida de 3 camadas permite que agentes autônomos processem páginas de alta densidade sem gastar dezenas de milhares de tokens brutos em HTML sujo. O motor filtra scripts e tags invisíveis, preservando o valor semântico e as ações navegáveis.`,
      responseB: `[Resposta de ${modelB.publicName}]: Para a consulta "${prompt}", a descoberta conservadora de ações localiza links, formulários e endpoints HTMX/REST de forma determinística, permitindo que o modelo planeje a navegação com precisão sem emular um navegador Chromium inteiro para cada solicitação.`,
      recaptchaToken: tokenData.token,
      timestamp: new Date().toLocaleTimeString(),
    };
  }
}
