// Tiny remark plugin: convert ```mermaid fences into a raw HTML <pre class="mermaid">
// block so client-side mermaid.run() can pick them up. Keeps build lightweight
// (no Playwright, no SSR mermaid).
import { visit } from 'unist-util-visit';

export default function remarkMermaid() {
  return (tree) => {
    visit(tree, 'code', (node, index, parent) => {
      if (node.lang !== 'mermaid' || !parent || typeof index !== 'number') return;
      const value = node.value || '';
      parent.children[index] = {
        type: 'html',
        value: `<pre class="mermaid not-content">${escapeHtml(value)}</pre>`,
      };
    });
  };
}

function escapeHtml(s) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}
