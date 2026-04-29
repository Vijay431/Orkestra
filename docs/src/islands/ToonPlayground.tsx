import { useMemo, useState } from 'react';
import { encodeAny, estimateTokens } from '../lib/toon';

const SAMPLE = JSON.stringify(
  {
    id: 'myapp-001',
    title: 'Fix auth bug',
    status: 'in_progress',
    priority: 'high',
    type: 'bug',
    labels: ['auth', 'security'],
    created_at: '2024-01-15T08:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
  null,
  2,
);

interface Props {
  variant?: 'compact' | 'full';
}

export default function ToonPlayground({ variant = 'full' }: Props) {
  const [json, setJson] = useState(SAMPLE);

  const { toon, error, jsonTokens, toonTokens, savings } = useMemo(() => {
    let parsed: unknown;
    let err: string | null = null;
    try {
      parsed = JSON.parse(json);
    } catch (e) {
      err = (e as Error).message;
    }
    const out = err ? '' : encodeAny(parsed);
    const jt = estimateTokens(json.replace(/\s+/g, ' '));
    const tt = estimateTokens(out);
    const sv = jt > 0 && tt > 0 ? Math.round((1 - tt / jt) * 100) : 0;
    return { toon: out, error: err, jsonTokens: jt, toonTokens: tt, savings: sv };
  }, [json]);

  return (
    <div className={`ork-playground ork-playground--${variant}`}>
      <div className="ork-playground__pane">
        <header aria-live="polite">
          <span className="ork-playground__label">JSON</span>
          <span className="ork-playground__count">~{jsonTokens} tokens</span>
        </header>
        <textarea
          className="ork-playground__editor"
          value={json}
          onChange={(e) => setJson(e.target.value)}
          spellCheck={false}
          aria-label="JSON input"
        />
      </div>

      <div className="ork-playground__divider">
        <div className="ork-playground__badge" data-positive={savings > 0}>
          {savings > 0 ? `−${savings}%` : savings < 0 ? `+${-savings}%` : '0%'}
        </div>
        <span className="sr-only">TOON format uses approximately {savings > 0 ? savings : 0}% fewer tokens than JSON</span>
      </div>

      <div className="ork-playground__pane">
        <header aria-live="polite">
          <span className="ork-playground__label">TOON</span>
          <span className="ork-playground__count">~{toonTokens} tokens</span>
        </header>
        <pre className="ork-playground__output" aria-label="TOON output">
          {error ? <span className="ork-playground__error" role="alert">{error}</span> : toon}
        </pre>
      </div>
    </div>
  );
}
