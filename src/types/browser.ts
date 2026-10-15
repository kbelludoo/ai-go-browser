export interface Command {
  cmd: string;
  url?: string;
  ref?: string;
  selector?: string;
  value?: string;
  confirm?: boolean;
  js?: boolean;
}

export interface Memory {
  rss_mib: number;
  heap_mib: number;
}

export interface JSStats {
  executed: boolean;
  scripts_count: number;
  duration_ms: number;
  peak_mib?: number;
  output?: string;
}

export interface TechStack {
  framework: string;
  renderMode: string;
  encoding: string;
  contentYield: number; // percentage
  notice?: string;
  wafStatus?: 'clean' | 'cloudflare_challenge' | 'rate_limited' | 'bot_detected';
}

export interface Heading {
  level: number;
  text: string;
}

export interface Link {
  ref: string;
  label: string;
  url: string;
}

export interface Button {
  ref: string;
  id?: string;
  label: string;
  type?: string;
}

export interface ActionField {
  name: string;
  type: string;
  value?: string;
}

export interface Form {
  ref: string;
  action: string;
  method: string;
  fields: ActionField[];
  submit_label?: string;
}

export interface Page {
  url: string;
  title: string;
  description?: string;
  text: string;
  markdown?: string;
  headings: Heading[];
  links: Link[];
  buttons: Button[];
  forms: Form[];
  status: number;
  bytes: number;
  stack?: TechStack;
  metadata?: Record<string, any>;
  nextPageUrl?: string;
  readabilityScore?: number;
}

export interface ElementMatch {
  tag: string;
  id?: string;
  role?: string;
  text?: string;
  href?: string;
  name?: string;
  value?: string;
}

export interface Action {
  ref: string;
  label: string;
  kind: 'link' | 'form' | 'button' | 'api';
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'UNKNOWN';
  url?: string;
  source: string;
  evidence?: string;
  preview?: string;
  confidence: 'declared' | 'inferred' | 'unknown';
  requires_confirmation: boolean;
  executable: boolean;
  fields?: ActionField[];
}

export interface Resource {
  url: string;
  status: number;
  content_type: string;
  body?: string;
  data?: any;
  truncated?: boolean;
}

export interface Result {
  ok: boolean;
  error?: string;
  help?: string[];
  page?: Page;
  matches?: ElementMatch[];
  actions?: Action[];
  action?: Action;
  resource?: Resource;
  memory: Memory;
  duration_ms: number;
  js?: JSStats;
}

export interface BenchResult {
  name: string;
  url: string;
  htmlBytes: number;
  htmlTokens: number;
  mdBytes: number;
  mdTokens: number;
  ratio: number; // reduction percentage
  links: number;
  buttons: number;
  forms: number;
  headings: number;
  durationMs: number;
  rssMiB: number;
}

export interface ArenaModel {
  id: string;
  name: string;
  publicName: string;
  organization: string;
  category?: string;
  score?: number;
}

export interface ArenaChatEvaluation {
  id: string;
  prompt: string;
  modelA: string;
  modelB: string;
  responseA: string;
  responseB: string;
  recaptchaToken: string;
  timestamp: string;
}
