<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionText,
    SectionTitle,
    Stack
  } from '$lib/layout';
</script>

<Section class="v2-attention" id="attention" aria-labelledby="attention-title">
  {#snippet band()}
    <SectionTitle id="attention-title" size="m">Blocked is not done</SectionTitle>
  {/snippet}
  <Stack gap="m">
    <SectionText size="m">This sounds obvious until an orchestrator gets it wrong.</SectionText>
    <SectionText size="m">Sometimes an agent cannot continue without a human decision.</SectionText>
    <SectionText size="m">That task is not finished.</SectionText>
    <SectionText size="m">The repository still belongs to it.</SectionText>
    <SectionText size="m">The session still belongs to it.</SectionText>
    <SectionText size="m">The next task must not start.</SectionText>
    <SectionText size="m">
      Fleet keeps ownership while the work is waiting for human input.
    </SectionText>
    <pre class="v2-diagram">RUNNING
   ↓
NEEDS ATTENTION
   ↓
RUNNING
   ↓
NEEDS REVIEW
   ↓
DONE</pre>
    <SectionText size="m">Only the appropriate transition releases the work.</SectionText>
    <SectionFoot>
      <SectionText size="xl">
        Otherwise "human in the loop" turns into two workers touching the same repository while the
        first one is still waiting for an answer.
      </SectionText>
    </SectionFoot>
  </Stack>
</Section>

<style>
  :global {
    .v2-diagram {
      margin: 0;
      padding: 18px 20px;
      border: 1px solid var(--line);
      border-radius: 4px;
      background: var(--panel);
      color: var(--muted);
      font-family: var(--font-mono);
      font-size: 13px;
      line-height: 1.55;
      white-space: pre;
      overflow-x: auto;
    }
  }
</style>
