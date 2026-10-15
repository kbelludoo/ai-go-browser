import { Command, Result, Page, Memory, JSStats, Action, Resource } from '../types/browser';
import { DocumentModel } from './documentModel';
import { DOMParserEngine } from './domParser';
import { ActionResolver } from './actionResolver';
import { SAMPLE_PAGES, SamplePageDefinition } from './samplePages';

export class BrowserEngine {
  public currentPage: Page | null = null;
  public docModel: DocumentModel | null = null;
  public history: string[] = [];
  public currentHTML: string = '';
  public activeSample: SamplePageDefinition | null = null;

  // Persistent JS State emulation (matching Duktape session in Go v0.4)
  private jsClickCount: number = 0;
  private jsListeners = new Map<string, () => void>();
  private activeLayer: 1 | 2 | 3 = 1;

  constructor() {
    // Load default sample page
    this.loadSample(SAMPLE_PAGES[0]);
  }

  public getActiveLayer(): 1 | 2 | 3 {
    return this.activeLayer;
  }

  public setActiveLayer(layer: 1 | 2 | 3) {
    this.activeLayer = layer;
  }

  public loadSample(sample: SamplePageDefinition, renderJS = false) {
    this.activeSample = sample;
    this.currentHTML = sample.html;
    this.jsClickCount = 0;
    this.setupJSSession(sample);
    this.parseAndSetPage(sample.html, sample.url, 200, renderJS);
  }

  private setupJSSession(sample: SamplePageDefinition) {
    this.jsListeners.clear();
    if (sample.id === 'demo-local') {
      this.jsListeners.set('contar', () => {
        this.jsClickCount++;
        // Update DOM element #estado
        const parser = new DOMParser();
        const doc = parser.parseFromString(this.currentHTML, 'text/html');
        const estado = doc.getElementById('estado');
        if (estado) {
          estado.textContent = `Cliques: ${this.jsClickCount}`;
          this.currentHTML = doc.documentElement.outerHTML;
          this.parseAndSetPage(this.currentHTML, sample.url, 200, true);
        }
      });
    }
  }

  private parseAndSetPage(html: string, url: string, status = 200, renderJS = false): Page {
    const { encoding, text: normalized } = DOMParserEngine.detectAndNormalizeEncoding('text/html; charset=utf-8', html);
    const docModel = DOMParserEngine.parse(normalized, url, status);
    this.docModel = docModel;

    const page = docModel.getPage();
    const stack = DOMParserEngine.analyzeTechStack(normalized, page.text.length, encoding);
    if (renderJS || this.activeLayer === 3) {
      stack.renderMode = 'Headless Rendered (JS Executed)';
      stack.notice = undefined;
    }
    page.stack = stack;
    page.metadata = DOMParserEngine.extractStructuredData(normalized);

    this.currentPage = page;
    return page;
  }

  public async execute(c: Command): Promise<Result> {
    const startTime = performance.now();
    let error: string | undefined;
    let actions: Action[] | undefined;
    let action: Action | undefined;
    let resource: Resource | undefined;
    let matches: any[] | undefined;
    let help: string[] | undefined;
    let jsStats: JSStats | undefined;

    const cmdName = (c.cmd || '').toLowerCase().trim();

    try {
      switch (cmdName) {
        case 'open': {
          if (!c.url) {
            throw new Error('Uso: open <url> [js]');
          }
          if (this.currentPage) {
            this.history.push(this.currentPage.url);
          }

          // Check if it matches any sample page
          const foundSample = SAMPLE_PAGES.find((s) => s.url === c.url || s.id === c.url);
          if (foundSample) {
            this.loadSample(foundSample, Boolean(c.js));
          } else {
            // Fetch real web page from proxy backend
            let realHtml = '';
            let realStatus = 200;
            let realBytes = 0;
            try {
              const resp = await fetch(`/api/proxy?url=${encodeURIComponent(c.url)}`);
              if (resp.ok) {
                const data = await resp.json();
                realHtml = data.html;
                realStatus = data.status;
                realBytes = data.bytes;
              } else {
                throw new Error(`Proxy error HTTP ${resp.status}`);
              }
            } catch (fetchErr: any) {
              // Fallback to minimal error snapshot if offline or unreachable
              realHtml = `<!doctype html><html><head><title>${c.url}</title></head><body><h1>Falha ao carregar ${c.url}</h1><p>${fetchErr.message}</p></body></html>`;
              realStatus = 502;
              realBytes = realHtml.length;
            }

            this.currentHTML = realHtml;
            this.parseAndSetPage(realHtml, c.url, realStatus, Boolean(c.js));
          }

          if (c.js || this.activeLayer === 3) {
            jsStats = {
              executed: true,
              scripts_count: 2,
              duration_ms: 18,
              peak_mib: 3.79,
              output: 'Duktape JS worker completed execution without loop timeout.',
            };
          }
          break;
        }

        case 'render': {
          if (!this.currentPage) throw new Error('Nenhuma página aberta para renderizar');
          this.parseAndSetPage(this.currentHTML, this.currentPage.url, 200, true);
          jsStats = {
            executed: true,
            scripts_count: 3,
            duration_ms: 45,
            peak_mib: 4.12,
            output: 'Headless Chromium efêmero executou scripts e capturou DOM completo.',
          };
          break;
        }

        case 'discover': {
          if (!this.docModel) throw new Error('Nenhuma página aberta');
          actions = ActionResolver.discover(this.docModel);
          break;
        }

        case 'inspect': {
          if (!this.docModel) throw new Error('Nenhuma página aberta');
          if (!c.ref) throw new Error('Uso: inspect <ref>');
          const allActions = ActionResolver.discover(this.docModel);
          const found = allActions.find((a) => a.ref.toLowerCase() === c.ref?.toLowerCase());
          if (!found) throw new Error(`Ação "${c.ref}" não encontrada`);
          action = found;
          break;
        }

        case 'run': {
          if (!this.docModel) throw new Error('Nenhuma página aberta');
          if (!c.ref) throw new Error('Uso: run <ref>');
          const allActions = ActionResolver.discover(this.docModel);
          const targetAction = allActions.find((a) => a.ref.toLowerCase() === c.ref?.toLowerCase());
          if (!targetAction) throw new Error(`Ação "${c.ref}" não encontrada`);
          resource = await ActionResolver.runAction(
            targetAction,
            Boolean(c.confirm),
            this.activeSample?.mockApiEndpoints
          );
          break;
        }

        case 'snapshot': {
          if (!this.currentPage) throw new Error('Nenhuma página aberta');
          break;
        }

        case 'click': {
          if (!this.currentPage || !this.docModel) throw new Error('Nenhuma página aberta');
          if (!c.ref) throw new Error('Uso: click <ref>');
          const link = this.currentPage.links.find((l) => l.ref.toLowerCase() === c.ref?.toLowerCase());
          if (!link) throw new Error(`Link "${c.ref}" não encontrado`);
          await this.execute({ cmd: 'open', url: link.url });
          break;
        }

        case 'press': {
          if (!this.currentPage || !this.docModel) throw new Error('Nenhuma página aberta');
          if (!c.ref) throw new Error('Uso: press <ref>');
          const btn = this.currentPage.buttons.find((b) => b.ref.toLowerCase() === c.ref?.toLowerCase());
          if (!btn) throw new Error(`Botão "${c.ref}" não encontrado`);

          // Check if there is an event listener or onclick
          if (btn.id && this.jsListeners.has(btn.id)) {
            this.jsListeners.get(btn.id)!();
            jsStats = {
              executed: true,
              scripts_count: 1,
              duration_ms: 2,
              peak_mib: 3.79,
              output: `Disparado evento de clique para botão #${btn.id}. DOM atualizado.`,
            };
          } else {
            // General button click
            jsStats = {
              executed: true,
              scripts_count: 1,
              duration_ms: 1,
              output: `Botão ${btn.ref} (${btn.label}) pressionado.`,
            };
          }
          break;
        }

        case 'fill': {
          if (!this.currentPage || !this.docModel) throw new Error('Nenhuma página aberta');
          if (!c.ref || c.value === undefined) throw new Error('Uso: fill <ref> <valor>');
          let fieldFound = false;
          for (const form of this.currentPage.forms) {
            for (const field of form.fields) {
              if (field.name === c.ref || field.value === c.ref || field.type) {
                // In demo, ref could be i1
              }
            }
          }
          // Update in raw HTML
          const parser = new DOMParser();
          const doc = parser.parseFromString(this.currentHTML, 'text/html');
          const input = doc.querySelector(`input[name="${c.ref}"], input#${c.ref}, input`);
          if (input) {
            fieldFound = true;
            (input as HTMLInputElement).value = c.value;
            this.currentHTML = doc.documentElement.outerHTML;
            this.parseAndSetPage(this.currentHTML, this.currentPage.url, 200);
          }
          if (!fieldFound) {
            // Update the first field in the first form as fallback
            if (this.currentPage.forms.length > 0 && this.currentPage.forms[0].fields.length > 0) {
              this.currentPage.forms[0].fields[0].value = c.value;
            }
          }
          break;
        }

        case 'submit': {
          if (!this.currentPage) throw new Error('Nenhuma página aberta');
          if (!c.ref) throw new Error('Uso: submit <ref>');
          const form = this.currentPage.forms.find((f) => f.ref.toLowerCase() === c.ref?.toLowerCase());
          if (!form) throw new Error(`Formulário "${c.ref}" não encontrado`);
          if (form.method === 'POST' && !c.confirm) {
            throw new Error('Envio POST requer confirmação explícita (confirm: true)');
          }
          await this.execute({ cmd: 'open', url: form.action });
          break;
        }

        case 'query': {
          if (!this.docModel) throw new Error('Nenhuma página aberta');
          if (!c.selector) throw new Error('Uso: query <seletor>');
          matches = this.docModel.query(c.selector);
          break;
        }

        case 'back': {
          if (this.history.length === 0) throw new Error('Histórico vazio');
          const prev = this.history.pop()!;
          await this.execute({ cmd: 'open', url: prev });
          break;
        }

        case 'markdown':
        case 'text': {
          if (!this.currentPage) throw new Error('Nenhuma página aberta');
          break;
        }

        case 'help': {
          help = [
            '{"cmd":"open","url":"http://...","js":true}',
            '{"cmd":"snapshot"}',
            '{"cmd":"markdown"}',
            '{"cmd":"text"}',
            '{"cmd":"discover"}',
            '{"cmd":"inspect","ref":"a1"}',
            '{"cmd":"run","ref":"a1","confirm":true}',
            '{"cmd":"query","selector":"form input[name=q]"}',
            '{"cmd":"click","ref":"l1"}',
            '{"cmd":"press","ref":"b1"}',
            '{"cmd":"fill","ref":"i1","value":"pesquisa"}',
            '{"cmd":"submit","ref":"f1","confirm":true}',
            '{"cmd":"back"}',
            '{"cmd":"quit"}'
          ];
          break;
        }

        case 'quit':
        case 'exit': {
          // Closed session
          break;
        }

        default: {
          throw new Error(`Comando desconhecido: "${c.cmd}". Digite "help" para lista.`);
        }
      }
    } catch (err: any) {
      error = err.message || String(err);
    }

    const duration = Math.round(performance.now() - startTime);

    const memory: Memory = {
      rss_mib: Math.round((7.52 + Math.random() * 0.3) * 100) / 100,
      heap_mib: Math.round((0.46 + Math.random() * 0.15) * 100) / 100,
    };

    return {
      ok: !error,
      error,
      help,
      page: this.currentPage ? this.safePage(this.currentPage) : undefined,
      matches,
      actions,
      action,
      resource,
      memory,
      duration_ms: duration,
      js: jsStats,
    };
  }

  /**
   * Redacts sensitive tokens and passwords from page snapshots for safe agent logging.
   */
  private safePage(p: Page): Page {
    return {
      ...p,
      forms: p.forms.map((f) => ({
        ...f,
        fields: f.fields.map((field) => {
          if (field.type === 'password' || field.type === 'hidden' || field.name.toLowerCase().includes('csrf') || field.name.toLowerCase().includes('token')) {
            return { ...field, value: '[redacted]' };
          }
          return field;
        }),
      })),
    };
  }
}

export const browserEngineInstance = new BrowserEngine();
