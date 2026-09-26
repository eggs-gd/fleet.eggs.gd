<script lang="ts">
  import { Plaque, SectionFoot, SectionText, SectionTitle, Stack } from '$lib/layout';
</script>

<Plaque class="lifecycle-plaque" id="lifecycle" aria-labelledby="lifecycle-plaque-title">
  <SectionTitle id="lifecycle-plaque-title" size="s">The manager runs the lifecycle too</SectionTitle>
  <Stack gap="m">
    <SectionText size="m">Creating a task is only the beginning.</SectionText>
    <SectionText size="m">The same manager can operate the work after intake:</SectionText>
    <ul class="v2-list">
      <li>put something in the backlog or make it ready now;</li>
      <li>choose a worker or use the configured default;</li>
      <li>start a task;</li>
      <li>add context, comments, screenshots, or links;</li>
      <li>tell me what is running, blocked, or waiting for review;</li>
      <li>reopen, close, or archive work;</li>
      <li>choose the next ready tasks by priority and launch them.</li>
    </ul>
    <SectionText size="m">If a choice is ambiguous, the manager can ask instead of guessing.</SectionText>
    <SectionText size="m">The point is not to replace the UI with chat.</SectionText>
    <SectionFoot>
      <SectionText size="xl">
        The point is that I should be able to operate the same system whether I am at the desk or
        holding a phone.
      </SectionText>
    </SectionFoot>
  </Stack>
</Plaque>

<style>
  :global {
    .lifecycle-plaque .v2-list {
      margin: 0;
      padding-left: 1.2em;
      color: var(--muted);
      font-size: var(--type-text-m);
      line-height: 1.55;
    }

    .lifecycle-plaque .v2-list li + li {
      margin-top: 8px;
    }

    .lifecycle-plaque .section-text.size-xl {
      font-size: var(--type-text-l);
      line-height: 1.35;
    }
  }
</style>
