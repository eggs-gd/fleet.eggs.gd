<script lang="ts">
  import { Plaque, SectionFoot, SectionText, SectionTitle, Stack } from '$lib/layout';
</script>

<Plaque class="manager-plaque" aria-labelledby="manager-plaque-title">
  <SectionTitle id="manager-plaque-title" size="s">
    All the projects it needs to understand.
  </SectionTitle>
  <Stack gap="m">
    <SectionText size="m">I have roughly 80 projects on disk.</SectionText>
    <SectionText size="m">
      I don't want to choose a project every time I have an idea. And I definitely don't want to load
      80 repositories into one model context.
    </SectionText>
    <SectionText size="m">
      The manager gets a lightweight map of the projects Fleet knows about: enough context to
      understand what each project is, what is happening there, and where new work probably belongs.
    </SectionText>
    <SectionText size="m">Then I can send:</SectionText>
    <SectionText size="l">The mobile layout breaks here.</SectionText>
    <SectionText size="m">Or a screenshot.</SectionText>
    <SectionText size="m">Or a voice note from outside.</SectionText>
    <SectionText size="m">Or a link and three half-formed sentences.</SectionText>
    <SectionText size="m">If the project is obvious, the manager routes the work there.</SectionText>
    <SectionText size="m">If it isn't, it asks.</SectionText>
    <SectionFoot>
      <SectionText size="xl">
        The manager is the front door to the work, not another project selector.
      </SectionText>
    </SectionFoot>
  </Stack>
</Plaque>
