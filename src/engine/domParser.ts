import { DocumentModel, NodeModel } from './documentModel';
import { Page, Heading, Link, Button, Form, ActionField, TechStack } from '../types/browser';

export class DOMParserEngine {
  /**
   * Detects and normalizes character encoding.
   */
  static detectAndNormalizeEncoding(contentType: string, raw: string): { encoding: string; text: string } {
    let enc = 'utf-8';
    const ctMatch = contentType.match(/charset=([a-zA-Z0-9_\-]+)/i);
    if (ctMatch) {
      enc = ctMatch[1].toLowerCase();
    } else {
      const metaMatch = raw.match(/<meta\s+[^>]*charset=["']?([a-zA-Z0-9_\-]+)/i) ||
                        raw.match(/<meta\s+[^>]*content=["'][^"']*charset=([a-zA-Z0-9_\-]+)/i);
      if (metaMatch) {
        enc = metaMatch[1].toLowerCase();
      }
    }
    return { encoding: enc, text: raw };
  }

  /**
   * Analyze tech stack: framework, rendering strategy (SSR vs CSR), content yield.
   */
  static analyzeTechStack(rawHTML: string, textLen: number, encoding: string): TechStack {
    const lower = rawHTML.toLowerCase();
    const totalLen = rawHTML.length;
    if (totalLen === 0) {
      return { framework: 'Empty', renderMode: 'None', encoding, contentYield: 0 };
    }

    const yieldRatio = Math.round(((textLen / totalLen) * 100) * 100) / 100;
    let framework = 'HTML/Static';
    let renderMode = 'SSR';
    let notice = '';
    let wafStatus: 'clean' | 'cloudflare_challenge' | 'rate_limited' | 'bot_detected' = 'clean';

    // WAF & Anti-Bot Detection
    if (lower.includes('just a moment...') || lower.includes('cf-browser-verification') || lower.includes('challenge-platform') || lower.includes('cloudflare ray id')) {
      wafStatus = 'cloudflare_challenge';
      notice = 'Cloudflare WAF Challenge detectado. A página requer desafio JS ou stealth headless.';
    } else if (lower.includes('datadome') || lower.includes('geo.captcha-delivery.com')) {
      wafStatus = 'bot_detected';
      notice = 'Proteção DataDome detectada.';
    } else if (lower.includes('recaptcha') || lower.includes('g-recaptcha')) {
      wafStatus = 'bot_detected';
      notice = 'reCAPTCHA detectado na página.';
    }

    if (lower.includes('__next_data__') || lower.includes('/_next/static/')) {
      framework = 'Next.js';
      renderMode = yieldRatio >= 2.0 ? 'SSR (Server Rendered)' : 'CSR (Client Hydration)';
    } else if (lower.includes('__nuxt__') || lower.includes('/_nuxt/')) {
      framework = 'Nuxt (Vue)';
      renderMode = yieldRatio >= 2.0 ? 'SSR (Server Rendered)' : 'CSR';
    } else if (lower.includes('data-reactroot') || lower.includes('id="root"') || lower.includes('id="app"')) {
      if (lower.includes('react')) {
        framework = 'React (SPA)';
      } else if (lower.includes('vue')) {
        framework = 'Vue (SPA)';
      } else {
        framework = 'SPA / Virtual DOM';
      }
      if (yieldRatio < 3.0 || textLen < 50) {
        renderMode = 'CSR (Client-Side Rendering)';
        if (!notice) notice = 'Página requer execução de JavaScript para renderizar conteúdo completo. Considere usar modo JS / Headless.';
      } else {
        renderMode = 'SSR / Pre-rendered';
      }
    } else if (lower.includes('wp-content') || lower.includes('wordpress')) {
      framework = 'WordPress (PHP)';
      renderMode = 'SSR';
    } else if (lower.includes('hx-get') || lower.includes('hx-post') || lower.includes('htmx')) {
      framework = 'HTMX / Hypermedia';
      renderMode = 'SSR + Dynamic Swapping';
    }

    return {
      framework,
      renderMode,
      encoding,
      contentYield: yieldRatio,
      notice: notice || undefined,
      wafStatus,
    };
  }

  /**
   * Extract JSON-LD, Next Data, and meta tags.
   */
  static extractStructuredData(rawHTML: string): Record<string, any> {
    const metadata: Record<string, any> = {};

    // 1. JSON-LD
    const jsonLdRegex = /<script\s+[^>]*type=["']application\/ld\+json["'][^>]*>([\s\S]*?)<\/script>/gi;
    const jsonLdEntries: any[] = [];
    let match: RegExpExecArray | null;
    while ((match = jsonLdRegex.exec(rawHTML)) !== null) {
      try {
        const parsed = JSON.parse(match[1].trim());
        jsonLdEntries.push(parsed);
      } catch {
        // invalid JSON-LD ignored
      }
    }
    if (jsonLdEntries.length > 0) {
      metadata['json_ld'] = jsonLdEntries.length === 1 ? jsonLdEntries[0] : jsonLdEntries;
    }

    // 2. Next.js Data Payload
    const nextDataMatch = rawHTML.match(/<script\s+[^>]*id=["']__NEXT_DATA__["'][^>]*>([\s\S]*?)<\/script>/i);
    if (nextDataMatch) {
      try {
        metadata['next_data'] = JSON.parse(nextDataMatch[1].trim());
      } catch {
        // ignore
      }
    }

    // 3. OpenGraph & Meta Tags
    const ogRegex = /<meta\s+[^>]*property=["']og:([^"']+)["']\s+content=["']([^"']*)["'][^>]*>/gi;
    const og: Record<string, string> = {};
    while ((match = ogRegex.exec(rawHTML)) !== null) {
      og[match[1]] = match[2];
    }
    if (Object.keys(og).length > 0) {
      metadata['open_graph'] = og;
    }

    return metadata;
  }

  /**
   * Parses HTML into a bounded DocumentModel.
   */
  static parse(rawHTML: string, baseURL: string, status = 200, bytes = 0): DocumentModel {
    const parser = new DOMParser();
    const doc = parser.parseFromString(rawHTML, 'text/html');
    return new DocumentModel(doc, rawHTML, baseURL, status, bytes || rawHTML.length);
  }
}
