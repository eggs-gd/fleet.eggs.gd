<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionText,
    SectionTitle,
    Stack
  } from '$lib/layout';
</script>

<Section class="v2-human" id="human" aria-labelledby="human-title">
  {#snippet band()}
    <SectionTitle id="human-title" size="m">Humans are workers too</SectionTitle>
  {/snippet}
  <Stack gap="m">
    <SectionText size="m">
      Fleet does not require every node in a task graph to be an AI agent.
    </SectionText>
    <SectionText size="m">I registered myself as a worker.</SectionText>
    <SectionText size="m">That means a project can contain a sequence like:</SectionText>
    <pre class="v2-diagram">Human — decide product behaviour
              ↓
Claude — investigate affected architecture
              ↓
Codex — implement the change
              ↓
Human — review</pre>
    <SectionText size="m">
      If a feature needs product thinking before implementation, the manager can assign that
      thinking to me and lock downstream work behind it.
    </SectionText>
    <SectionText size="m">
      I make the decision, attach the context or artifact, complete my task, and the rest becomes
      executable.
    </SectionText>
    <SectionText size="m">The same mechanism works for any number of human workers.</SectionText>
    <SectionFoot>
      <SectionText size="xl">
        Human-in-the-loop is not an exception path. It is part of the task graph.
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
