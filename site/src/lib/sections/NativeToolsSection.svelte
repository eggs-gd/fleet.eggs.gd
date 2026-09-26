<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionSubtitle,
    SectionTag,
    SectionText,
    SectionTitle,
    Split
  } from '$lib/layout';
</script>

<Section class="native-section" aria-labelledby="native-title">
  {#snippet tag()}
    <SectionTag size="m">Native tools</SectionTag>
  {/snippet}
  <Split flat>
    {#snippet start()}
      <div class="native-copy">
        <SectionTitle id="native-title" size="s">Your agents stay your agents.</SectionTitle>
        <SectionSubtitle size="s">
          Fleet does not replace Codex, Claude Code, Cursor, or Gemini. It doesn't hide them behind
          another generic chat interface.
        </SectionSubtitle>
        <SectionText size="s">
          Coding stays where it already works best. Fleet sits around those tools and coordinates
          the work between them.
        </SectionText>
        <SectionFoot>
          <strong>Native tools stay native. Fleet keeps the operating state.</strong>
        </SectionFoot>
      </div>
    {/snippet}
    {#snippet end()}
      <div class="agent-strip" aria-label="Native agent sessions">
        <article>
          <span>Claude Code</span>
          <p>&gt; working on docs<br />&gt; repository: core<br />&gt; branch: launch-policy</p>
          <strong>remote</strong>
        </article>
        <article>
          <span>Codex</span>
          <p>&gt; running tests<br />&gt; repository: perceptrail<br />&gt; branch: agent-launch</p>
          <strong>local</strong>
        </article>
        <article>
          <span>Cursor</span>
          <p>&gt; editing files<br />&gt; repository: career-wizard<br />&gt; branch: compact-cards</p>
          <strong>cloud</strong>
        </article>
        <article>
          <span>Gemini CLI</span>
          <p>&gt; analyzing<br />&gt; repository: life-os<br />&gt; branch: research</p>
          <strong>shell</strong>
        </article>
      </div>
    {/snippet}
  </Split>
</Section>

<style>
:global {
  .native-section {
    display: grid;
    grid-template-columns: minmax(0, 0.82fr) minmax(560px, 1.18fr);
    gap: 56px;
    align-items: center;
  }

  .native-copy {
    display: grid;
    gap: 18px;
  }

  .native-copy strong {
    display: block;
    color: #eef2f7;
    font-size: 17px;
  }

  .native-copy .section-subtitle,
  .native-copy .section-text {
    color: var(--muted);
  }

  .agent-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    background: transparent;
  }

  .agent-strip article {
    display: grid;
    align-content: start;
    min-height: 220px;
    border: 1px solid rgba(218, 226, 237, 0.16);
    border-radius: 7px;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent 38%),
      #10151b;
    padding: 16px;
  }

  .agent-strip article::before {
    content: "";
    width: 42px;
    height: 9px;
    margin-bottom: 22px;
    border-radius: 999px;
    background:
      radial-gradient(circle at 4px 50%, #ff6565 0 3px, transparent 4px),
      radial-gradient(circle at 20px 50%, #f5d46b 0 3px, transparent 4px),
      radial-gradient(circle at 36px 50%, #9ce7b2 0 3px, transparent 4px);
  }

  .agent-strip span {
    color: #7c8797;
    font-family: var(--font-mono);
    font-size: 11px;
    text-transform: uppercase;
  }

  .agent-strip strong {
    align-self: end;
    color: #f4f7fb;
    font-size: 21px;
    line-height: 1;
  }

  .agent-strip p {
    margin: 18px 0 0;
    color: #8f9aa9;
    font-family: var(--font-mono);
    font-size: 12px;
    line-height: 1.68;
  }

  @media (min-width: 1181px) {
    .native-section {
      grid-template-columns: minmax(0, 30%) minmax(0, 70%);
      column-gap: 32px;
      row-gap: 0;
      align-items: start;
    }

    .native-section .native-copy {
      grid-column: 1;
      grid-row: 2;
      gap: 8px;
    }

    .native-section .agent-strip {
      grid-column: 2;
      grid-row: 2;
      gap: 10px;
    }

    .native-section .native-copy > * {
      margin: 0;
    }

    .native-section h2 {
      max-width: none;
      line-height: 0.98;
    }

    .native-section .native-copy .section-subtitle,
    .native-section .native-copy .section-text {
      line-height: 1.35;
    }

    .native-section .native-copy .section-foot strong,
    .native-section .native-copy strong {
      font-size: 14px;
    }

    .native-section .agent-strip article {
      min-height: 0;
      padding: 16px 12px 14px;
    }

    .native-section .agent-strip article::before {
      width: 58px;
      height: 12px;
      margin-bottom: 12px;
      background:
        radial-gradient(circle at 6px 50%, #ff6565 0 4px, transparent 5px),
        radial-gradient(circle at 28px 50%, #f5d46b 0 4px, transparent 5px),
        radial-gradient(circle at 50px 50%, #9ce7b2 0 4px, transparent 5px);
    }

    .native-section .agent-strip span {
      color: #e7edf5;
      font-family: var(--font-sans);
      font-size: 16px;
      font-weight: 680;
      letter-spacing: 0;
      text-transform: none;
    }

    .native-section .agent-strip p {
      margin-top: 10px;
      font-size: 12px;
      line-height: 1.45;
    }

    .native-section .agent-strip strong {
      align-self: start;
      margin-top: 10px;
      font-size: 16px;
    }
  }

  @media (max-width: 1180px) {
    .native-section {
      grid-template-columns: 1fr;
    }

    .agent-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 820px) {
    .agent-strip {
      grid-template-columns: 1fr;
    }
  }
}
</style>
