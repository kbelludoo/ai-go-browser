import { Heading, Link, Button, Form, ActionField, Page, ElementMatch } from '../types/browser';

export interface NodeModel {
  tag: string;
  text: string;
  attr: Record<string, string>;
  children: NodeModel[];
}

export class DocumentModel {
  public rawDoc: Document;
  public rawHTML: string;
  public url: string;
  public status: number;
  public bytes: number;

  public linksMap = new Map<string, HTMLAnchorElement>();
  public buttonsMap = new Map<string, HTMLButtonElement | HTMLElement>();
  public fieldsMap = new Map<string, HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>();
  public formsMap = new Map<string, HTMLFormElement>();

  constructor(doc: Document, rawHTML: string, url: string, status: number, bytes: number) {
    this.rawDoc = doc;
    this.rawHTML = rawHTML;
    this.url = url;
    this.status = status;
    this.bytes = bytes;
  }

  /**
   * Generates a safe and comprehensive Page representation.
   */
  public getPage(maxText = 8000): Page {
    this.linksMap.clear();
    this.buttonsMap.clear();
    this.fieldsMap.clear();
    this.formsMap.clear();

    const title = this.rawDoc.title || 'Sem título';
    const descEl = this.rawDoc.querySelector('meta[name="description"]');
    const description = descEl ? descEl.getAttribute('content') || '' : '';

    // 1. Extract Headings
    const headings: Heading[] = [];
    const hEls = this.rawDoc.querySelectorAll('h1, h2, h3, h4, h5, h6');
    hEls.forEach((el) => {
      const level = parseInt(el.tagName.substring(1), 10);
      const text = (el.textContent || '').trim().replace(/\s+/g, ' ');
      if (text) {
        headings.push({ level, text });
      }
    });

    // 2. Extract Links
    const links: Link[] = [];
    const aEls = this.rawDoc.querySelectorAll('a[href]');
    let linkIdx = 1;
    aEls.forEach((el) => {
      const a = el as HTMLAnchorElement;
      const ref = `l${linkIdx++}`;
      this.linksMap.set(ref, a);
      const rawHref = a.getAttribute('href') || '';
      let resolvedUrl = rawHref;
      try {
        resolvedUrl = new URL(rawHref, this.url).toString();
      } catch {
        resolvedUrl = rawHref;
      }
      const label = (a.textContent || '').trim().replace(/\s+/g, ' ') || resolvedUrl;
      links.push({ ref, label, url: resolvedUrl });
    });

    // 3. Extract Buttons
    const buttons: Button[] = [];
    const btnEls = this.rawDoc.querySelectorAll('button, input[type="button"], input[type="submit"]');
    let btnIdx = 1;
    btnEls.forEach((el) => {
      const ref = `b${btnIdx++}`;
      this.buttonsMap.set(ref, el as HTMLElement);
      const id = el.getAttribute('id') || undefined;
      let label = (el.textContent || '').trim().replace(/\s+/g, ' ');
      if (!label && el instanceof HTMLInputElement) {
        label = el.value || 'Botão';
      }
      buttons.push({
        ref,
        id,
        label: label || id || 'Botão',
        type: el.getAttribute('type') || 'button',
      });
    });

    // 4. Extract Forms and Fields
    const forms: Form[] = [];
    const formEls = this.rawDoc.querySelectorAll('form');
    let formIdx = 1;
    let fieldIdx = 1;

    formEls.forEach((formEl) => {
      const formRef = `f${formIdx++}`;
      this.formsMap.set(formRef, formEl);
      const actionAttr = formEl.getAttribute('action') || '';
      let actionUrl = actionAttr;
      try {
        actionUrl = new URL(actionAttr, this.url).toString();
      } catch {
        actionUrl = actionAttr || this.url;
      }

      const method = (formEl.getAttribute('method') || 'GET').toUpperCase();
      const fields: ActionField[] = [];

      const inputEls = formEl.querySelectorAll('input, textarea, select');
      inputEls.forEach((input) => {
        const fieldRef = `i${fieldIdx++}`;
        const inputEl = input as HTMLInputElement;
        this.fieldsMap.set(fieldRef, inputEl);
        const name = inputEl.name || inputEl.id || `field_${fieldRef}`;
        const type = (inputEl.type || 'text').toLowerCase();
        const value = inputEl.value || '';
        fields.push({
          name,
          type,
          value,
        });
      });

      const submitBtn = formEl.querySelector('button[type="submit"], input[type="submit"], button:not([type])');
      const submit_label = submitBtn ? (submitBtn.textContent || (submitBtn as HTMLInputElement).value || '').trim() : 'Pesquisar';

      forms.push({
        ref: formRef,
        action: actionUrl,
        method,
        fields,
        submit_label,
      });
    });

    // 5. Pagination detection for multi-page crawling
    let nextPageUrl: string | undefined;
    const nextLink = this.rawDoc.querySelector('link[rel="next"], a[rel="next"]');
    if (nextLink) {
      const href = nextLink.getAttribute('href');
      if (href) {
        try {
          nextPageUrl = new URL(href, this.url).toString();
        } catch {
          nextPageUrl = href;
        }
      }
    } else {
      // Heuristic search in navigation links
      for (const link of links) {
        if (/^(próxim[ao]|next|avançar|>|»|page\s*2|\?p=2|\?page=2)$/i.test(link.label.trim())) {
          nextPageUrl = link.url;
          break;
        }
      }
    }

    // 6. Clean text extraction with noise & cookie banner suppression
    const rawText = this.extractCleanText(this.rawDoc.body);
    const text = rawText.length > maxText ? rawText.substring(0, maxText) + '...' : rawText;

    // 7. Markdown generation
    const markdown = this.generateMarkdown();

    const readabilityScore = Math.min(100, Math.round((text.length / Math.max(1, this.bytes)) * 400));

    return {
      url: this.url,
      title,
      description,
      text,
      markdown,
      headings,
      links,
      buttons,
      forms,
      status: this.status,
      bytes: this.bytes,
      nextPageUrl,
      readabilityScore,
    };
  }

  /**
   * Executes CSS query selector against the parsed document.
   */
  public query(selector: string): ElementMatch[] {
    const results: ElementMatch[] = [];
    try {
      const els = this.rawDoc.querySelectorAll(selector);
      const limit = Math.min(els.length, 32);
      for (let i = 0; i < limit; i++) {
        const el = els[i];
        results.push({
          tag: el.tagName.toLowerCase(),
          id: el.getAttribute('id') || undefined,
          role: el.getAttribute('role') || undefined,
          text: (el.textContent || '').trim().replace(/\s+/g, ' ').substring(0, 100),
          href: el.getAttribute('href') || undefined,
          name: el.getAttribute('name') || undefined,
          value: (el as HTMLInputElement).value || undefined,
        });
      }
    } catch {
      throw new Error(`Seletor CSS inválido: "${selector}"`);
    }
    return results;
  }

  private extractCleanText(root: HTMLElement | null): string {
    if (!root) return '';
    const clone = root.cloneNode(true) as HTMLElement;
    // Remove invisible and non-content tags
    const toRemove = clone.querySelectorAll(
      'script, style, noscript, svg, template, iframe, ' +
      '[class*="cookie" i], [class*="consent" i], [id*="cookie" i], [id*="gdpr" i], ' +
      '[aria-label*="cookie" i], [aria-label*="consent" i], ' +
      '.advertisement, .ad-banner, [role="alertdialog"], [aria-modal="true"]'
    );
    toRemove.forEach((el) => el.remove());

    // Prefer main or article container if it contains substantial text
    const mainEl = clone.querySelector('main, article, #content, .post-content, .entry-content');
    const targetRoot = mainEl && (mainEl.textContent || '').trim().length > 100 ? mainEl : clone;

    const walker = this.rawDoc.createTreeWalker(targetRoot, NodeFilter.SHOW_TEXT);
    const chunks: string[] = [];
    let currentNode = walker.nextNode();
    while (currentNode) {
      const val = (currentNode.nodeValue || '').trim().replace(/\s+/g, ' ');
      if (val) {
        chunks.push(val);
      }
      currentNode = walker.nextNode();
    }
    return chunks.join(' ');
  }

  private generateMarkdown(): string {
    const lines: string[] = [];
    const title = this.rawDoc.title || '';
    if (title) {
      lines.push(`# ${title}\n`);
    }

    const walker = this.rawDoc.createTreeWalker(this.rawDoc.body || this.rawDoc.documentElement, NodeFilter.SHOW_ELEMENT);
    let curr = walker.nextNode() as HTMLElement | null;
    while (curr) {
      const tag = curr.tagName.toLowerCase();
      if (['h1', 'h2', 'h3', 'h4', 'h5', 'h6'].includes(tag)) {
        const hashes = '#'.repeat(parseInt(tag[1], 10));
        lines.push(`\n${hashes} ${(curr.textContent || '').trim()}\n`);
      } else if (tag === 'p') {
        const text = (curr.textContent || '').trim().replace(/\s+/g, ' ');
        if (text) lines.push(`${text}\n`);
      } else if (tag === 'li') {
        const text = (curr.textContent || '').trim().replace(/\s+/g, ' ');
        if (text) lines.push(`- ${text}`);
      }
      curr = walker.nextNode() as HTMLElement | null;
    }
    return lines.join('\n').trim();
  }
}
