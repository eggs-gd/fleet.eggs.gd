<script lang="ts">
  import { ScaleLock, Section, SectionTag, SectionTitle, Split } from '$lib/layout';
</script>

<Section class="ownership-section" id="ownership" aria-labelledby="ownership-title">
  {#snippet tag()}
    <SectionTag size="m">Ownership</SectionTag>
  {/snippet}
  <ScaleLock>
    <Split flat>
      {#snippet start()}
        <div class="conflict-demo" aria-label="Repository conflict demo">
          <div class="repo-surface">
            <header>
              <strong>perceptrail</strong>
              <span>Active</span>
            </header>
            <div class="owner-row running">
              <span>Codex</span>
              <strong>Running</strong>
            </div>
            <div class="owner-row blocked">
              <span>Claude Code</span>
              <strong>Blocked</strong>
            </div>
            <p>Repository currently owned by Codex.</p>
          </div>
        </div>
      {/snippet}
      {#snippet end()}
        <div class="ownership-statement">
          <SectionTitle id="ownership-title" size="xl">
            <span>One repository.</span>
            <span>One active task.</span>
            <span>One owner.</span>
          </SectionTitle>
        </div>
      {/snippet}
    </Split>
  </ScaleLock>
</Section>

<style>
:global {
  .ownership-section {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(520px, 0.84fr);
    gap: 52px;
    align-items: center;
  }

  .ownership-statement {
    container-type: inline-size;
  }

  .ownership-statement h2 {
    display: grid;
    gap: 4px;
    max-width: none;
    margin-top: 20px;
    line-height: 0.88;
    text-transform: uppercase;
  }

  .ownership-statement span {
    display: block;
    white-space: nowrap;
  }

  .conflict-demo {
    display: grid;
  }

  .repo-surface {
    border: 1px solid var(--line);
    border-radius: 6px;
    background: #12171d;
    overflow: hidden;
  }

  .repo-surface header,
  .owner-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 22px 24px;
  }

  .repo-surface header {
    border-bottom: 1px solid var(--line);
    color: #f4f7fb;
    font-size: 22px;
  }

  .repo-surface header span,
  .owner-row strong {
    font-weight: 680;
  }

  .repo-surface header span {
    color: var(--good);
  }

  .owner-row {
    border-bottom: 1px solid rgba(218, 226, 237, 0.1);
    color: #d5dde8;
    font-size: 20px;
  }

  .owner-row span {
    white-space: nowrap;
  }

  .owner-row.running strong {
    color: var(--good);
  }

  .owner-row.blocked strong {
    color: var(--blocked);
  }

  .repo-surface p {
    margin: 0;
    padding: 14px 24px 18px;
    color: var(--muted);
    font-size: 14px;
    line-height: 1.4;
  }

  @media (min-width: 1181px) {
    .ownership-section {
      display: block;
      container-type: inline-size;
      align-items: stretch;
    }

    .ownership-canvas {
      display: grid;
      grid-template-columns: 460px minmax(0, 1fr);
      align-items: stretch;
      width: 1372px;
      column-gap: 52px;
      zoom: calc(100cqi / 1372px);
    }

    .ownership-section .ownership-statement {
      display: contents;
    }

    .ownership-section .ownership-statement h2 {
      grid-column: 2;
      grid-row: 1;
      margin-top: 0;
      line-height: 0.88;
      gap: 18px;
    }

    .ownership-section .ownership-statement span {
      font-size: inherit;
      line-height: inherit;
    }

    .ownership-section .conflict-demo {
      display: flex;
      grid-column: 1;
      grid-row: 1;
      justify-self: start;
      align-self: stretch;
      width: 460px;
      margin-top: 0;
    }

    .ownership-section .repo-surface {
      display: flex;
      flex: 1;
      flex-direction: column;
    }

    .ownership-section .repo-surface header,
    .ownership-section .owner-row {
      padding: 28px 32px;
    }

    .ownership-section .repo-surface header {
      font-size: 28px;
    }

    .ownership-section .owner-row {
      flex: 1;
      font-size: 26px;
    }

    .ownership-section .repo-surface p {
      padding: 16px 32px 20px;
      font-size: 16px;
    }
  }

  @media (max-width: 1180px) {
    .ownership-section {
      grid-template-columns: 1fr;
    }

    .ownership-statement h2 {
      font-size: 11.2cqi;
    }
  }
}
</style>
