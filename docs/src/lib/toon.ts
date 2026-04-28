// SPDX-License-Identifier: MIT
//
// TypeScript port of internal/toon/encoder.go.
// Used for the in-browser TOON Playground; round-trip parity with the Go
// encoder is enforced by toon.test.ts.

export type Status = 'bk' | 'td' | 'ip' | 'dn' | 'bl' | 'cl';
export type Priority = 'cr' | 'h' | 'm' | 'l';
export type TicketType = 'bug' | 'ft' | 'tsk' | 'ep' | 'chr';
export type ExecMode = 'par' | 'seq';
export type LinkKind = 'blk' | 'rel' | 'dup';

export interface Comment {
  author: string;
  body: string;
  createdAt: Date | string;
}

export interface Link {
  fromId: string;
  toId: string;
  linkType: LinkKind;
}

export interface Ticket {
  id: string;
  title: string;
  status: Status;
  priority: Priority;
  type: TicketType;
  execMode?: ExecMode;
  execOrder?: number | null;
  labels?: string[];
  parentId?: string;
  children?: string[];
  description?: string;
  assignee?: string;
  comments?: Comment[];
  links?: Link[];
  createdAt: Date | string;
  updatedAt: Date | string;
}

export const VERSION = 'TOON/1 ';

const STATUS_ORDER: Status[] = ['bk', 'td', 'ip', 'bl', 'cl', 'dn'];

export function encode(t: Ticket): string {
  return VERSION + encodeTicket(t, false);
}

export function encodeSummary(t: Ticket): string {
  return encodeTicket(t, true);
}

export function encodeError(code: string, msg: string): string {
  return VERSION + `ERR{code:${code},msg:${escapeString(msg)}}`;
}

export function encodeOK(): string {
  return VERSION + '{ok:true}';
}

export function encodeBoard(board: Partial<Record<Status, Ticket[]>>): string {
  const parts: string[] = [];
  for (const status of STATUS_ORDER) {
    const tickets = board[status];
    if (!tickets || tickets.length === 0) continue;
    parts.push(`${status}:[${tickets.map((t) => encodeTicket(t, true)).join(',')}]`);
  }
  return VERSION + `BOARD{${parts.join(',')}}`;
}

export function etagOf(t: Ticket): string {
  // JS Date truncates to ms; pass strings through verbatim to preserve Go's ns precision.
  if (typeof t.updatedAt === 'string') return t.updatedAt;
  return formatRFC3339Nano(t.updatedAt);
}

function encodeTicket(t: Ticket, summary: boolean): string {
  const parts: string[] = [];
  parts.push(`id:${t.id}`);
  parts.push(`t:${escapeString(t.title)}`);
  parts.push(`s:${t.status}`);
  parts.push(`p:${t.priority}`);
  parts.push(`typ:${t.type}`);

  if (t.execMode && t.execMode !== 'par') parts.push(`em:${t.execMode}`);
  if (t.execOrder !== undefined && t.execOrder !== null) parts.push(`ord:${t.execOrder}`);
  if (t.labels && t.labels.length > 0) {
    parts.push(`lbl:[${t.labels.map(escapeString).join(',')}]`);
  }
  if (t.parentId) parts.push(`par:${t.parentId}`);
  if (t.children && t.children.length > 0) parts.push(`ch:[${t.children.join(',')}]`);

  if (!summary) {
    if (t.description) parts.push(`d:${escapeString(t.description)}`);
    if (t.assignee) parts.push(`as:${escapeString(t.assignee)}`);
    if (t.comments && t.comments.length > 0) {
      const cmt = t.comments
        .map(
          (c) =>
            `C{a:${escapeString(c.author)},t:${escapeString(c.body)},ts:${formatShortTs(toDate(c.createdAt))}}`,
        )
        .join(',');
      parts.push(`cmt:[${cmt}]`);
    }
    if (t.links && t.links.length > 0) {
      const lnk = t.links
        .map((l) => `L{f:${l.fromId},t:${l.toId},k:${l.linkType}}`)
        .join(',');
      parts.push(`lnk:[${lnk}]`);
    }
  }

  parts.push(`ca:${formatDate(toDate(t.createdAt))}`);
  parts.push(`ua:${etagOf(t)}`);
  return `T{${parts.join(',')}}`;
}

function escapeString(s: string): string {
  if (s === '') return '""';
  let needs = false;
  for (const ch of s) {
    if (
      ch === ' ' ||
      ch === ':' ||
      ch === '{' ||
      ch === '}' ||
      ch === ',' ||
      ch === '"' ||
      ch === '\\' ||
      ch === '\n'
    ) {
      needs = true;
      break;
    }
  }
  if (!needs) return s;
  let out = '"';
  for (const ch of s) {
    if (ch === '"') out += '\\"';
    else if (ch === '\\') out += '\\\\';
    else if (ch === '\n') out += '\\n';
    else out += ch;
  }
  out += '"';
  return out;
}

function toDate(v: Date | string): Date {
  return v instanceof Date ? v : new Date(v);
}

function pad(n: number, len = 2): string {
  return n.toString().padStart(len, '0');
}

function formatDate(d: Date): string {
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`;
}

function formatShortTs(d: Date): string {
  return `${formatDate(d)}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

// Mirrors Go's "2006-01-02T15:04:05.999999999Z" — trailing-zero trimming on
// fractional seconds.
function formatRFC3339Nano(d: Date): string {
  const base = `${formatDate(d)}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}`;
  const ms = d.getUTCMilliseconds();
  if (ms === 0) return `${base}Z`;
  // Trim trailing zeros from milliseconds (Go's .999... behavior).
  const frac = pad(ms, 3).replace(/0+$/, '');
  return frac ? `${base}.${frac}Z` : `${base}Z`;
}

// Rough GPT-style token estimator: 1 token ≈ 4 chars. Good enough for the
// playground's "savings" badge.
export function estimateTokens(s: string): number {
  if (!s) return 0;
  return Math.max(1, Math.ceil(s.length / 4));
}

// Encode any JSON-ish object as TOON if it looks like a ticket; otherwise
// fall through to a "best-effort" rendering used by the playground.
export function encodeAny(value: unknown): string {
  if (Array.isArray(value)) {
    const tickets = value.filter(looksLikeTicket).map(normalizeTicket);
    if (tickets.length === value.length && tickets.length > 0) {
      return VERSION + `[${tickets.map((t) => encodeTicket(t, true)).join(',')}]`;
    }
  }
  if (value && typeof value === 'object' && looksLikeTicket(value)) {
    return encode(normalizeTicket(value));
  }
  // Generic fallback: shorten field names and drop quotes where possible.
  return VERSION + genericEncode(value);
}

const STATUS_ALIAS: Record<string, Status> = {
  bk: 'bk', backlog: 'bk',
  td: 'td', todo: 'td',
  ip: 'ip', in_progress: 'ip', 'in-progress': 'ip', inprogress: 'ip',
  dn: 'dn', done: 'dn',
  bl: 'bl', blocked: 'bl',
  cl: 'cl', cancelled: 'cl', canceled: 'cl',
};
const PRIORITY_ALIAS: Record<string, Priority> = {
  cr: 'cr', critical: 'cr',
  h: 'h', high: 'h',
  m: 'm', medium: 'm', med: 'm',
  l: 'l', low: 'l',
};
const TYPE_ALIAS: Record<string, TicketType> = {
  bug: 'bug',
  ft: 'ft', feature: 'ft',
  tsk: 'tsk', task: 'tsk',
  ep: 'ep', epic: 'ep',
  chr: 'chr', chore: 'chr',
};
const EXEC_ALIAS: Record<string, ExecMode> = {
  par: 'par', parallel: 'par',
  seq: 'seq', sequential: 'seq',
};

function aliasOr<T extends string>(map: Record<string, T>, v: unknown, fallback: T): T {
  if (typeof v !== 'string') return fallback;
  return map[v.toLowerCase()] ?? fallback;
}

function looksLikeTicket(v: unknown): v is Record<string, unknown> {
  if (!v || typeof v !== 'object') return false;
  const o = v as Record<string, unknown>;
  return typeof o.id === 'string' && (typeof o.title === 'string' || typeof o.t === 'string');
}

function normalizeTicket(o: Record<string, unknown> | unknown): Ticket {
  const r = o as Record<string, unknown>;
  return {
    id: String(r.id ?? ''),
    title: String(r.title ?? r.t ?? ''),
    status: aliasOr(STATUS_ALIAS, r.status ?? r.s, 'bk'),
    priority: aliasOr(PRIORITY_ALIAS, r.priority ?? r.p, 'm'),
    type: aliasOr(TYPE_ALIAS, r.type ?? r.typ, 'tsk'),
    execMode: (r.exec_mode ?? r.em) !== undefined
      ? aliasOr(EXEC_ALIAS, r.exec_mode ?? r.em, 'par')
      : undefined,
    execOrder: (r.exec_order ?? r.ord) as number | undefined,
    labels: (r.labels ?? r.lbl) as string[] | undefined,
    parentId: (r.parent_id ?? r.par) as string | undefined,
    children: (r.children ?? r.ch) as string[] | undefined,
    description: (r.description ?? r.d) as string | undefined,
    assignee: (r.assignee ?? r.as) as string | undefined,
    createdAt: (r.created_at ?? r.ca ?? new Date().toISOString()) as string,
    updatedAt: (r.updated_at ?? r.ua ?? new Date().toISOString()) as string,
  };
}

function genericEncode(v: unknown): string {
  if (v === null) return 'null';
  if (typeof v === 'string') return escapeString(v);
  if (typeof v === 'number' || typeof v === 'boolean') return String(v);
  if (Array.isArray(v)) return `[${v.map(genericEncode).join(',')}]`;
  if (typeof v === 'object') {
    const entries = Object.entries(v as Record<string, unknown>).map(
      ([k, val]) => `${k}:${genericEncode(val)}`,
    );
    return `{${entries.join(',')}}`;
  }
  return String(v);
}
