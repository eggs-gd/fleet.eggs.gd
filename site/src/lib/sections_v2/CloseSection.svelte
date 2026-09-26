<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionText,
    SectionTitle,
    Stack
  } from '$lib/layout';
</script>

<Section class="v2-close" id="close" aria-labelledby="close-title">
  {#snippet band()}
    <SectionTitle id="close-title" size="m">That's Fleet</SectionTitle>
  {/snippet}
  <Stack gap="m">
    <SectionText size="m">
      It started because I wanted to give coding agents work from my phone.
    </SectionText>
    <SectionText size="m">Then managing separate agents became another job.</SectionText>
    <SectionText size="m">So I made one manager.</SectionText>
    <SectionText size="m">
      The manager needed enough context to understand all my projects.
    </SectionText>
    <SectionText size="m">Then it needed to turn rough intent into properly shaped work.</SectionText>
    <SectionText size="m">
      Then that work needed dependencies, workers, sessions, ownership, state, review, and human
      intervention.
    </SectionText>
    <SectionText size="m">
      And all of that needed something deterministic underneath it so the intelligent part could not
      accidentally turn execution into chaos.
    </SectionText>
    <SectionText size="m">So I built Fleet.</SectionText>
    <SectionFoot>
      <SectionText size="xl">
        One manager above the work. Real coding agents underneath. Fleet in between keeping execution
        honest.
      </SectionText>
    </SectionFoot>
  </Stack>
</Section>
