import type { AnyRecord, Repository } from './types.ts';

export const primaryTechnologyKinds = ['languages', 'frameworks', 'runtimes'];

export const auditToolingTags = new Set([
  'docker',
  'docker-compose',
  'make',
  'npm',
  'pnpm',
  'vite',
  'yarn'
]);

const knownFlatPrimaryTags = new Set([
  'angular',
  'csharp',
  'dart',
  'dotnet',
  'flutter',
  'go',
  'github',
  'javascript',
  'node',
  'php',
  'python',
  'react',
  'ruby',
  'rust',
  'svelte',
  'tauri',
  'typescript',
  'unity',
  'vue'
]);

interface IconEntry {
  title: string;
  svg: string;
}

const svgIcon = (title: string, svg: string): IconEntry => ({ title, svg });

// Local SVG marks only — no icon package. Known tags never render shortened text labels.
export const technologyIconMap: Record<string, IconEntry> = {
  angular: svgIcon(
    'Angular',
    '<path fill="#DD0031" d="M12 2L2.5 5.5l1.5 13L12 22l8-3.5 1.5-13L12 2zm0 3.2l5.4 12.2h-2.1L14 14H10l-1.3 3.4H6.6L12 5.2zm0 3.3L10.7 12h2.6L12 8.5z"/>'
  ),
  csharp: svgIcon(
    'C#',
    '<circle fill="#512BD4" cx="12" cy="12" r="10"/><path fill="#fff" d="M8.2 8.4h1.5v2.1h2v1.4h-2v2.1H8.2v-2.1H6.2v-1.4h2V8.4zm7.2 1.8c.3-.2.7-.3 1.1-.3.6 0 1 .2 1.3.5l-.8.9c-.2-.2-.3-.3-.6-.3-.2 0-.4.1-.5.3s-.2.5-.2.8.1.6.2.8.3.3.5.3c.3 0 .4-.1.6-.3l.8.9c-.3.3-.7.5-1.3.5-.4 0-.8-.1-1.1-.3s-.6-.6-.7-1.1-.2-.9.1-1.4.4-.8.7-1z"/>'
  ),
  dart: svgIcon(
    'Dart',
    '<path fill="#0175C2" d="M4.2 9.5L12 2l7.8 7.5L12 22 4.2 9.5zm2.4.4L12 18.6l5.4-8.7L12 4.6 6.6 9.9z"/>'
  ),
  docker: svgIcon(
    'Docker',
    '<path fill="#2496ED" d="M4 14.2h1.5V12.8H4zm2 0h1.5V12.8H6zm2 0h1.5V12.8H8zm2 0h1.5V12.8h-1.5zm-6-1.9h1.5V10.9H4zm2 0h1.5V10.9H6zm2 0h1.5V10.9H8zm2 0h1.5V10.9h-1.5zm2 0H14V10.9h-1.5zM8 8.5h1.5V7.1H8zm2 0h1.5V7.1h-1.5zm2 0H14V7.1h-1.5zm3.2 3c.6 0 1.4-.1 1.9-.4.3-.7.2-1.5 0-1.9-.4-.1-.7-.1-1.2 0 0-.5-.2-1.2-.7-1.5-.2.2-.4.5-.4.9 0 0-.7-.1-1.3.4-.1.2-.1.5 0 1-.9.4-.9 1.5-.5 2 .4.4 1.2.5 2.2.5zM3 15.8c.3 1.5 1.5 3 4.1 3h7.6c2.4 0 3.9-1 4.7-2.8.6-.1 2-.4 1.9-2.1-.1-1.2-1.1-1.7-1.6-1.7H16"/>'
  ),
  'docker-compose': svgIcon(
    'Docker Compose',
    '<path fill="#2496ED" d="M4 14.2h1.5V12.8H4zm2 0h1.5V12.8H6zm2 0h1.5V12.8H8zm2 0h1.5V12.8h-1.5zm-4-1.9h1.5V10.9H6zm2 0h1.5V10.9H8zm2 0h1.5V10.9h-1.5zm2 0H14V10.9h-1.5zM8 8.5h1.5V7.1H8zm2 0h1.5V7.1h-1.5zM3 15.8c.3 1.5 1.5 3 4.1 3h7.6c2.4 0 3.9-1 4.7-2.8.6-.1 2-.4 1.9-2.1-.1-1.2-1.1-1.7-1.6-1.7h-1.7c.6 0 1.4-.1 1.9-.4.3-.7.2-1.5 0-1.9-.4-.1-.7-.1-1.2 0 0-.5-.2-1.2-.7-1.5-.2.2-.4.5-.4.9 0 0-.7-.1-1.3.4-.1.2-.1.5 0 1-.9.4-.9 1.5-.5 2 .4.4 1.2.5 2.2.5H3z"/>'
  ),
  dotnet: svgIcon(
    '.NET',
    '<rect width="24" height="24" rx="4" fill="#512BD4"/><path fill="#fff" d="M4 8h5.2l1.4 4.4L12.2 8H17v1.6h-3.2l-1.6 4.8h-1.7L8.4 9.6H5.6v8.2H4V8z"/>'
  ),
  flutter: svgIcon(
    'Flutter',
    '<path fill="#02569B" d="M14.3 2L3.5 12.8l3.4 3.4L21 2h-6.7z"/><path fill="#02569B" d="M14.3 12.3l-3.9 3.9 3.9 3.9H21l-5.2-5.2L21 9.7h-6.7z"/><path fill="#45D1FD" d="M10.4 16.2l1.9 1.9-1.9 1.9-1.9-1.9 1.9-1.9z"/>'
  ),
  github: svgIcon(
    'GitHub',
    '<path fill="#181717" d="M12 2C6.5 2 2 6.6 2 12.2c0 4.5 2.9 8.3 6.9 9.6.5.1.7-.2.7-.5v-1.9c-2.8.6-3.4-1.4-3.4-1.4-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 0 1.5 1 1.5 1 .9 1.6 2.4 1.1 3 .9.1-.7.4-1.1.6-1.4-2.2-.3-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.8 1a9.4 9.4 0 015 0c2-1.3 2.8-1 2.8-1 .5 1.4.2 2.4.1 2.7.6.7 1 1.6 1 2.7 0 3.9-2.3 4.7-4.6 5 .4.3.7 1 .7 2v3c0 .3.2.6.7.5 4-1.3 6.9-5.1 6.9-9.6C22 6.6 17.5 2 12 2z"/>'
  ),
  go: svgIcon(
    'Go',
    '<path fill="#00ADD8" d="M3.2 10.4c-.2 0-.3.1-.3.3v.4c0 .2.1.3.3.3h1.7c.1 0 .1 0 .1.1v.1c0 .7-.4 1.1-1.2 1.1-.7 0-1.2-.4-1.2-1.1V9.9c0-.8.6-1.3 1.5-1.3.7 0 1.2.3 1.4.9h-1c-.1-.2-.3-.3-.5-.3-.4 0-.7.3-.7.7v.5zm5.2-.5c0-.4-.3-.7-.7-.7s-.7.3-.7.7v2.5c0 .4.3.7.7.7s.7-.3.7-.7zm-1.8 0c0-.8.6-1.4 1.4-1.4.5 0 .9.2 1.1.6h0V7.8h1.1v4.9h-1v-.5h0c-.2.4-.6.6-1.1.6-.8 0-1.5-.6-1.5-1.4zm5.5 1.4h2.3c-.1.5-.5.8-1.1.8-.7 0-1.2-.5-1.2-1.3v-.1c0-.8.5-1.3 1.2-1.3.6 0 1 .3 1.1.8h-2.3zm3.4-.6c0-1.2-.9-2.1-2.2-2.1-1.3 0-2.3.9-2.3 2.1v.3c0 1.2.9 2.1 2.3 2.1 1.2 0 2.1-.7 2.2-1.8h-1.1c-.1.4-.5.7-1.1.7-.7 0-1.2-.5-1.2-1.2v-.1h3.4v-.1zm3.3.6h2.3c-.1.5-.5.8-1.1.8-.7 0-1.2-.5-1.2-1.3v-.1c0-.8.5-1.3 1.2-1.3.6 0 1 .3 1.1.8h-2.3zm3.4-.6c0-1.2-.9-2.1-2.2-2.1-1.3 0-2.3.9-2.3 2.1v.3c0 1.2.9 2.1 2.3 2.1 1.2 0 2.1-.7 2.2-1.8h-1.1c-.1.4-.5.7-1.1.7-.7 0-1.2-.5-1.2-1.2v-.1h3.4v-.1z"/><ellipse fill="#00ADD8" cx="5.7" cy="8.7" rx=".35" ry=".35"/><ellipse fill="#00ADD8" cx="16.3" cy="8.7" rx=".35" ry=".35"/>'
  ),
  javascript: svgIcon(
    'JavaScript',
    '<rect width="24" height="24" fill="#F7DF1E"/><path fill="#000" d="M11.1 16.9c0 1.9-1.1 2.8-3 2.8-1.4 0-2.4-.6-3-1.7l1.6-.9c.3.6.7.9 1.3.9.6 0 1-.3 1-1.3v-5.9h2.1v5.1zm4.4 2.8c-1.7 0-2.8-.8-3.3-1.9l1.6-.9c.3.6.8 1 1.6 1 .7 0 1.1-.3 1.1-.8 0-.5-.4-.7-1.3-1l-.5-.2c-1.4-.6-2.3-1.4-2.3-3 0-1.5 1.1-2.6 2.9-2.6 1.3 0 2.2.4 2.8 1.6l-1.5.9c-.3-.5-.7-.7-1.3-.7-.5 0-.9.3-.9.7 0 .5.3.7 1.2 1l.5.2c1.7.7 2.5 1.5 2.5 3.1 0 1.8-1.4 2.8-3.1 2.8z"/>'
  ),
  make: svgIcon(
    'Make',
    '<path fill="#6B7280" d="M4 5h16v2.2H4V5zm0 5.9h16V13H4v-2.1zm0 5.9h10V19H4v-2.2z"/>'
  ),
  node: svgIcon(
    'Node.js',
    '<path fill="#339933" d="M12 2.1L3.5 7v10L12 21.9 20.5 17V7L12 2.1zm0 1.9l6.5 3.7v7.4L12 18.9 5.5 15.1V7.7L12 4z"/><path fill="#339933" d="M10.4 9v6h1.5v-2.2h.8c1.5 0 2.4-.8 2.4-2 0-1.2-.9-1.8-2.4-1.8h-2.3zm1.5 1.2h.7c.6 0 1 .3 1 .8s-.4.7-1 .7h-.7V10.2z"/>'
  ),
  npm: svgIcon(
    'npm',
    '<rect width="24" height="24" rx="2" fill="#CB3837"/><path fill="#fff" d="M5.2 5.2v13.6h5.4V8.6h3.2v10.2h3V5.2H5.2z"/>'
  ),
  php: svgIcon(
    'PHP',
    '<ellipse fill="#777BB4" cx="12" cy="12" rx="10" ry="6"/><path fill="#fff" d="M7.2 9.4h1.4c1.1 0 1.7.5 1.7 1.5 0 1.1-.7 1.6-1.8 1.6h-.6l.3 1.7H7.1L6 9.4zm1.5 1.2l.2 1.2h.4c.4 0 .7-.1.7-.6s-.2-.5-.6-.5l-.7-.1zm4.1-1.2h1.5l.2 1.4h.1c.2-.4.7-.8 1.4-.7l-.2 1.2c-.4 0-.9.1-1.1.6l.5 2.4h-1.5l-.9-4.9zm4.2 0h1.5l-1 4.9h-1.5l1-4.9z"/>'
  ),
  pnpm: svgIcon(
    'pnpm',
    '<path fill="#F69220" d="M2 2h6v6H2V2zm7 0h6v6H9V2zm7 0h6v6h-6V2zM2 9h6v6H2V9zm7 0h6v6H9V9zm0 7h6v6H9v-6zm7-7h6v6h-6V9z"/><path fill="#4C4C4C" d="M9 16h6v6H9z"/>'
  ),
  python: svgIcon(
    'Python',
    '<path fill="#3776AB" d="M12.1 2c-2.2 0-2.1.9-2.1 2.1v1.5h4.2v.4H7.5C5.8 6 4.5 7.2 4.5 9.3v2.1c0 2.1 1.1 2.8 2.9 2.8h1.5V12c0-2.2 1.9-3.2 4.1-3.2h2.2c1.9 0 2.8-1 2.8-2.8V4.1C18 2.5 17 2 12.1 2zm-2 1.4c.4 0 .7.3.7.7s-.3.7-.7.7-.7-.3-.7-.7.3-.7.7-.7z"/><path fill="#FFD43B" d="M16.6 9.8v2.2c0 2.3-2 3.2-4.1 3.2h-2.2c-1.9 0-2.8 1.1-2.8 2.8v2.1c0 1.7 1.1 2.1 4.1 2.1 2.2 0 2.1-.9 2.1-2.1v-1.5H9.5v-.4h6.7c1.7 0 2.9-1.1 2.9-3.2V12.6c0-2.1-1.2-2.8-2.5-2.8zm-1.9 9.3c.4 0 .7.3.7.7s-.3.7-.7.7-.7-.3-.7-.7.3-.7.7-.7z"/>'
  ),
  react: svgIcon(
    'React',
    '<circle fill="#61DAFB" cx="12" cy="12" r="2"/><ellipse fill="none" stroke="#61DAFB" stroke-width="1.4" cx="12" cy="12" rx="10" ry="4"/><ellipse fill="none" stroke="#61DAFB" stroke-width="1.4" cx="12" cy="12" rx="10" ry="4" transform="rotate(60 12 12)"/><ellipse fill="none" stroke="#61DAFB" stroke-width="1.4" cx="12" cy="12" rx="10" ry="4" transform="rotate(120 12 12)"/>'
  ),
  ruby: svgIcon(
    'Ruby',
    '<path fill="#CC342D" d="M18.8 7.3L12.2 2 5 7.5l2.3 8.5L12.2 22l5.3-5.9 1.3-8.8zM7.4 8.4l4.1-3.1 4 3.2-1.7 1.5H9.1L7.4 8.4zm.7 1.9l1.5 4.7-3.2-3.2 1.7-1.5zm1.9 5.5l2.3 3.3-4.4-2.5 2.1-.8zm3.1 3.3l2.3-3.3 2.1.8-4.4 2.5zm3-4.1l1.5-4.7 1.7 1.5-3.2 3.2zm-1.1-5.4l-2.5-2-2.5 2h5z"/>'
  ),
  rust: svgIcon(
    'Rust',
    '<path fill="#000" d="M12 2.2l1.1 1.1 1.5-.3.7 1.3 1.5.4.2 1.5 1.3.8-.4 1.5.8 1.3-1 1.1.4 1.5-1.3.7-.3 1.5-1.5.2-1.1 1.1-1.5-.4-.7 1.3-1.5-.2-.8 1.3-1.3-.8-1.1 1-1.5-.4-.7-1.3-1.5-.2-.2-1.5-1.3-.8.4-1.5-.8-1.3 1-1.1-.4-1.5 1.3-.7.3-1.5 1.5-.2L9.4 3l1.5.4L12 2.2zm0 4.3a5.5 5.5 0 100 11 5.5 5.5 0 000-11zm0 1.6a3.9 3.9 0 110 7.8 3.9 3.9 0 010-7.8z"/>'
  ),
  svelte: svgIcon(
    'Svelte',
    '<path fill="#FF3E00" d="M20.6 4.8C18.8 2 15.2.9 11.7 2.1L6.7 4c-1.3.5-2.4 1.4-3.1 2.6-.9 1.6-.9 3.5.1 5l.2.3a5.7 5.7 0 00-1.1 3.4c.1 1.9.9 3.6 2.5 4.8C7.1 22 10.7 23.1 14.2 21.9l5-1.9c1.3-.5 2.4-1.4 3.1-2.6.9-1.6.9-3.5-.1-5l-.2-.3c.7-1 .9-2.2 1.1-3.4-.1-1.9-.9-3.6-2.5-4.9zM10.4 20c-1.9.7-4 .1-5.1-1.5-.6-.9-.8-1.9-.6-2.9l.1-.4 1.3.6c.1 0 .1.1.2.1-.1.3-.1.6 0 .9.4 1.1 1.6 1.5 2.7 1.1l5-1.9c.5-.2.9-.6 1.1-1.1.2-.5.2-1.1 0-1.6-.4-.6-1-1-1.7-1l-.5.1-3.7 1.4c-1.9.7-4 .1-5.1-1.5C2.9 10.6 3 8.5 4.6 7.2c.8-.6 1.7-.9 2.7-.9.7 0 1.4.2 2 .4l3.7-1.4c1.9-.7 4-.1 5.1 1.5.6.9.8 1.9.6 2.9l-.1.4-1.3-.6-.2-.1c.1-.3.1-.6 0-.9-.4-1.1-1.6-1.5-2.7-1.1l-5 1.9c-.5.2-.9.6-1.1 1.1-.2.5-.2 1.1 0 1.6.4.6 1 1 1.7 1l.5-.1 3.7-1.4c1.9-.7 4-.1 5.1 1.5 1.2 1.7 1.1 3.8-.5 5.1-1.6 1.4-3.8 1.6-5.7.9L10.4 20z"/>'
  ),
  tauri: svgIcon(
    'Tauri',
    '<path fill="#24C8DB" d="M12 3.5c-4.7 0-8.5 3.1-8.5 7 0 2.5 1.5 4.7 3.8 6l.9-1.6c-1.6-.9-2.6-2.5-2.6-4.4 0-2.8 2.9-5.1 6.4-5.1s6.4 2.3 6.4 5.1c0 1.9-1 3.5-2.6 4.4l.9 1.6c2.3-1.3 3.8-3.5 3.8-6 0-3.9-3.8-7-8.5-7zm0 4.2c-1.7 0-3 1.3-3 3s1.3 3 3 3 3-1.3 3-3-1.3-3-3-3zm-6.7 8.3l1.5 1.1L12 21l5.2-4 1.5-1.1-1.8-1.3L12 18.2l-4.9-3.5-1.8 1.3z"/>'
  ),
  typescript: svgIcon(
    'TypeScript',
    '<rect width="24" height="24" fill="#3178C6"/><path fill="#fff" d="M13.3 11.1H6.8V9.4h12.1v1.7h-3.3V20h-2.3V11.1zm5.4 5.2c.2.5.5.8 1.1.8.5 0 .8-.2.8-.6 0-.4-.2-.5-.9-.8l-.5-.2c-1.2-.5-1.8-1.1-1.8-2.2 0-1.2.9-2.1 2.4-2.1 1.1 0 1.9.3 2.5 1.2l-1.4.9c-.2-.4-.5-.6-1-.6-.4 0-.6.2-.6.5 0 .3.2.5.9.8l.5.2c1.4.6 2 1.2 2 2.4 0 1.4-1.1 2.2-2.6 2.2-1.4 0-2.3-.6-2.8-1.6l1.4-.9z"/>'
  ),
  unity: svgIcon(
    'Unity',
    '<path fill="#000" d="M12 2L3.5 7v10L12 22l8.5-5V7L12 2zm0 2.3l6.2 3.6v1.9l-3.1-1.8-3.1 1.8V8.1L12 6.3 8.9 8.1v1.7L5.8 9.9V8.2L12 4.3zM5.8 11.8l3.1 1.8v3.5L5.8 15.3v-3.5zm4.9 2.8L12 15.7l1.3-1.1v3.7L12 19.4l-1.3-1.1v-3.7zm3.6-1l3.1-1.8v3.5l-3.1 1.8v-3.5z"/>'
  ),
  vite: svgIcon(
    'Vite',
    '<path fill="#646CFF" d="M12.8 2L3 19.5h5.2L12.8 8l3.4 7.2h2.2L12.8 2z"/><path fill="#FFD62E" d="M12.8 8l3.4 7.2-3.4 6.3-3.4-6.3L12.8 8z"/>'
  ),
  vue: svgIcon(
    'Vue',
    '<path fill="#4FC08D" d="M1.5 3h4.4L12 13.2 18.1 3H22.5L12 21 1.5 3z"/><path fill="#35495E" d="M8.7 3L12 8.5 15.3 3h-2.4L12 5.2 11.1 3H8.7z"/>'
  ),
  yarn: svgIcon(
    'Yarn',
    '<path fill="#2C8EBB" d="M12 2C7 2 3.4 5.4 3 10.1c1.1-.3 2-.1 2.5.3.5-1.1 1.4-1.5 2.3-1.7.1-.9.5-1.8 1.4-2.5.1.5.4 1.2 1 1.5C11 6.5 12 5.8 13.5 6c1.2.1 1.9.9 2.2 1.9 1 .3 1.7 1 1.9 2.1.8.1 1.6.6 1.9 1.5.7.2 1.2.8 1.3 1.6.7 3.5-2.3 6.9-6.8 6.9-4.1 0-7.2-2.7-7.9-6.2-.8.1-1.8-.4-2.1-1.4C3.3 17.8 7.2 22 12 22c5.5 0 10-4.5 10-10S17.5 2 12 2z"/>'
  )
};

export const normalizeTechnologyTag = (tag: unknown): string => {
  let value = String(tag || '')
    .trim()
    .toLowerCase();
  while (/^(raw|detected|eff|effective)[:\s_-]+/.test(value)) {
    value = value.replace(/^(raw|detected|eff|effective)[:\s_-]+/, '');
  }
  if (value === 'golang') return 'go';
  if (value === 'nodejs' || value === 'node.js') return 'node';
  if (value === 'github.com') return 'github';
  if (value === '.net') return 'dotnet';
  if (value === 'c#') return 'csharp';
  if (value === 'docker compose') return 'docker-compose';
  return value;
};

const fallbackTechnologyLabel = (tag: string): string => {
  const words = tag.split(/[^a-z0-9+#.]+/).filter(Boolean);
  if (words.length > 1) {
    return words
      .slice(0, 2)
      .map((word) => word.slice(0, 2))
      .join('')
      .slice(0, 4);
  }
  return tag.slice(0, 4) || '?';
};

const technologyTitle = (tag: string): string =>
  tag
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ') || 'Unknown technology';

export interface TechnologyTagDisplay {
  name: string;
  glyph: string;
  label: string;
  title: string;
  known: boolean;
  svg: string;
}

export const technologyTagDisplay = (tag: unknown): TechnologyTagDisplay => {
  const name = normalizeTechnologyTag(tag);
  const mapped = technologyIconMap[name];
  if (mapped?.svg) {
    return {
      name,
      glyph: '',
      label: '',
      title: mapped.title,
      known: true,
      svg: mapped.svg
    };
  }
  return {
    name,
    glyph: '',
    label: fallbackTechnologyLabel(name),
    title: technologyTitle(name),
    known: false,
    svg: ''
  };
};

export const displayTechnologyTags = (tags: string[], limit = 8) => ({
  visible: tags.slice(0, limit).map(technologyTagDisplay),
  hidden: Math.max(tags.length - limit, 0)
});

export interface TechnologyGroup {
  variant?: string;
  tags?: string[];
}

/** Flatten variant groups into one overflow-aware icon row. */
export const displayTechnologyGroups = (groups: TechnologyGroup[] = [], limit = 8) => {
  const flattened: { tag: TechnologyTagDisplay; variant: string }[] = [];
  const seen = new Set<string>();
  for (const group of groups) {
    const variant = group?.variant || 'primary';
    for (const raw of group?.tags || []) {
      const name = normalizeTechnologyTag(raw);
      if (!name || seen.has(name)) continue;
      seen.add(name);
      flattened.push({ tag: technologyTagDisplay(name), variant });
    }
  }
  return {
    visible: flattened.slice(0, Math.max(limit, 0)),
    hidden: Math.max(flattened.length - Math.max(limit, 0), 0)
  };
};

export const uniqueTechnologyTags = (tags: unknown[]): string[] => {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const tag of tags) {
    const value = normalizeTechnologyTag(tag);
    if (!value || seen.has(value)) continue;
    seen.add(value);
    result.push(value);
  }
  return result;
};

const withoutTags = (tags: string[], blocked: Set<string>): string[] =>
  tags.filter((tag) => !blocked.has(tag));
const profileTags = (profile: AnyRecord | null | undefined, kinds: string[]): string[] =>
  uniqueTechnologyTags(kinds.flatMap((kind) => profile?.[kind] || []));
const repositoryProfileTags = (
  repo: AnyRecord | null | undefined,
  source: string,
  kinds: string[]
): string[] => profileTags(repo?.[source], kinds);
const technology = (item: AnyRecord | null | undefined): AnyRecord => item?.technology || {};
export const techRepositories = (item: AnyRecord | null | undefined): Repository[] =>
  technology(item).repositories || [];
const itemProfileTags = (
  item: AnyRecord | null | undefined,
  source: string,
  kinds: string[]
): string[] =>
  uniqueTechnologyTags(
    techRepositories(item).flatMap((repo) => repositoryProfileTags(repo, source, kinds))
  );
const flatTechnologyTags = (item: AnyRecord | null | undefined, source: string): string[] =>
  uniqueTechnologyTags(technology(item)?.[`${source}_tags`] || []);
const mergeTechnologyTags = (...lists: unknown[][]): string[] => uniqueTechnologyTags(lists.flat());
const fallbackPrimaryTags = (item: AnyRecord | null | undefined, source: string): string[] =>
  flatTechnologyTags(item, source).filter(
    (tag) => knownFlatPrimaryTags.has(tag) && !auditToolingTags.has(tag)
  );
const categoryFallbackTags = (
  item: AnyRecord | null | undefined,
  source: string,
  category: Set<string>
): string[] => flatTechnologyTags(item, source).filter((tag) => category.has(tag));
const primaryProfileTags = (item: AnyRecord | null | undefined, source: string): string[] =>
  withoutTags(itemProfileTags(item, source, primaryTechnologyKinds), auditToolingTags);
const toolingProfileTags = (item: AnyRecord | null | undefined, source: string): string[] =>
  mergeTechnologyTags(
    itemProfileTags(item, source, ['tooling']),
    itemProfileTags(item, source, primaryTechnologyKinds).filter((tag) => auditToolingTags.has(tag))
  );
const repositoryPrimaryProfileTags = (
  repo: AnyRecord | null | undefined,
  source: string
): string[] =>
  withoutTags(repositoryProfileTags(repo, source, primaryTechnologyKinds), auditToolingTags);
const repositoryToolingProfileTags = (
  repo: AnyRecord | null | undefined,
  source: string
): string[] =>
  mergeTechnologyTags(
    repositoryProfileTags(repo, source, ['tooling']),
    repositoryProfileTags(repo, source, primaryTechnologyKinds).filter((tag) =>
      auditToolingTags.has(tag)
    )
  );

export interface TechnologyView {
  primary: string[];
  tooling: string[];
  hasAny: boolean;
}

export const itemTechnologyView = (item: AnyRecord | null | undefined): TechnologyView => {
  const effectivePrimary = primaryProfileTags(item, 'effective');
  const detectedPrimary = primaryProfileTags(item, 'detected');
  const primary = mergeTechnologyTags(
    effectivePrimary.length ? effectivePrimary : fallbackPrimaryTags(item, 'effective'),
    detectedPrimary.length ? detectedPrimary : fallbackPrimaryTags(item, 'detected')
  );
  const tooling = mergeTechnologyTags(
    toolingProfileTags(item, 'effective'),
    toolingProfileTags(item, 'detected'),
    categoryFallbackTags(item, 'effective', auditToolingTags),
    categoryFallbackTags(item, 'detected', auditToolingTags)
  );
  return {
    primary,
    tooling,
    hasAny: primary.length > 0 || tooling.length > 0 || techRepositories(item).length > 0
  };
};

export const repositoryTechnologyView = (repo: Repository | AnyRecord | null | undefined) => ({
  primary: mergeTechnologyTags(
    repositoryPrimaryProfileTags(repo, 'effective'),
    repositoryPrimaryProfileTags(repo, 'detected'),
    uniqueTechnologyTags(repo?.effective_tags || []).filter(
      (tag) => knownFlatPrimaryTags.has(tag) && !auditToolingTags.has(tag)
    ),
    uniqueTechnologyTags(repo?.detected_tags || []).filter(
      (tag) => knownFlatPrimaryTags.has(tag) && !auditToolingTags.has(tag)
    )
  ),
  tooling: mergeTechnologyTags(
    repositoryToolingProfileTags(repo, 'effective'),
    repositoryToolingProfileTags(repo, 'detected'),
    uniqueTechnologyTags([...(repo?.effective_tags || []), ...(repo?.detected_tags || [])]).filter(
      (tag) => auditToolingTags.has(tag)
    )
  )
});

export const hasTechnology = (item: AnyRecord | null | undefined): boolean =>
  itemTechnologyView(item).hasAny;

export const repoHasTechnology = (repo: AnyRecord | null | undefined): boolean => {
  const view = repositoryTechnologyView(repo);
  return view.primary.length > 0 || view.tooling.length > 0 || (repo?.evidence?.length || 0) > 0;
};

// Registry remotes are stored however git reports them (often SSH, e.g.
// git@github.com:org/repo.git) — browsers can't open that as a link, so it
// silently falls through to the current page. Normalize to https before use.
export const repoWebUrl = (remote: unknown): string => {
  const value = String(remote || '').trim();
  if (!value) return '';
  const withoutSuffix = value.replace(/\.git$/, '');
  // Check the full ssh://[user@]host[:port]/path form before the scp-style
  // shorthand below — otherwise the scp regex greedily reads "ssh" itself as
  // the host (matching up to the first ':') and never reaches this branch.
  const sshUrlMatch = withoutSuffix.match(/^ssh:\/\/(?:[\w.-]+@)?([^/:]+)(?::\d+)?\/(.+)$/);
  if (sshUrlMatch) {
    return `https://${sshUrlMatch[1]}/${sshUrlMatch[2]}`;
  }
  if (/^https?:\/\//i.test(withoutSuffix)) return withoutSuffix;
  const scpMatch = withoutSuffix.match(/^(?:[\w.-]+@)?([\w.-]+):(.+)$/);
  if (scpMatch && scpMatch[1].toLowerCase() !== 'ssh') {
    return `https://${scpMatch[1]}/${scpMatch[2]}`;
  }
  return '';
};
