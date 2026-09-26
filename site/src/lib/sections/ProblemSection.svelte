<script lang="ts">
  import { Section, SectionFoot, SectionText, SectionTitle, Stack } from '$lib/layout';
</script>

<Section class="problem-section" id="problem" aria-labelledby="problem-title">
  {#snippet tag()}
    <SectionTitle class="problem-tag" size="m">Agent chats are easy to start.</SectionTitle>
  {/snippet}
  {#snippet band()}
    <SectionTitle id="problem-title" size="l">They're hard to operate.</SectionTitle>
  {/snippet}
  <Stack class="problem-stack" gap="0">
    <div class="question-strip" aria-label="Operating questions">
      <article>
        <strong>What is it working on?</strong>
        <span>Which repository can it touch?</span>
      </article>
      <article>
        <strong>Is something already running there?</strong>
        <span>Can another agent start safely?</span>
      </article>
      <article>
        <strong>Did it actually finish?</strong>
        <span>Where is the result?</span>
      </article>
      <article>
        <strong>Who reviewed it?</strong>
        <span>Is the work done, or did the agent merely stop talking?</span>
      </article>
    </div>
  </Stack>
  {#snippet footer()}
    <SectionFoot>
      <SectionText class="statement" size="xl">
        Fleet keeps those answers outside the chat. The agents stay in their native tools. The work
        gets a durable operating layer.
      </SectionText>
    </SectionFoot>
  {/snippet}
</Section>

<style>
:global {
  .problem-section {
    display: grid;
    gap: 40px;
  }

  .problem-tag {
    max-width: 920px;
    color: rgba(245, 247, 251, 0.42);
    line-height: 0.95;
  }

  .problem-section > .section-title:not(.problem-tag) {
    max-width: 980px;
  }

  .question-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1px;
    border-top: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
    background: transparent;
  }

  .question-strip article {
    min-height: 166px;
    border-right: 1px solid rgba(218, 226, 237, 0.11);
    background: transparent;
    padding: 24px 22px 24px 0;
  }

  .question-strip article:not(:first-child) {
    padding-left: 22px;
  }

  .question-strip article:last-child {
    border-right: 0;
  }

  .question-strip strong {
    display: block;
    color: #f2f5f9;
    font-size: 19px;
    line-height: 1.16;
  }

  .question-strip span {
    display: block;
    margin-top: 18px;
    color: #8e99a8;
    font-size: 15px;
    line-height: 1.45;
  }

  @media (min-width: 1181px) {
    .problem-section {
      margin-top: 0;
      gap: 18px;
    }

    .problem-tag {
      max-width: none;
      line-height: 1;
    }

    .problem-section > .section-title:not(.problem-tag) {
      max-width: none;
      line-height: 0.98;
    }

    .question-strip {
      border-top: 0;
      border-bottom: 0;
      margin-top: 36px;
      margin-bottom: 20px;
    }

    .question-strip article {
      min-height: 0;
      padding: 2px 20px 2px 0;
      border-right: 1px solid var(--line-strong);
    }

    .question-strip article:not(:first-child) {
      padding-left: 20px;
    }

    .question-strip strong {
      font-size: 21px;
      font-weight: 700;
      line-height: 1.25;
    }

    .question-strip span {
      margin-top: 6px;
      font-size: 18px;
      line-height: 1.35;
    }

    .problem-section .statement {
      max-width: none;
      margin-top: 8px;
    }
  }

  @media (max-width: 1180px) {
    .question-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 820px) {
    .question-strip {
      grid-template-columns: 1fr;
    }
  }
}
</style>
