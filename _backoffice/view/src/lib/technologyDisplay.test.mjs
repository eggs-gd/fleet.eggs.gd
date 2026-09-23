import assert from 'node:assert/strict';
import test from 'node:test';

import {
  displayTechnologyGroups,
  displayTechnologyTags,
  itemTechnologyView,
  normalizeTechnologyTag,
  repoWebUrl,
  repositoryTechnologyView,
  technologyIconMap,
  technologyTagDisplay
} from './technologyDisplay.js';

test('repoWebUrl normalizes scp-style and full ssh:// remotes to https, and leaves https alone', () => {
  assert.equal(repoWebUrl('git@github.com:eggs-gd/core.eggs.gd.git'), 'https://github.com/eggs-gd/core.eggs.gd');
  // Full ssh:// remotes (with or without an explicit port) used to be mangled
  // because the scp-style branch ran first and read "ssh" itself as the host.
  assert.equal(repoWebUrl('ssh://git@example.com:2222/org/repo.git'), 'https://example.com/org/repo');
  assert.equal(repoWebUrl('ssh://git@example.com/org/repo'), 'https://example.com/org/repo');
  assert.equal(repoWebUrl('https://github.com/org/repo.git'), 'https://github.com/org/repo');
  assert.equal(repoWebUrl(''), '');
  assert.equal(repoWebUrl(null), '');
});

test('normalizes scanner provenance prefixes and obvious aliases', () => {
  assert.equal(normalizeTechnologyTag('raw javascript'), 'javascript');
  assert.equal(normalizeTechnologyTag('eff:typescript'), 'typescript');
  assert.equal(normalizeTechnologyTag('effective_svelte'), 'svelte');
  assert.equal(normalizeTechnologyTag('detected-nodejs'), 'node');
  assert.equal(normalizeTechnologyTag('Golang'), 'go');
  assert.equal(normalizeTechnologyTag('raw .NET'), 'dotnet');
  assert.equal(normalizeTechnologyTag('eff c#'), 'csharp');
});

test('technology tag display maps known tags to SVG icons with full titles', () => {
  const javascript = technologyTagDisplay('raw javascript');
  assert.equal(javascript.name, 'javascript');
  assert.equal(javascript.title, 'JavaScript');
  assert.equal(javascript.known, true);
  assert.equal(javascript.label, '');
  assert.equal(javascript.glyph, '');
  assert.match(javascript.svg, /<rect[^>]*fill="#F7DF1E"/);
  assert.doesNotMatch(javascript.svg, />\s*JS\s*</);

  const node = technologyTagDisplay('eff node.js');
  assert.equal(node.name, 'node');
  assert.equal(node.title, 'Node.js');
  assert.equal(node.known, true);
  assert.match(node.svg, /<path/);

  const dockerCompose = technologyTagDisplay('docker compose');
  assert.equal(dockerCompose.name, 'docker-compose');
  assert.equal(dockerCompose.title, 'Docker Compose');
  assert.equal(dockerCompose.known, true);
  assert.ok(dockerCompose.svg.includes('path'));

  const react = technologyTagDisplay('react');
  assert.equal(react.title, 'React');
  assert.match(react.svg, /ellipse/);
  assert.equal(react.label, '');

  for (const [name, mapped] of Object.entries(technologyIconMap)) {
    assert.ok(mapped.svg.includes('<'), `${name} should provide SVG markup`);
    assert.ok(mapped.title, `${name} should provide a full title`);
  }
});

test('technology tag display keeps unknown tags as compact fallback chips', () => {
  assert.deepEqual(technologyTagDisplay('raw custom-profile-label'), {
    name: 'custom-profile-label',
    glyph: '',
    label: 'cupr',
    title: 'Custom Profile Label',
    known: false,
    svg: ''
  });

  const limited = displayTechnologyTags(['typescript', 'custom-profile-label', 'python'], 2);
  assert.equal(limited.hidden, 1);
  assert.equal(limited.visible[0].name, 'typescript');
  assert.ok(limited.visible[0].svg.includes('rect'));
  assert.equal(limited.visible[0].label, '');
  assert.equal(limited.visible[1].name, 'custom-profile-label');
  assert.equal(limited.visible[1].known, false);
  assert.equal(limited.visible[1].label, 'cupr');
  assert.equal(limited.visible[1].svg, '');
});

test('technology groups flatten variants into one overflow-aware row', () => {
  const shown = displayTechnologyGroups(
    [
      { tags: ['typescript', 'go'], variant: 'primary' },
      { tags: ['npm', 'typescript'], variant: 'tooling' },
      { tags: ['frontend'], variant: 'capability' }
    ],
    3
  );

  assert.equal(shown.visible.length, 3);
  assert.equal(shown.hidden, 1);
  assert.equal(shown.visible[0].tag.name, 'typescript');
  assert.equal(shown.visible[0].variant, 'primary');
  assert.equal(shown.visible[1].tag.name, 'go');
  assert.equal(shown.visible[1].variant, 'primary');
  assert.equal(shown.visible[2].tag.name, 'npm');
  assert.equal(shown.visible[2].variant, 'tooling');
});

test('project technology view separates primary technology, tooling, and capabilities', () => {
  const project = {
    technology: {
      effective_tags: ['eff typescript', 'frontend', 'npm', 'custom-signal'],
      detected_tags: ['raw javascript', 'backend', 'vite'],
      repositories: [
        {
          effective: {
            languages: ['eff typescript'],
            frameworks: ['effective:svelte'],
            runtimes: ['nodejs'],
            tooling: ['npm'],
            capabilities: ['frontend']
          },
          detected: {
            languages: ['raw javascript', 'typescript'],
            frameworks: [],
            runtimes: [],
            tooling: ['vite'],
            capabilities: ['backend']
          },
          effective_tags: ['typescript', 'svelte', 'node', 'npm', 'frontend'],
          detected_tags: ['javascript', 'typescript', 'vite', 'backend']
        }
      ]
    }
  };

  const view = itemTechnologyView(project);

  assert.deepEqual(view.primary, ['typescript', 'svelte', 'node', 'javascript']);
  assert.deepEqual(view.tooling, ['npm', 'vite']);
  assert.deepEqual(view.capabilities, ['frontend', 'backend']);
  assert.equal(view.primary.includes('custom-signal'), false);
});

test('flat fallback keeps known primary tags without promoting broad buckets', () => {
  const project = {
    technology: {
      effective_tags: ['eff typescript', 'frontend', 'docker', 'unknown-profile-label'],
      detected_tags: ['raw javascript', 'backend', 'npm'],
      repositories: []
    }
  };

  const view = itemTechnologyView(project);

  assert.deepEqual(view.primary, ['typescript', 'javascript']);
  assert.deepEqual(view.tooling, ['docker', 'npm']);
  assert.deepEqual(view.capabilities, ['frontend', 'backend']);
});

test('repository view keeps runtimes primary and deduplicates raw/effective copies', () => {
  const repo = {
    effective: {
      languages: ['golang'],
      frameworks: [],
      runtimes: ['dotnet'],
      tooling: ['docker compose'],
      capabilities: ['backend']
    },
    detected: {
      languages: ['raw go'],
      frameworks: [],
      runtimes: ['eff dotnet'],
      tooling: ['docker-compose'],
      capabilities: ['frontend']
    },
    effective_tags: ['golang', 'dotnet', 'docker compose', 'backend'],
    detected_tags: ['raw go', 'eff dotnet', 'docker-compose', 'frontend']
  };

  const view = repositoryTechnologyView(repo);

  assert.deepEqual(view.primary, ['go', 'dotnet']);
  assert.deepEqual(view.tooling, ['docker-compose']);
  assert.deepEqual(view.capabilities, ['backend', 'frontend']);
});
