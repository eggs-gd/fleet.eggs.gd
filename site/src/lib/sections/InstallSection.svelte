<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionSubtitle,
    SectionTag,
    SectionTitle
  } from '$lib/layout';
</script>

<Section class="pricing-section" id="install" aria-labelledby="install-title">
  {#snippet tag()}
    <SectionTag size="m">Run Fleet locally</SectionTag>
  {/snippet}
  {#snippet band()}
    <SectionTitle id="install-title" size="s">
      Run coding agents without losing track of the work.
    </SectionTitle>
  {/snippet}
  <div class="install-band">
    <div class="pricing-panel">
      <span>First 5 completed tasks are free.</span>
      <strong>$9 / month</strong>
      <SectionFoot>
        <div class="download-options">
          <a class="primary-cta" href="https://github.com/eggs-gd/fleet.eggs.gd/releases">Download Mac</a>
          <a class="secondary-cta" href="https://github.com/eggs-gd/fleet.eggs.gd/releases">Linux</a>
          <a class="secondary-cta" href="https://github.com/eggs-gd/fleet.eggs.gd/releases">Windows</a>
        </div>
      </SectionFoot>
    </div>
    <SectionSubtitle class="install-lead" size="l">
      Fleet runs against your local projects and data. Connect the coding agents you already use and
      start with real work.
    </SectionSubtitle>
  </div>
</Section>

<style>
:global {
  .pricing-section {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 28px;
    align-items: start;
  }

  .pricing-section h2 {
    max-width: none;
    margin-top: 0;
  }

  .install-band {
    display: grid;
    gap: 28px;
    align-items: center;
  }

  .install-lead {
    max-width: 34em;
    margin: 0;
    color: var(--muted);
  }

  .pricing-panel {
    display: grid;
    align-content: start;
    gap: 0;
    border: 1px solid var(--line-strong);
    border-radius: 6px;
    padding: 28px 32px 32px;
  }

  .download-options {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 22px;
  }

  .pricing-panel > span {
    color: #aeb7c4;
    font-size: 15px;
  }

  .pricing-panel > strong {
    color: #f8fafc;
    font-size: clamp(40px, 4.5vw, 64px);
    line-height: 0.95;
  }

  @media (min-width: 1181px) {
    .pricing-section {
      row-gap: 0;
      align-items: start;
    }

    .pricing-section > h2 {
      grid-row: 2;
      margin: 0 0 48px;
      line-height: 1.12;
    }

    .pricing-section .install-band {
      grid-row: 3;
      grid-template-columns: minmax(0, 0.92fr) minmax(0, 1.08fr);
      column-gap: 88px;
    }

    .pricing-section .install-lead {
      max-width: 28em;
      margin-top: 0;
      color: #d5dde8;
    }

    .pricing-panel > span {
      font-size: 16px;
    }

    .pricing-panel > strong {
      margin-top: 14px;
      font-size: 72px;
      letter-spacing: -0.03em;
    }
  }

  @media (max-width: 1180px) {
    .pricing-section {
      grid-template-columns: 1fr;
    }
  }
}
</style>
