<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionSubtitle,
    SectionTag,
    SectionText,
    SectionTitle,
    Stack
  } from '$lib/layout';
</script>

<Section class="dashboard-section" aria-labelledby="dashboard-title">
  {#snippet tag()}
    <SectionTag size="m">Product proof</SectionTag>
  {/snippet}
  {#snippet band()}
    <div class="section-heading">
      <SectionTitle id="dashboard-title" size="m">From one task to many projects.</SectionTitle>
      <SectionSubtitle size="m">
        Fleet gives you one place to see projects, repositories, dependencies, active sessions, work
        waiting for review, and agent availability.
      </SectionSubtitle>
    </div>
  {/snippet}
  <Stack gap="0">
    <figure class="product-shot dashboard-shot">
      <img
        src="/product/fleet-work-dashboard.png"
        alt="Fleet dashboard overview with projects, sessions, tasks, needs attention, and completed work"
      />
    </figure>
  </Stack>
  {#snippet footer()}
    <SectionFoot>
      <SectionText class="statement" size="xl">
        The control plane between your intent and the agents doing the work.
      </SectionText>
      <div class="positioning-line">
        <span>Not another IDE.</span>
        <span>Not another model.</span>
        <span>Not another cloud project-management suite.</span>
      </div>
    </SectionFoot>
  {/snippet}
</Section>

<style>
:global {
  .dashboard-section {
    display: grid;
    gap: 30px;
  }

  .section-heading {
    display: grid;
    gap: 18px;
    max-width: 850px;
    margin-bottom: 42px;
  }

  .section-heading .section-subtitle {
    color: var(--muted);
  }

  .dashboard-shot {
    width: 100%;
    aspect-ratio: 16 / 10;
    margin: 10px auto 0;
  }

  .dashboard-shot img {
    object-position: center top;
  }

  .positioning-line {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .positioning-line span {
    border: 1px solid var(--line);
    border-radius: 4px;
    padding: 8px 10px;
    color: #c8d0db;
    background: rgba(255, 255, 255, 0.02);
    font-size: 13px;
    font-weight: 640;
  }

  .dashboard-section .section-foot {
    display: grid;
    gap: 14px;
    margin-top: 16px;
  }

  .dashboard-section .section-foot .statement {
    margin: 0;
    max-width: none;
    text-align: left;
  }

  @media (min-width: 1181px) {
    .dashboard-section {
      grid-template-columns: minmax(0, 1fr);
      row-gap: 0;
      justify-items: start;
    }

    .dashboard-section .section-heading {
      grid-row: 2;
      justify-self: stretch;
      width: 100%;
      max-width: none;
      margin-bottom: 0;
      gap: 14px;
      text-align: left;
    }

    .dashboard-section .section-heading h2 {
      max-width: none;
    }

    .dashboard-section .dashboard-shot {
      grid-row: 3;
      justify-self: stretch;
      width: 100%;
      max-width: none;
      aspect-ratio: 16 / 10;
      margin: 28px 0 0;
    }

    .dashboard-section .section-foot {
      grid-row: 4;
      justify-self: stretch;
      width: 100%;
    }

    .dashboard-section .dashboard-shot img {
      object-fit: contain;
      object-position: center center;
      transform: none;
    }
  }
}
</style>
