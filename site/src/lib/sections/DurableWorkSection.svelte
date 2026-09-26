<script lang="ts">
  import { Section, SectionSubtitle, SectionTag, SectionTitle, Split } from '$lib/layout';
</script>

<Section class="local-section" aria-labelledby="local-title">
  {#snippet tag()}
    <SectionTag size="m">Durable work</SectionTag>
  {/snippet}
  <Split flat>
    {#snippet start()}
      <div class="local-copy">
        <SectionTitle id="local-title" size="l">Nothing disappears when the chat ends.</SectionTitle>
        <SectionSubtitle size="s">
          Tasks, decisions, context, results, and review state survive the conversation that created
          them.
        </SectionSubtitle>
      </div>
    {/snippet}
    {#snippet end()}
      <article class="run-record" aria-label="Completed agent run waiting for review">
        <header>
          <span>Task #124</span>
          <strong>Needs review</strong>
        </header>
        <h3>Add repository ownership gates</h3>
        <p class="run-context">perceptrail · Codex · feat/repository-ownership</p>
        <h4>Work produced</h4>
        <dl>
          <div>
            <dt>launcher.go</dt>
            <dd>Modified</dd>
          </div>
          <div>
            <dt>launcher_test.go</dt>
            <dd>Added</dd>
          </div>
          <div>
            <dt>Launch policy</dt>
            <dd>Updated</dd>
          </div>
          <div>
            <dt>Repository ownership</dt>
            <dd>Decision</dd>
          </div>
        </dl>
        <footer>
          <span>Agent finished 12 min ago</span>
          <strong>Review changes →</strong>
        </footer>
      </article>
    {/snippet}
  </Split>
</Section>

<style>
:global {
  .local-section {
    display: grid;
    grid-template-columns: minmax(0, 0.76fr) minmax(620px, 1.24fr);
    gap: 56px;
    align-items: start;
  }

  .local-copy {
    display: grid;
    gap: 18px;
  }

  .local-copy .section-subtitle {
    color: var(--muted);
  }

  .run-record {
    border: 1px solid var(--line);
    border-radius: 6px;
    background: #12171d;
    padding: 28px 32px 22px;
  }

  .run-record header,
  .run-record footer,
  .run-record dl div {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 16px;
  }

  .run-record header span,
  .run-record h4,
  .run-record dd,
  .run-record footer span {
    color: #8b96a6;
    font-family: var(--font-mono);
    font-size: 12px;
    font-weight: 650;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .run-record header strong {
    color: var(--warn);
    font-family: var(--font-mono);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .run-record h3 {
    margin-top: 18px;
    font-size: 28px;
    line-height: 1.15;
  }

  .run-context {
    margin: 10px 0 0;
    color: #c5ceda;
    font-size: 16px;
  }

  .run-record h4 {
    margin: 22px 0 0;
    padding-top: 18px;
    border-top: 1px solid var(--line);
  }

  .run-record dl {
    display: grid;
    gap: 12px;
    margin: 16px 0 0;
  }

  .run-record dt {
    color: #e7edf5;
    font-size: 16px;
  }

  .run-record dd {
    margin: 0;
  }

  .run-record footer {
    margin-top: 22px;
    padding-top: 16px;
    border-top: 1px solid var(--line);
  }

  .run-record footer strong {
    color: #f4f7fb;
    font-size: 15px;
    font-weight: 680;
  }

  @media (min-width: 1181px) {
    .local-section {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1.19fr);
      column-gap: 48px;
      row-gap: 0;
      align-items: start;
    }

    .local-section .local-copy {
      grid-column: 1;
      grid-row: 2;
      gap: 16px;
    }

    .local-section .run-record {
      grid-column: 2;
      grid-row: 2;
    }

    .local-section h2 {
      max-width: none;
      line-height: 0.96;
    }

    .local-section .local-copy .section-subtitle {
      line-height: 1.4;
    }
  }

  @media (max-width: 1180px) {
    .local-section {
      grid-template-columns: 1fr;
    }
  }
}
</style>
