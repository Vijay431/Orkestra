import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import react from '@astrojs/react';
import remarkMermaid from './src/lib/remark-mermaid.mjs';

export default defineConfig({
	site: 'https://vijay431.github.io',
	base: '/Orkestra',
	trailingSlash: 'ignore',
	markdown: {
		remarkPlugins: [remarkMermaid],
	},
	integrations: [
		react(),
		starlight({
			title: 'Orkestra',
			description: 'Self-hosted MCP ticket server for autonomous LLM agents — no cloud, no rate limits.',
			logo: { src: './src/assets/logo.svg', replacesTitle: false },
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/Vijay431/Orkestra' }],
			customCss: ['./src/styles/global.css'],
			head: [
				{
					tag: 'link',
					attrs: { rel: 'preconnect', href: 'https://cdn.jsdelivr.net' },
				},
				{
					tag: 'script',
					attrs: { type: 'module' },
					content: `
            import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10.9.1/dist/mermaid.esm.min.mjs';
            mermaid.initialize({
              startOnLoad: false,
              theme: 'base',
              themeVariables: {
                darkMode: true,
                background: '#0B0F14',
                primaryColor: '#11161D',
                primaryTextColor: '#E6EDF3',
                primaryBorderColor: '#243042',
                lineColor: '#3a4a60',
                secondaryColor: '#0E2A33',
                tertiaryColor: '#1a2330',
                fontFamily: 'JetBrains Mono, ui-monospace, monospace',
              },
            });
            const run = () => mermaid.run({ querySelector: 'pre.mermaid' });
            if (document.readyState === 'loading') {
              document.addEventListener('DOMContentLoaded', run);
            } else { run(); }
            document.addEventListener('astro:page-load', run);
          `,
				},
			],
			sidebar: [
				{
					label: 'Get Started',
					items: [
						{ label: 'Quickstart', slug: 'quickstart' },
						{ label: 'Architecture', slug: 'architecture' },
						{ label: 'Data Safety', slug: 'data-safety' },
					],
				},
				{
					label: 'Concepts',
					items: [
						{ label: 'TOON Format', slug: 'toon' },
						{ label: 'Workflows', slug: 'workflows' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'All Tools', slug: 'tools' },
						{ label: 'API Guide', slug: 'tools/api-guide' },
						{ label: 'Examples', slug: 'tools/examples' },
						{ label: 'Troubleshooting', slug: 'tools/troubleshooting' },
					],
				},
				{
					label: 'Project',
					items: [
						{ label: 'Contributing', slug: 'contributing' },
						{ label: 'Changelog', slug: 'changelog' },
						{ label: 'Acknowledgments', slug: 'acknowledgments' },
					],
				},
			],
		}),
	],
});
