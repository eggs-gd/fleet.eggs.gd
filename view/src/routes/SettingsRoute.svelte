<script lang="ts">
  import SettingsView from '../components/settings/SettingsView.svelte';

  export let section = 'general';
  export let themePref = 'system';
  export let onThemeChange: (pref: string) => void = () => {};
  export let refreshMs = 5000;
  export let onRefreshChange: (ms: number) => void = () => {};
</script>

<SettingsView {section} {themePref} {onThemeChange} {refreshMs} {onRefreshChange} />
