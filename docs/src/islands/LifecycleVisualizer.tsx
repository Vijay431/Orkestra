import { useState } from 'react';

type State = 'bk' | 'td' | 'ip' | 'dn' | 'bl' | 'cl';

interface StateMeta {
  id: State;
  label: string;
  full: string;
  color: string;
  x: number;
  y: number;
}

const STATES: StateMeta[] = [
  { id: 'bk', label: 'bk', full: 'backlog', color: '#6B7785', x: 80, y: 130 },
  { id: 'td', label: 'td', full: 'todo', color: '#FFE066', x: 220, y: 60 },
  { id: 'ip', label: 'ip', full: 'in_progress', color: '#00ADD8', x: 360, y: 130 },
  { id: 'bl', label: 'bl', full: 'blocked', color: '#ff6464', x: 360, y: 240 },
  { id: 'dn', label: 'dn', full: 'done', color: '#9BE564', x: 500, y: 60 },
  { id: 'cl', label: 'cl', full: 'cancelled', color: '#cccccc', x: 500, y: 200 },
];

interface Edge {
  from: State;
  to: State;
  via: string;
}

const EDGES: Edge[] = [
  { from: 'bk', to: 'td', via: 'refined' },
  { from: 'bk', to: 'ip', via: 'ticket_claim' },
  { from: 'td', to: 'ip', via: 'ticket_claim' },
  { from: 'ip', to: 'dn', via: 'ticket_update s=dn' },
  { from: 'ip', to: 'bl', via: 'blocked externally' },
  { from: 'bl', to: 'ip', via: 'unblocked' },
  { from: 'bk', to: 'cl', via: 'ticket_update s=cl' },
  { from: 'ip', to: 'cl', via: 'ticket_update s=cl' },
];

function pos(id: State) {
  return STATES.find((s) => s.id === id)!;
}

export default function LifecycleVisualizer() {
  const [active, setActive] = useState<State | null>('bk');
  const outgoing = active ? EDGES.filter((e) => e.from === active) : [];

  return (
    <div className="ork-lifecycle">
      <svg viewBox="0 0 600 320" className="ork-lifecycle__svg" role="group" aria-label="Ticket lifecycle — interactive state diagram. Use arrow keys to navigate states.">
        <defs>
          <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" fill="#3a4a60" />
          </marker>
          <marker id="arrow-active" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" fill="#00ADD8" />
          </marker>
        </defs>

        {EDGES.map((e, i) => {
          const a = pos(e.from);
          const b = pos(e.to);
          const isActive = active === e.from;
          return (
            <line
              key={i}
              x1={a.x}
              y1={a.y}
              x2={b.x}
              y2={b.y}
              stroke={isActive ? '#00ADD8' : '#3a4a60'}
              strokeWidth={isActive ? 2 : 1}
              markerEnd={isActive ? 'url(#arrow-active)' : 'url(#arrow)'}
              opacity={isActive || !active ? 1 : 0.35}
            />
          );
        })}

        {STATES.map((s) => (
          <g
            key={s.id}
            transform={`translate(${s.x},${s.y})`}
            className="ork-lifecycle__node"
            data-active={active === s.id}
            aria-current={active === s.id ? 'step' : undefined}
            onClick={() => setActive(s.id)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                setActive(s.id);
              } else if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
                e.preventDefault();
                const idx = STATES.findIndex((st) => st.id === s.id);
                setActive(STATES[(idx + 1) % STATES.length].id);
              } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
                e.preventDefault();
                const idx = STATES.findIndex((st) => st.id === s.id);
                setActive(STATES[(idx - 1 + STATES.length) % STATES.length].id);
              }
            }}
          >
            <circle r="28" fill={s.color} fillOpacity={active === s.id ? 1 : 0.15} stroke={s.color} strokeWidth="2" />
            <text textAnchor="middle" y="6" fontFamily="JetBrains Mono, monospace" fontSize="14" fontWeight="600" fill={active === s.id ? '#0B0F14' : '#E6EDF3'}>
              {s.label}
            </text>
          </g>
        ))}
      </svg>

      <aside className="ork-lifecycle__detail" aria-live="polite">
        {active ? (
          <>
            <h4>
              <code>{active}</code> — {pos(active).full}
            </h4>
            {outgoing.length === 0 ? (
              <p>Terminal-ish. Move on by archiving.</p>
            ) : (
              <ul>
                {outgoing.map((e) => (
                  <li key={`${e.from}-${e.to}`}>
                    <span className="ork-lifecycle__via">{e.via}</span>
                    <span className="ork-lifecycle__arrow">→</span>
                    <code>{e.to}</code> <span className="ork-lifecycle__sub">({pos(e.to).full})</span>
                  </li>
                ))}
              </ul>
            )}
          </>
        ) : (
          <p>Click a state to inspect transitions.</p>
        )}
      </aside>
    </div>
  );
}
