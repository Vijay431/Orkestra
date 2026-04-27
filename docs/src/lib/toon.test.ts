// Round-trip fixtures mirroring internal/toon/encoder_test.go. If the Go
// encoder evolves these need to be re-synced — the parity matters because the
// playground promises "same output as the server."

import { describe, expect, it } from 'vitest';
import {
  encode,
  encodeBoard,
  encodeError,
  encodeOK,
  encodeSummary,
  etagOf,
  estimateTokens,
  type Ticket,
} from './toon';

const baseTicket = (): Ticket => ({
  id: 'test-001',
  title: 'simple',
  status: 'bk',
  priority: 'm',
  type: 'tsk',
  execMode: 'par',
  createdAt: new Date(Date.UTC(2024, 0, 15, 10, 0, 0)),
  updatedAt: new Date(Date.UTC(2024, 0, 15, 10, 0, 0)),
});

describe('encode', () => {
  it('prefixes TOON/1', () => {
    expect(encode(baseTicket()).startsWith('TOON/1 ')).toBe(true);
  });

  it('escapes special chars in title', () => {
    const cases: Array<[string, string]> = [
      ['say "hello"', 't:"say \\"hello\\""'],
      ['line1\nline2', 't:"line1\\nline2"'],
      ['back\\slash', 't:"back\\\\slash"'],
      ['has:colon', 't:"has:colon"'],
      ['has,comma', 't:"has,comma"'],
      ['has{brace}', 't:"has{brace}"'],
      ['nospace', 't:nospace'],
    ];
    for (const [title, want] of cases) {
      const t = baseTicket();
      t.title = title;
      expect(encode(t)).toContain(want);
    }
  });

  it('omits empty children array', () => {
    const t = baseTicket();
    t.children = [];
    expect(encode(t).includes('ch:')).toBe(false);
  });

  it('emits ord:0 for zero exec_order', () => {
    const t = baseTicket();
    t.execOrder = 0;
    expect(encode(t)).toContain('ord:0');
  });

  it('handles 50 labels', () => {
    const t = baseTicket();
    t.labels = Array(50).fill('lbl');
    const out = encode(t);
    expect(out).toContain('lbl:[');
    expect((out.match(/lbl/g) ?? []).length).toBeGreaterThanOrEqual(50);
  });

  it('omits parallel exec_mode', () => {
    const t = baseTicket();
    t.execMode = 'par';
    expect(encode(t).includes('em:')).toBe(false);
  });

  it('includes sequential exec_mode', () => {
    const t = baseTicket();
    t.execMode = 'seq';
    expect(encode(t)).toContain('em:seq');
  });
});

describe('errors and OK', () => {
  it('encodeError shape', () => {
    const out = encodeError('not_found', 'test-001 does not exist');
    expect(out.startsWith('TOON/1 ')).toBe(true);
    expect(out).toContain('ERR{');
    expect(out).toContain('code:not_found');
  });

  it('encodeOK exact', () => {
    expect(encodeOK()).toBe('TOON/1 {ok:true}');
  });
});

describe('board', () => {
  it('orders bk before ip', () => {
    const bk = baseTicket();
    bk.title = 'backlog item';
    const ip = baseTicket();
    ip.id = 'test-002';
    ip.title = 'active item';
    ip.status = 'ip';
    const out = encodeBoard({ bk: [bk], ip: [ip] });
    expect(out.startsWith('TOON/1 BOARD{')).toBe(true);
    expect(out.indexOf('bk:')).toBeLessThan(out.indexOf('ip:'));
  });
});

describe('comments and links', () => {
  it('emits comment envelope', () => {
    const t = baseTicket();
    t.comments = [
      { author: 'llm', body: 'started work', createdAt: new Date(Date.UTC(2024, 0, 15, 10, 0, 0)) },
    ];
    const out = encode(t);
    expect(out).toContain('cmt:[');
    expect(out).toContain('a:llm');
  });

  it('emits link envelope', () => {
    const t = baseTicket();
    t.links = [{ fromId: 'test-001', toId: 'test-002', linkType: 'blk' }];
    const out = encode(t);
    expect(out).toContain('lnk:[');
    expect(out).toContain('k:blk');
  });
});

describe('summary mode', () => {
  it('drops description and comments', () => {
    const t = baseTicket();
    t.description = 'detailed description';
    t.comments = [{ author: 'llm', body: 'comment', createdAt: new Date() }];
    const out = encodeSummary(t);
    expect(out.includes('detailed description')).toBe(false);
    expect(out.includes('cmt:')).toBe(false);
  });
});

describe('etag', () => {
  it('contains the date', () => {
    expect(etagOf(baseTicket())).toContain('2024-01-15');
  });
});

describe('token estimator', () => {
  it('roughly halves with TOON', () => {
    const ticket = baseTicket();
    ticket.labels = ['auth', 'security'];
    const json = JSON.stringify({
      id: ticket.id,
      title: 'Fix auth bug',
      status: 'in_progress',
      priority: 'high',
      type: 'bug',
      labels: ticket.labels,
      created_at: '2024-01-15T08:00:00Z',
      updated_at: '2024-01-15T10:00:00Z',
    });
    const toon = encode({ ...ticket, title: 'Fix auth bug', status: 'ip', priority: 'h', type: 'bug' });
    expect(estimateTokens(toon)).toBeLessThan(estimateTokens(json));
  });
});
