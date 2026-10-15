import { Action, ActionField, Resource } from '../types/browser';
import { DocumentModel } from './documentModel';

export class ActionResolver {
  /**
   * Discovers network-facing actions from the DOM and embedded scripts.
   */
  static discover(docModel: DocumentModel): Action[] {
    const actions: Action[] = [];
    let actIdx = 1;
    const addAction = (act: Omit<Action, 'ref'>) => {
      if (actions.length >= 100) return;
      actions.push({
        ref: `a${actIdx++}`,
        ...act,
      });
    };

    const doc = docModel.rawDoc;
    const baseURL = docModel.url;

    // 1. HTML Forms
    const forms = doc.querySelectorAll('form');
    forms.forEach((form, idx) => {
      const actionAttr = form.getAttribute('action') || '';
      let url = actionAttr;
      try {
        url = new URL(actionAttr, baseURL).toString();
      } catch {
        url = actionAttr;
      }
      const method = (form.getAttribute('method') || 'GET').toUpperCase() as any;
      const fields: ActionField[] = [];
      const inputs = form.querySelectorAll('input, select, textarea');
      inputs.forEach((inp) => {
        const inputEl = inp as HTMLInputElement;
        const name = inputEl.name || inputEl.id;
        if (name) {
          fields.push({
            name,
            type: inputEl.type || 'text',
            value: inputEl.value || '',
          });
        }
      });

      const submitBtn = form.querySelector('button[type="submit"], input[type="submit"], button:not([type])');
      const label = submitBtn ? (submitBtn.textContent || (submitBtn as HTMLInputElement).value || '').trim() : `Enviar Formulario #${idx + 1}`;

      addAction({
        label: label || 'Enviar Formulário',
        kind: 'form',
        method: method || 'GET',
        url,
        source: `html:form#f${idx + 1}`,
        evidence: 'form action/method',
        preview: `${method} ${url}`,
        confidence: 'declared',
        requires_confirmation: method !== 'GET',
        executable: true,
        fields,
      });
    });

    // 2. Buttons with JS event handlers or API triggers
    const buttons = doc.querySelectorAll('button[id], input[type="button"][id]');
    buttons.forEach((btn, idx) => {
      const id = btn.getAttribute('id') || `b${idx + 1}`;
      const label = (btn.textContent || (btn as HTMLInputElement).value || '').trim() || id;

      // Check for inline onclick
      const onclick = btn.getAttribute('onclick') || '';
      const fetchMatch = onclick.match(/fetch\(['"`]([^'"`]+)['"`]\)/i);
      if (fetchMatch) {
        let apiTarget = fetchMatch[1];
        try {
          apiTarget = new URL(apiTarget, baseURL).toString();
        } catch {
          // relative
        }
        addAction({
          label,
          kind: 'api',
          method: 'GET',
          url: apiTarget,
          source: `javascript:button#${id}`,
          evidence: 'fetch(literal URL) in onclick',
          preview: `GET ${apiTarget}`,
          confidence: 'inferred',
          requires_confirmation: true,
          executable: true,
        });
      }
    });

    // 3. HTMX attributes on ANY element (button, section, div, a)
    const htmxElements = doc.querySelectorAll('[hx-get], [hx-post], [hx-put], [hx-patch], [hx-delete]');
    htmxElements.forEach((el, idx) => {
      const hxGet = el.getAttribute('hx-get');
      const hxPost = el.getAttribute('hx-post');
      const hxPut = el.getAttribute('hx-put');
      const hxPatch = el.getAttribute('hx-patch');
      const hxDelete = el.getAttribute('hx-delete');

      let method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' = 'GET';
      let endpoint = '';
      if (hxGet) { method = 'GET'; endpoint = hxGet; }
      else if (hxPost) { method = 'POST'; endpoint = hxPost; }
      else if (hxPut) { method = 'PUT'; endpoint = hxPut; }
      else if (hxPatch) { method = 'PATCH'; endpoint = hxPatch; }
      else if (hxDelete) { method = 'DELETE'; endpoint = hxDelete; }

      let fullUrl = endpoint;
      try {
        fullUrl = new URL(endpoint, baseURL).toString();
      } catch {
        fullUrl = endpoint;
      }

      const label = (el.textContent || '').trim().substring(0, 40) || `HTMX ${method} ${endpoint}`;
      addAction({
        label,
        kind: method === 'GET' ? 'button' : 'api',
        method,
        url: fullUrl,
        source: `htmx:${el.tagName.toLowerCase()}#${el.id || idx + 1}`,
        evidence: `hx-${method.toLowerCase()} attribute`,
        preview: `${method} ${fullUrl}`,
        confidence: 'declared',
        requires_confirmation: method !== 'GET',
        executable: method === 'GET',
      });
    });

    // 4. Advertised JSON representations in <head>
    const altLinks = doc.querySelectorAll('link[rel="alternate"][type="application/json"], link[rel="service-desc"]');
    altLinks.forEach((link) => {
      const href = link.getAttribute('href') || '';
      let jsonUrl = href;
      try {
        jsonUrl = new URL(href, baseURL).toString();
      } catch {
        jsonUrl = href;
      }
      addAction({
        label: 'Representação JSON Anunciada',
        kind: 'api',
        method: 'GET',
        url: jsonUrl,
        source: 'html:link[rel=alternate]',
        evidence: 'link[rel="alternate"][type="application/json"]',
        preview: `GET ${jsonUrl}`,
        confidence: 'declared',
        requires_confirmation: false,
        executable: true,
      });
    });

    // 5. Candidate URLs from script tags (fetch, axios, jQuery)
    const scripts = doc.querySelectorAll('script:not([type="application/json"]):not([type="application/ld+json"])');
    scripts.forEach((s) => {
      const code = s.textContent || '';
      // fetch matches
      const fetchRegex = /fetch\s*\(\s*['"`]([^'"`]+)['"`]/g;
      let match: RegExpExecArray | null;
      while ((match = fetchRegex.exec(code)) !== null) {
        const path = match[1];
        if (path.startsWith('http://') || path.startsWith('https://') || path.startsWith('/')) {
          let resolved = path;
          try { resolved = new URL(path, baseURL).toString(); } catch {}
          addAction({
            label: `API: ${path}`,
            kind: 'api',
            method: 'GET',
            url: resolved,
            source: 'javascript:fetch',
            evidence: `fetch("${path}") literal script`,
            preview: `GET ${resolved}`,
            confidence: 'inferred',
            requires_confirmation: true,
            executable: true,
          });
        }
      }

      // axios matches
      const axiosRegex = /axios\s*\.\s*(get|post)\s*\(\s*['"`]([^'"`]+)['"`]/gi;
      while ((match = axiosRegex.exec(code)) !== null) {
        const m = match[1].toUpperCase() as any;
        const p = match[2];
        let resolved = p;
        try { resolved = new URL(p, baseURL).toString(); } catch {}
        addAction({
          label: `Axios ${m}: ${p}`,
          kind: 'api',
          method: m,
          url: resolved,
          source: `javascript:axios.${match[1]}`,
          evidence: `axios.${match[1]}("${p}") script`,
          preview: `${m} ${resolved}`,
          confidence: 'inferred',
          requires_confirmation: true,
          executable: m === 'GET',
        });
      }
    });

    // 6. Navigation links (sample high-priority ones)
    const links = doc.querySelectorAll('a[href]');
    let linkCount = 0;
    links.forEach((a) => {
      if (linkCount >= 5) return;
      const href = a.getAttribute('href') || '';
      if (href && !href.startsWith('#') && !href.startsWith('javascript:')) {
        let fullHref = href;
        try { fullHref = new URL(href, baseURL).toString(); } catch {}
        const label = (a.textContent || '').trim().substring(0, 30) || fullHref;
        addAction({
          label: `Link: ${label}`,
          kind: 'link',
          method: 'GET',
          url: fullHref,
          source: `html:a[href]`,
          evidence: 'href attribute',
          preview: `GET ${fullHref}`,
          confidence: 'declared',
          requires_confirmation: false,
          executable: true,
        });
        linkCount++;
      }
    });

    return actions;
  }

  /**
   * Safe execution of a discovered action.
   */
  static async runAction(
    action: Action,
    confirm: boolean,
    mockEndpoints?: Record<string, { status: number; contentType: string; data: any }>
  ): Promise<Resource> {
    if (action.requires_confirmation && !confirm) {
      throw new Error(`Ação requer confirmação explícita (confirm: true) para evitar efeitos colaterais.`);
    }

    if (!action.executable) {
      throw new Error(`Ação ${action.ref} (${action.method} ${action.url}) não é executável de forma autônoma sem preenchimento adicional.`);
    }

    const targetUrl = action.url || '';
    const parsedUrl = new URL(targetUrl, 'http://127.0.0.1:33137');
    const path = parsedUrl.pathname;

    // Check mock endpoints first
    if (mockEndpoints && mockEndpoints[path]) {
      const endpoint = mockEndpoints[path];
      return {
        url: targetUrl,
        status: endpoint.status,
        content_type: endpoint.contentType,
        data: endpoint.data,
      };
    }

    // Default simulated response for external endpoints
    return {
      url: targetUrl,
      status: 200,
      content_type: 'application/json',
      data: {
        executed: true,
        method: action.method,
        action_ref: action.ref,
        confidence: action.confidence,
        timestamp: new Date().toISOString(),
        payload: {
          status: 'success',
          items_retrieved: 2,
          records: [
            { id: 101, name: 'Sample Item A', active: true },
            { id: 102, name: 'Sample Item B', active: false },
          ],
        },
      },
    };
  }
}
