<script lang="ts">
  import DownloadButton from '$lib/download/DownloadButton.svelte';
  import { Section, SectionText, SectionTitle, Stack } from '$lib/layout';
</script>

<Section class="v2-install" id="install" aria-labelledby="install-title">
  {#snippet band()}
    <SectionTitle id="install-title" size="m">
      Run Fleet against your local projects and agents.
    </SectionTitle>
  {/snippet}

  <Stack gap="m">
    <SectionText size="m">
      Fleet works with your local projects and the coding agents you already use.
    </SectionText>

    <div class="download-row">
      <DownloadButton />
      <span class="early-badge">Early access</span>
    </div>
  </Stack>
</Section>

<style>
  :global {
    .v2-install {
      text-align: center;
    }

    .v2-install .section-title {
      max-width: 920px;
      margin: 0 auto;
    }

    .v2-install .download-row {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: center;
      gap: 14px;
    }
  }
</style>
