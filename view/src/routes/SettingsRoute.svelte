<script lang="ts">
  import SettingsView from '../components/settings/SettingsView.svelte';
  import { refreshMs, setRefreshMs, setThemePref, themePref } from '../stores/layout';
  import { settingsSectionId } from '../stores/location';
</script>

<SettingsView
  section={$settingsSectionId}
  themePref={$themePref}
  onThemeChange={setThemePref}
  refreshMs={$refreshMs}
  onRefreshChange={setRefreshMs}
/>
