// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';

// https://astro.build/config
export default defineConfig({
  site: 'https://albertocavalcante.github.io',
  base: '/bazelle/',
  integrations: [
    starlight({
      title: 'Bazelle',
      description: 'Polyglot Gazelle CLI - a unified BUILD file generator with multiple language extensions',
      logo: {
        src: './src/assets/logo.svg',
        replacesTitle: false,
      },
      social: {
        github: 'https://github.com/albertocavalcante/bazelle',
      },
      editLink: {
        baseUrl: 'https://github.com/albertocavalcante/bazelle/edit/main/docs/',
      },
      head: [
        // OpenGraph / Social sharing
        {
          tag: 'meta',
          attrs: { property: 'og:image', content: 'https://albertocavalcante.github.io/bazelle/og-image.png' },
        },
        {
          tag: 'meta',
          attrs: { property: 'og:type', content: 'website' },
        },
        {
          tag: 'meta',
          attrs: { name: 'twitter:card', content: 'summary_large_image' },
        },
      ],
      customCss: [
        './src/styles/custom.css',
      ],
      plugins: [
        starlightLlmsTxt({
          projectName: 'Bazelle',
          description: 'Bazelle is a polyglot Gazelle CLI - a unified BUILD file generator for Bazel with support for Go, Kotlin, Python, and C/C++.',
          promote: ['index', 'getting-started', 'installation', 'configuration'],
          demote: ['faq', 'troubleshooting', 'contributing'],
        }),
      ],
      components: {
        PageTitle: './src/components/PageTitle.astro',
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Introduction', slug: '' },
            { label: 'Why Bazelle?', slug: 'why-bazelle', badge: { text: 'New', variant: 'tip' } },
            { label: 'Quick Start', slug: 'getting-started' },
            { label: 'Installation', slug: 'installation' },
            { label: 'Configuration', slug: 'configuration' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'Overview', slug: 'guides' },
            { label: 'Monorepo Setup', slug: 'guides/monorepo-setup' },
            { label: 'CI/CD Integration', slug: 'guides/ci-integration' },
          ],
        },
        {
          label: 'Languages',
          items: [
            { label: 'Overview', slug: 'languages' },
            { label: 'Go', slug: 'languages/go', badge: { text: 'Core', variant: 'success' } },
            { label: 'Kotlin', slug: 'languages/kotlin', badge: { text: 'Own', variant: 'note' } },
            { label: 'Python', slug: 'languages/python', badge: { text: 'Own', variant: 'note' } },
            { label: 'C/C++', slug: 'languages/cpp', badge: { text: 'Ext', variant: 'caution' } },
          ],
        },
        {
          label: 'CLI Reference',
          items: [
            { label: 'Overview', slug: 'cli' },
            { label: 'update', slug: 'cli/update' },
            { label: 'fix', slug: 'cli/fix' },
            { label: 'init', slug: 'cli/init' },
            { label: 'watch', slug: 'cli/watch' },
          ],
        },
        {
          label: 'Resources',
          items: [
            { label: 'FAQ', slug: 'faq' },
            { label: 'Troubleshooting', slug: 'troubleshooting' },
          ],
        },
        {
          label: 'Community',
          items: [
            { label: 'Contributing', slug: 'contributing' },
          ],
        },
      ],
    }),
  ],
});
