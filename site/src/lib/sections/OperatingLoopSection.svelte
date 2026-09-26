<script lang="ts">
  import { Section, SectionFoot, SectionTag, SectionText, SectionTitle, Stack } from '$lib/layout';

  type PipelineStep = {
    number: string;
    title: string;
    body: string;
    meta: string;
  };

  const pipeline: PipelineStep[] = [
    {
      number: '01',
      title: 'Capture',
      body: 'Start with a voice note, sentence, link, or half-formed idea.',
      meta: 'raw input'
    },
    {
      number: '02',
      title: 'Shape',
      body: 'Attach the work to a project, repository, owner, dependencies, and priority.',
      meta: 'ready task'
    },
    {
      number: '03',
      title: 'Launch',
      body: 'Check ownership, active sessions, and repository conflicts before handoff.',
      meta: 'safe run'
    },
    {
      number: '04',
      title: 'Review',
      body: 'Bring artifacts back for human acceptance, rework, or unblock.',
      meta: 'accepted done'
    }
  ];
</script>

<Section class="workflow-section" id="workflow" aria-labelledby="workflow-title">
  {#snippet tag()}
    <SectionTag size="m">Operating loop</SectionTag>
  {/snippet}
  {#snippet band()}
    <SectionTitle id="workflow-title" size="s">
      Fleet turns rough input into work that can survive beyond a conversation.
    </SectionTitle>
  {/snippet}
  <Stack class="workflow-stack" gap="0">
    <div class="pipeline" aria-label="Fleet operating pipeline">
      {#each pipeline as step}
        <article class="pipeline-step">
          <span>{step.number}</span>
          <small>{step.meta}</small>
          <h3>{step.title}</h3>
          <SectionText size="s">{step.body}</SectionText>
        </article>
      {/each}
    </div>
    <figure class="workflow-shot">
      <img
        src="/product/core-151-task.png"
        alt="Task CORE-151 in the Fleet list: Require standalone Codex CLI for launcher"
      />
    </figure>
  </Stack>
  {#snippet footer()}
    <SectionFoot>
      <SectionText class="workflow-note" size="xl">
        Done means accepted by a human. No silent swarm touching the same codebase.
      </SectionText>
    </SectionFoot>
  {/snippet}
</Section>

<style>
:global {
  .pipeline {
    position: relative;
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0;
    min-height: 540px;
    border-top: 1px solid var(--line-strong);
    border-bottom: 1px solid var(--line);
  }

  .pipeline::before {
    content: "";
    position: absolute;
    top: 118px;
    right: 0;
    left: 0;
    height: 1px;
    background: rgba(218, 226, 237, 0.26);
  }

  .pipeline::after {
    content: "";
    position: absolute;
    right: 10%;
    bottom: 184px;
    left: 7%;
    height: 2px;
    background:
      linear-gradient(90deg, rgba(156, 231, 178, 0), rgba(156, 231, 178, 0.82) 32%, rgba(156, 231, 178, 0.16) 72%, rgba(156, 231, 178, 0));
    box-shadow: 0 0 28px rgba(156, 231, 178, 0.12);
  }

  .pipeline-step {
    position: relative;
    display: grid;
    align-content: start;
    min-height: 300px;
    border-right: 1px solid rgba(218, 226, 237, 0.11);
    padding: 28px 24px;
  }

  .pipeline-step::before {
    content: "";
    position: absolute;
    top: 113px;
    left: 24px;
    z-index: 1;
    width: 11px;
    height: 11px;
    border: 1px solid rgba(156, 231, 178, 0.62);
    border-radius: 999px;
    background: #0c0e11;
  }

  .pipeline-step:last-of-type {
    border-right: 0;
  }

  .pipeline-step span {
    color: #f4f7fb;
    font-family: var(--font-mono);
    font-size: 22px;
  }

  .pipeline-step small {
    margin-top: 52px;
    color: #778393;
    font-family: var(--font-mono);
    font-size: 11px;
    text-transform: uppercase;
  }

  .pipeline-step h3 {
    margin-top: 16px;
    font-size: 28px;
  }

  .pipeline-step .section-text {
    max-width: 270px;
    margin: 14px 0 0;
    color: #9ea8b6;
  }

  .workflow-note {
    margin: 26px 0 0;
    color: #dbe2eb;
    font-weight: 650;
  }

  @media (min-width: 1181px) {
    .workflow-section {
      display: grid;
      grid-template-columns: minmax(0, 1fr);
      align-items: start;
    }

    .workflow-section > h2 {
      grid-row: 2;
      max-width: none;
      margin: 0 0 28px;
      line-height: 1.12;
    }

    .workflow-section .workflow-note {
      grid-row: 5;
      max-width: none;
      margin: 20px 0 0;
      line-height: 1.4;
    }

    .workflow-section .pipeline {
      position: relative;
      grid-column: 1;
      grid-row: 3;
      width: 100%;
      min-height: 0;
      border: 0;
      padding-bottom: 0;
    }

    .workflow-section .workflow-shot {
      grid-row: 4;
      width: 100%;
      margin: 32px 0 0;
    }

    .workflow-section .workflow-shot img {
      display: block;
      width: 100%;
      height: auto;
      border: 1px solid rgba(218, 226, 237, 0.18);
      border-radius: 6px;
    }

    .workflow-section .pipeline::before {
      top: 30px;
      background: var(--line-strong);
    }

    .workflow-section .pipeline::after {
      display: none;
    }

    .workflow-section .pipeline-step {
      grid-row: 1;
      min-height: 0;
      padding: 0 18px 0 0;
    }

    .workflow-section .pipeline-step::before {
      display: none;
    }

    .workflow-section .pipeline-step:not(:nth-child(4))::after {
      content: "→";
      position: absolute;
      top: 1px;
      right: 8px;
      color: #8b96a6;
      font-size: 14px;
      line-height: 1;
    }

    .workflow-section .pipeline-step span {
      font-size: 15px;
    }

    .workflow-section .pipeline-step small {
      margin-top: 28px;
    }

    .workflow-section .pipeline-step h3 {
      margin-top: 14px;
      font-size: 22px;
    }

    .workflow-section .pipeline-step .section-text {
      max-width: none;
      margin-top: 12px;
    }
  }

  @media (max-width: 820px) {
    .pipeline {
      display: grid;
      grid-template-columns: 1fr;
      min-height: 0;
    }

    .pipeline::before,
    .pipeline::after {
      display: none;
    }

    .pipeline-step {
      min-height: 0;
      border-right: 0;
      border-bottom: 1px solid rgba(218, 226, 237, 0.11);
    }

    .pipeline-step small {
      margin-top: 22px;
    }
  }
}
</style>
