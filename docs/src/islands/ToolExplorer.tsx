import { useMemo, useState } from 'react';
import toolsData from '../data/tools.json';

interface Param {
  name: string;
  type: string;
  required?: boolean;
  default?: string;
  notes?: string;
}

interface Tool {
  name: string;
  category: 'lifecycle' | 'discovery' | 'collaboration';
  summary: string;
  required?: string[];
  params: Param[];
  returns: string;
  errors?: string[];
  example?: string;
}

const TOOLS = toolsData as Tool[];

const CATEGORIES: Array<{ id: 'all' | Tool['category']; label: string; tint: string }> = [
  { id: 'all', label: `All ${TOOLS.length}`, tint: 'var(--ork-text-dim)' },
  { id: 'lifecycle', label: 'Lifecycle', tint: 'var(--ork-teal)' },
  { id: 'discovery', label: 'Discovery', tint: 'var(--ork-green)' },
  { id: 'collaboration', label: 'Collaboration', tint: 'var(--ork-pink)' },
];

export default function ToolExplorer() {
  const [filter, setFilter] = useState<'all' | Tool['category']>('all');
  const [query, setQuery] = useState('');
  const [openName, setOpenName] = useState<string | null>(null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return TOOLS.filter((t) => {
      if (filter !== 'all' && t.category !== filter) return false;
      if (!q) return true;
      return (
        t.name.toLowerCase().includes(q) ||
        t.summary.toLowerCase().includes(q) ||
        t.params.some((p) => p.name.toLowerCase().includes(q))
      );
    });
  }, [filter, query]);

  return (
    <div className="ork-explorer">
      <div className="ork-explorer__controls">
        <div className="ork-explorer__chips" role="group" aria-label="Filter by tool category">
          {CATEGORIES.map((c) => (
            <button
              key={c.id}
              type="button"
              aria-pressed={filter === c.id}
              className="ork-explorer__chip"
              data-active={filter === c.id}
              style={{ ['--chip-tint' as string]: c.tint }}
              onClick={() => setFilter(c.id)}
            >
              {c.label}
            </button>
          ))}
        </div>
        <input
          className="ork-explorer__search"
          type="search"
          placeholder="Filter tools or params…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          aria-label="Search tools"
        />
      </div>

      <ul className="ork-explorer__grid">
        {filtered.map((t) => (
          <li key={t.name} className="ork-explorer__card" data-category={t.category}>
            <button
              className="ork-explorer__card-header"
              aria-expanded={openName === t.name}
              aria-controls={`tool-detail-${t.name}`}
              onClick={() => setOpenName(openName === t.name ? null : t.name)}
            >
              <code className="ork-explorer__name">{t.name}</code>
              <span className="ork-explorer__category">{t.category}</span>
            </button>
            <p className="ork-explorer__summary">{t.summary}</p>
            {openName === t.name && (
              <div className="ork-explorer__detail" id={`tool-detail-${t.name}`}>
                <h4>Parameters</h4>
                <table>
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Type</th>
                      <th>Default</th>
                      <th>Notes</th>
                    </tr>
                  </thead>
                  <tbody>
                    {t.params.map((p) => (
                      <tr key={p.name}>
                        <td>
                          <code>{p.name}</code>
                          {p.required && <span className="ork-explorer__req"> *</span>}
                        </td>
                        <td>{p.type}</td>
                        <td>{p.default ?? '—'}</td>
                        <td>{p.notes ?? ''}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <h4>Returns</h4>
                <code className="ork-explorer__returns">{t.returns}</code>
                {t.errors && t.errors.length > 0 && (
                  <>
                    <h4>Errors</h4>
                    <div className="ork-explorer__errors">
                      {t.errors.map((e) => (
                        <code key={e}>{e}</code>
                      ))}
                    </div>
                  </>
                )}
                {t.example && (
                  <>
                    <h4>Example</h4>
                    <pre className="ork-explorer__example">{t.example}</pre>
                  </>
                )}
              </div>
            )}
          </li>
        ))}
      </ul>

      {filtered.length === 0 && (
        <p className="ork-explorer__empty">No tools match — try clearing filters.</p>
      )}
    </div>
  );
}
