<script lang="ts">
  import { Section, SectionFoot, SectionText, SectionTitle, Stack } from '$lib/layout';

  let shot: HTMLElement;
  let seen = $state(false);

  $effect(() => {
    const io = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          seen = true;
          io.disconnect();
        }
      },
      { threshold: 0.4 }
    );
    io.observe(shot);
    return () => io.disconnect();
  });
</script>

<Section class="v2-dashboard" id="dashboard" aria-labelledby="dashboard-title">
  {#snippet band()}
    <SectionTitle id="dashboard-title" size="m">
      Chat when the input is rough. Dashboard when the work is exact.
    </SectionTitle>
  {/snippet}

  <Stack gap="m">
    <SectionText size="m">
      The dashboard is where exact work gets operated directly: assign workers, change state, add
      context, launch tasks, review results, reopen, close, archive.
    </SectionText>

    <figure class="product-shot dashboard-shot" class:seen bind:this={shot}>
      <img
        src="/product/fleet-work-dashboard.png"
        alt="Full Fleet dashboard with projects, tasks, sessions, ownership, attention, and review"
      />
      <span class="callout c1" aria-hidden="true"><b>01</b><i></i><em>sessions</em></span>
      <span class="callout c2" aria-hidden="true"><b>02</b><i></i><em>projects</em></span>
      <span class="callout c3" aria-hidden="true"><b>03</b><i></i><em>current work</em></span>
      <span class="callout c4 flip" aria-hidden="true"
        ><b>04</b><i></i><em>needs attention</em></span
      >
      <span class="callout c5 flip" aria-hidden="true"
        ><b>05</b><i></i><em>recent completed</em></span
      >
      <span class="callout c6" aria-hidden="true"><b>06</b><i></i><em>manager</em></span>
    </figure>

    <SectionFoot>
      <SectionText size="xl">
        One place to see projects, tasks, sessions, ownership, attention, and review.
      </SectionText>
    </SectionFoot>
  </Stack>
</Section>

<style>
  :global {
    .v2-dashboard {
      --acid: var(--good);
      text-align: center;
    }

    .v2-dashboard .section-title {
      max-width: 920px;
      margin: 0 auto;
    }

    .v2-dashboard .dashboard-shot {
      width: 100%;
      aspect-ratio: 16 / 10;
    }

    .v2-dashboard .dashboard-shot img {
      object-fit: contain;
      object-position: left top;
    }

    .v2-dashboard .dashboard-shot {
      container-type: inline-size;
    }

    .v2-dashboard .callout {
      position: absolute;
      z-index: 2;
      display: flex;
      align-items: center;
      font-size: max(10px, 1.3cqw);
      opacity: 0;
      transition: opacity 0.6s ease;
    }

    .v2-dashboard .callout.flip {
      flex-direction: row-reverse;
      transform: translateX(-100%);
    }

    .v2-dashboard .callout b {
      border: 0.14em solid var(--acid);
      border-radius: 0.25em;
      background: #0d1117;
      padding: 0.2em 0.55em;
      color: var(--acid);
      font-family: var(--font-mono);
      font-size: 1.1em;
      font-weight: 800;
      box-shadow:
        0 0 0 0.1em rgba(6, 10, 6, 0.85),
        0 0 1.1em rgba(156, 231, 178, 0.75);
    }

    .v2-dashboard .callout i {
      width: 1.4em;
      height: 0.18em;
      background: var(--acid);
      box-shadow: 0 0 0.7em rgba(156, 231, 178, 0.9);
    }

    .v2-dashboard .callout em {
      border: 0.14em solid var(--acid);
      border-radius: 0.25em;
      background: #070b07;
      padding: 0.3em 0.75em;
      color: var(--acid);
      font-family: var(--font-mono);
      font-size: 1em;
      font-style: normal;
      font-weight: 800;
      white-space: nowrap;
      box-shadow:
        0 0 0 0.1em rgba(6, 10, 6, 0.85),
        0 0 1.1em rgba(156, 231, 178, 0.6),
        0 0.3em 1.6em rgba(0, 0, 0, 0.9);
    }

    .v2-dashboard .c1 {
      left: 38%;
      top: 10.8%;
    }
    .v2-dashboard .c2 {
      left: 5%;
      top: 33%;
    }
    .v2-dashboard .c3 {
      left: 38%;
      top: 62%;
    }
    .v2-dashboard .c4 {
      left: 90%;
      top: 16%;
    }
    .v2-dashboard .c5 {
      left: 90%;
      top: 47%;
    }
    .v2-dashboard .c6 {
      left: 46%;
      top: 87%;
    }

    .v2-dashboard .seen .c1 {
      opacity: 1;
      transition-delay: 0.3s;
    }
    .v2-dashboard .seen .c2 {
      opacity: 1;
      transition-delay: 1.2s;
    }
    .v2-dashboard .seen .c3 {
      opacity: 1;
      transition-delay: 2.1s;
    }
    .v2-dashboard .seen .c4 {
      opacity: 1;
      transition-delay: 3s;
    }
    .v2-dashboard .seen .c5 {
      opacity: 1;
      transition-delay: 3.9s;
    }
    .v2-dashboard .seen .c6 {
      opacity: 1;
      transition-delay: 4.8s;
    }

    @media (prefers-reduced-motion: reduce) {
      .v2-dashboard .callout {
        opacity: 1;
        transition: none;
      }
    }

    @media (max-width: 720px) {
      .v2-dashboard .callout {
        font-size: 12px;
      }
      .v2-dashboard .callout em {
        display: none;
      }
      .v2-dashboard .callout i {
        display: none;
      }
    }
  }
</style>
