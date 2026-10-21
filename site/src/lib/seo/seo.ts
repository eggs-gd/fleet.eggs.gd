export const SITE = 'https://fleet.eggs.gd';
export const SITE_NAME = 'Fleet';
export const OG_IMAGE = `${SITE}/og.png`;

export const HOME_TITLE = 'fleet.eggs.gd — one manager for coding-agent work';
export const HOME_DESCRIPTION =
  'Fleet coordinates coding agents across local projects without replacing Cursor, Claude Code, Codex, or your review workflow.';

export const jsonLdHome = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: SITE_NAME,
  url: `${SITE}/`,
  description: HOME_DESCRIPTION,
  applicationCategory: 'DeveloperApplication',
  sameAs: ['https://github.com/eggs-gd/fleet.eggs.gd'],
  publisher: { '@type': 'Organization', name: 'earwiG Games', url: 'https://eggs.gd/' }
};
