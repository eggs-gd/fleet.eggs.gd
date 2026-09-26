<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionText,
    SectionTitle,
    Stack
  } from '$lib/layout';
</script>

<Section class="v2-remote" id="remote" aria-labelledby="remote-title">
  {#snippet band()}
    <SectionTitle id="remote-title" size="m">
      Human input should not require sitting at the desk.
    </SectionTitle>
  {/snippet}

  <Stack gap="m">
    <SectionText size="m">
      If an agent needs me, Fleet should preserve enough state for me to see what is waiting, re-enter
      the right session, answer from the phone, and let the work continue.
    </SectionText>
    <SectionText size="m">
      The dashboard is still first-class when I am at the computer. The manager is better when the
      input is rough, remote, or cross-project.
    </SectionText>
    <SectionFoot>
      <SectionText size="xl">Both operate the same work.</SectionText>
    </SectionFoot>
  </Stack>
</Section>
