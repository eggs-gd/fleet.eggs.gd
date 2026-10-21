<script lang="ts">
  import { OG_IMAGE, SITE, SITE_NAME } from './seo';

  interface Props {
    title: string;
    description: string;
    /** Absolute path from the site root, no trailing slash except for "/". */
    path: string;
    jsonLd?: object;
  }

  let { title, description, path, jsonLd }: Props = $props();

  const url = $derived(`${SITE}${path}`);
  // "<" is escaped so the JSON can never close the script element.
  const ld = $derived(jsonLd ? JSON.stringify(jsonLd).replace(/</g, '\\u003c') : '');
</script>

<svelte:head>
  <title>{title}</title>
  <meta name="description" content={description} />
  <link rel="canonical" href={url} />
  <meta property="og:type" content="website" />
  <meta property="og:site_name" content={SITE_NAME} />
  <meta property="og:title" content={title} />
  <meta property="og:description" content={description} />
  <meta property="og:url" content={url} />
  <meta property="og:image" content={OG_IMAGE} />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
  <meta
    property="og:image:alt"
    content="Fleet dashboard: tasks, agents and execution state across projects"
  />
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content={title} />
  <meta name="twitter:description" content={description} />
  <meta name="twitter:image" content={OG_IMAGE} />
  {#if ld}
    {@html `<script type="application/ld+json">${ld}</script>`}
  {/if}
</svelte:head>
