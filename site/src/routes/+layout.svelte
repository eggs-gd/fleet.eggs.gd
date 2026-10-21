<script>
  import '@fontsource-variable/space-grotesk';
  import '@fontsource-variable/jetbrains-mono';
  import '../styles.css';
  import ConsentBanner from '$lib/analytics/ConsentBanner.svelte';
</script>

<slot />
<ConsentBanner privacyHref="https://eggs.gd/privacy" />
