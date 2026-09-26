<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionSubtitle,
    SectionTag,
    SectionText,
    SectionTitle,
    Split
  } from '$lib/layout';
</script>

<Section class="hero" id="product" aria-labelledby="hero-title">
  {#snippet tag()}
    <SectionTag size="m">Local control plane for coding agents</SectionTag>
  {/snippet}
  {#snippet band()}
    <SectionTitle id="hero-title" level={1} size="l">
      Keep every agent attached to the work it owns.
    </SectionTitle>
  {/snippet}
  <Split flat>
    {#snippet start()}
      <div class="hero-copy">
        <div class="hero-text">
          <SectionSubtitle size="l">
            Fleet is a local control plane for coding-agent work across real projects and
            repositories.
          </SectionSubtitle>
          <SectionText size="l">
            Use Codex, Claude Code, Cursor, Gemini, and the tools you already use. Fleet keeps the
            work around them durable: what needs to be done, who owns it, what is running, what is
            waiting, and what needs your review.
          </SectionText>
        </div>
        <SectionFoot>
          <div class="cta-row">
            <a class="primary-cta" href="https://github.com/eggs-gd/fleet.eggs.gd/releases">
              Download Fleet
            </a>
            <a class="secondary-cta" href="#install">View install notes</a>
          </div>
          <p class="price-line">First 5 completed tasks free. Then $9/month.</p>
        </SectionFoot>
      </div>
    {/snippet}
    {#snippet end()}
      <figure class="product-shot hero-shot">
        <img
          src="/product/fleet-work-dashboard.png"
          alt="Fleet work dashboard with projects, sessions, tasks, manager input, and review queues"
        />
      </figure>
    {/snippet}
  </Split>
</Section>

<style>
:global {
  .hero {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: clamp(18px, 1.35vw, 30px);
    align-items: center;
    min-height: calc(100vh - 96px);
    padding: clamp(46px, 4vw, 76px) 0 clamp(74px, 6vw, 112px);
  }

  .hero-copy {
    display: grid;
    grid-column: 1 / 6;
    gap: 0;
    min-width: 0;
  }

  .hero-text {
    display: grid;
    grid-template-columns: minmax(0, 0.86fr) minmax(0, 1.14fr);
    gap: clamp(20px, 1.4vw, 30px);
    max-width: 820px;
    margin-top: clamp(26px, 2.2vw, 42px);
  }

  .hero-text .section-subtitle {
    color: #d8e0ea;
    font-weight: 540;
  }

  .hero .cta-row {
    margin-top: 30px;
  }

  .hero .price-line {
    margin: 13px 0 0;
    color: var(--faint);
    font-size: 13px;
  }

  .hero-shot {
    grid-column: 6 / -1;
    grid-row: 1 / -1;
    align-self: center;
    width: calc(100% + clamp(36px, 5vw, 120px));
    aspect-ratio: 16 / 10;
  }

  .hero-shot img {
    object-position: center top;
  }

  @media (min-width: 1600px) {
    .hero-shot {
      aspect-ratio: 16 / 10;
    }
  }

  @media (min-width: 1181px) {
    .hero {
      grid-template-columns: minmax(0, 35%) minmax(0, 8%) minmax(0, 1fr);
      gap: 0;
      align-items: start;
      min-height: 0;
      padding: 42px 0 92px;
    }

    .hero > h1 {
      grid-column: 1 / -1;
      grid-row: 2;
      max-width: none;
      margin: 0 0 32px;
    }

    .hero-copy {
      grid-column: 1 / 3;
      grid-row: 3;
    }

    .hero-text,
    .hero .cta-row,
    .hero .price-line {
      width: 81.4%;
    }

    .hero-text {
      grid-template-columns: minmax(0, 1fr);
      gap: 8px;
      max-width: none;
      margin-top: 20px;
    }

    .hero .cta-row {
      margin-top: 22px;
    }

    .hero .price-line {
      margin-top: 14px;
    }

    .hero-shot {
      grid-column: 3;
      grid-row: 3;
      align-self: start;
      width: 100%;
      margin-top: 0;
      aspect-ratio: 16 / 10;
    }

    .hero-shot img {
      object-fit: cover;
      object-position: 64% 70%;
      transform: scale(1.4);
      transform-origin: 64% 70%;
    }
  }

  @media (max-width: 1180px) {
    .hero {
      grid-template-columns: 1fr;
    }

    .hero-copy {
      display: grid;
      gap: 0;
      max-width: 920px;
    }

    .hero-copy .eyebrow,
    .hero h1,
    .hero-text,
    .hero .cta-row,
    .hero .price-line,
    .hero-shot {
      grid-column: 1;
      grid-row: auto;
    }

    .hero-shot {
      width: 100%;
    }
  }

  @media (max-width: 820px) {
    .hero {
      gap: 32px;
      min-height: 0;
      padding: 38px 0 56px;
    }

    .hero-text {
      grid-template-columns: 1fr;
    }
  }
}
</style>
