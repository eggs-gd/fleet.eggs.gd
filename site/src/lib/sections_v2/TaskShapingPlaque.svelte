<script lang="ts">
  import { Plaque, SectionFoot, SectionText, SectionTitle, Stack } from '$lib/layout';
</script>

<Plaque class="shaping-plaque" id="shaping" aria-labelledby="shaping-plaque-title">
  <SectionTitle id="shaping-plaque-title" size="s">
    I also got tired of writing three-page prompts for coding agents.
  </SectionTitle>
  <Stack gap="m">
    <SectionText size="m">A coding agent can execute a good task remarkably well.</SectionText>
    <SectionText size="m">Writing the good task is still work.</SectionText>
    <SectionText size="m">
      Sometimes I know exactly what needs to happen. Sometimes I have a screenshot and approximately
      three sentences worth of patience.
    </SectionText>
    <SectionText size="m">
      The manager uses project context to turn that rough intent into work an executor can actually
      pick up: scope, relevant context, constraints, expected result, dependencies, and a sensible
      worker.
    </SectionText>
    <SectionText size="m">When one task is enough, it creates one task.</SectionText>
    <SectionText size="m">When the work needs decomposition, it creates a chain.</SectionText>
    <SectionText size="m">
      When there is not enough information to specify implementation safely, it does
      <strong>not</strong> have to invent the missing context.
    </SectionText>
    <SectionText size="m">It can create the missing thinking first.</SectionText>
    <pre class="v2-diagram">Investigate current authentication flow
                ↓
Document constraints and proposed change
                ↓
Implement the change
                ↓
Verify migration</pre>
    <SectionText size="m">
      The implementation tasks stay locked by the investigation until the prerequisite is complete.
    </SectionText>
    <SectionText size="m">That first task can go to an agent.</SectionText>
    <SectionText size="m">Or it can go to me.</SectionText>
    <SectionFoot>
      <SectionText size="xl">Three sentences in. A usable task graph out.</SectionText>
    </SectionFoot>
  </Stack>
</Plaque>

<style>
  :global {
    .shaping-plaque .v2-diagram {
      margin: 0;
      padding: 14px 12px;
      border: 1px solid var(--line);
      border-radius: 4px;
      background: var(--panel-strong);
      color: var(--muted);
      font-family: var(--font-mono);
      font-size: 11px;
      line-height: 1.45;
      white-space: pre-wrap;
      overflow-x: auto;
    }

    .shaping-plaque .section-text.size-xl {
      font-size: var(--type-text-l);
      line-height: 1.35;
    }
  }
</style>
