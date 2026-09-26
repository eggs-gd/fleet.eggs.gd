<script lang="ts">
  import { Section, SectionFoot, SectionText, SectionTitle, Stack } from '$lib/layout';
</script>

<Section class="v2-workers" id="workers" aria-labelledby="workers-title">
  {#snippet band()}
    <SectionTitle id="workers-title" size="m">The agents are still the agents.</SectionTitle>
  {/snippet}

  <Stack gap="m">
    <SectionText size="m">
      Fleet does not replace Codex, Claude Code, Cursor, Gemini, or any other coding tool with
      another generic chat UI.
    </SectionText>
    <SectionText size="m">
      They remain native workers with different strengths, limits, subscriptions, and launch
      behavior.
    </SectionText>
    <SectionText size="m">
      Humans are workers too: product decisions, architecture investigation, manual unblocking,
      and review can sit in the same task graph as agent work.
    </SectionText>

    <div class="roster-scene" aria-hidden="true">
      <div class="assign-task at-1">Investigate auth architecture</div>
      <div class="assign-task at-2">Fix mobile nav regression</div>
      <div class="assign-task at-3">Decide onboarding behavior</div>
      <div class="assign-task at-4">Tune current UI interaction</div>
      <div class="roster">
      <div class="worker-card">
        <div class="wc-head"><span class="wc-icon wc-claude"><svg viewBox="0 0 24 24"><path d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z" /></svg></span><strong>Claude Code</strong></div>
        <span class="wc-role">Architecture</span>
        <span class="wc-role">Large context</span>
        <span class="wc-state"><i class="wc-dot dot-1"></i><span class="wc-label st-free st-free-1">available</span><span class="wc-label st-busy st-busy-1">busy</span></span>
        <span class="wc-badge">canonical ✓</span>
      </div>
      <div class="worker-card">
        <div class="wc-head"><span class="wc-icon wc-codex">X</span><strong>Codex</strong></div>
        <span class="wc-role">Implementation</span>
        <span class="wc-role">Focused tasks</span>
        <span class="wc-state"><i class="wc-dot dot-2"></i><span class="wc-label st-free st-free-2">available</span><span class="wc-label st-busy st-busy-2">busy</span></span>
        <span class="wc-badge">canonical ✓</span>
      </div>
      <div class="worker-card">
        <div class="wc-head"><span class="wc-icon wc-cursor"><svg viewBox="0 0 24 24"><path d="M11.503.131 1.891 5.678a.84.84 0 0 0-.42.726v11.188c0 .3.162.575.42.724l9.609 5.55a1 1 0 0 0 .998 0l9.61-5.55a.84.84 0 0 0 .42-.724V6.404a.84.84 0 0 0-.42-.726L12.497.131a1.01 1.01 0 0 0-.996 0M2.657 6.338h18.55c.263 0 .43.287.297.515L12.23 22.918c-.062.107-.229.064-.229-.06V12.335a.59.59 0 0 0-.295-.51l-9.11-5.257c-.109-.063-.064-.23.061-.23" /></svg></span><strong>Cursor</strong></div>
        <span class="wc-role">Local work</span>
        <span class="wc-role">Interactive</span>
        <span class="wc-state"><i class="wc-dot dot-busy"></i><span class="wc-label st-busy-static">active</span></span>
        <span class="wc-badge">active session</span>
      </div>
      <div class="worker-card">
        <div class="wc-head"><span class="wc-icon wc-you">Y</span><strong>You</strong></div>
        <span class="wc-role">Product</span>
        <span class="wc-role">Decisions</span>
        <span class="wc-state"><i class="wc-dot dot-4"></i><span class="wc-label st-free st-free-4">available</span><span class="wc-label st-busy st-busy-4">busy</span></span>
        <span class="wc-badge">human</span>
      </div>
      </div>
    </div>

    <SectionFoot>
      <SectionText size="xl">
        Human-in-the-loop is part of the workflow, not an exception path.
      </SectionText>
    </SectionFoot>
  </Stack>
</Section>

<style>
  :global {
    .v2-workers {
      text-align: center;
    }

    .v2-workers .section-title {
      max-width: 920px;
      margin: 0 auto;
    }

    .v2-workers .roster-scene {
      position: relative;
      height: 320px;
      border: 1px solid var(--line);
      border-radius: 6px;
      background: var(--panel);
      overflow: hidden;
      text-align: left;
    }

    .v2-workers .roster {
      position: absolute;
      left: 2%;
      right: 2%;
      bottom: 16px;
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 10px;
    }

    .v2-workers .worker-card {
      display: grid;
      gap: 3px;
      border: 1px solid var(--line-strong);
      border-radius: 4px;
      background: var(--panel-strong);
      padding: 12px;
    }

    .v2-workers .wc-head {
      display: flex;
      align-items: center;
      gap: 7px;
      margin-bottom: 4px;
    }

    .v2-workers .wc-head strong {
      color: var(--text);
      font-size: 13px;
      font-weight: 760;
    }

    .v2-workers .wc-icon {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 14px;
      height: 14px;
      font-family: var(--font-mono);
      font-size: 11px;
      font-weight: 760;
    }

    .v2-workers .wc-icon svg {
      width: 100%;
      height: 100%;
      fill: currentColor;
    }

    .v2-workers .wc-claude { color: #c98a63; }
    .v2-workers .wc-codex { color: var(--faint); }
    .v2-workers .wc-cursor { color: #b7b7c2; }
    .v2-workers .wc-you { color: var(--good); }

    .v2-workers .worker-card:nth-child(1) { --wk: #c98a63; }
    .v2-workers .worker-card:nth-child(2) { --wk: #6fb0b8; }
    .v2-workers .worker-card:nth-child(3) { --wk: #c886a6; }
    .v2-workers .worker-card:nth-child(4) { --wk: #c9bb8a; }

    .v2-workers .worker-card {
      box-shadow: inset 0 2px 0 var(--wk);
    }

    .v2-workers .wc-role {
      color: var(--wk);
      font-size: 12px;
      font-weight: 620;
    }

    .v2-workers .wc-icon {
      color: var(--wk) !important;
    }

    .v2-workers .at-1 { --wk: #c98a63; }
    .v2-workers .at-2 { --wk: #6fb0b8; }
    .v2-workers .at-3 { --wk: #c9bb8a; }
    .v2-workers .at-4 { --wk: #c886a6; }

    .v2-workers .wc-state {
      display: flex;
      align-items: center;
      gap: 6px;
      margin-top: 8px;
      color: var(--faint);
      font-family: var(--font-mono);
      font-size: 10px;
    }

    .v2-workers .wc-dot {
      flex: none;
      width: 7px;
      height: 7px;
      border-radius: 50%;
      background: var(--good);
    }

    .v2-workers .dot-busy {
      background: var(--warn);
    }

    .v2-workers .wc-label {
      display: inline-block;
    }

    .v2-workers .st-free {
      color: var(--good);
    }

    .v2-workers .st-busy,
    .v2-workers .st-busy-static {
      color: var(--warn);
    }

    .v2-workers .wc-state {
      display: grid;
      grid-template-columns: 7px auto;
      column-gap: 6px;
    }

    .v2-workers .wc-state .wc-label {
      grid-column: 2;
      grid-row: 1;
    }

    .v2-workers .st-busy {
      opacity: 0;
    }

    .v2-workers .wc-badge {
      justify-self: start;
      margin-top: 2px;
      border: 1px solid var(--line);
      border-radius: 3px;
      padding: 1px 5px;
      color: var(--faint);
      font-family: var(--font-mono);
      font-size: 9px;
    }

    .v2-workers .assign-task {
      position: absolute;
      z-index: 2;
      transform: translate(-50%, -50%);
      border: 1px solid var(--wk);
      border-radius: 4px;
      background: var(--panel-strong);
      padding: 5px 9px;
      color: var(--wk);
      font-family: var(--font-mono);
      font-size: 10px;
      white-space: nowrap;
      opacity: 0;
    }

    .v2-workers .at-1 { animation: assign-1 16s infinite; }
    .v2-workers .at-2 { animation: assign-2 16s infinite; }
    .v2-workers .at-3 { animation: assign-3 16s infinite; }
    .v2-workers .at-4 { animation: assign-4 16s infinite; }
    .v2-workers .worker-card:nth-child(1) { animation: hit-1 16s infinite; }
    .v2-workers .dot-1 { animation: dot-1 16s infinite; }
    .v2-workers .worker-card:nth-child(2) { animation: hit-2 16s infinite; }
    .v2-workers .dot-2 { animation: dot-2 16s infinite; }
    .v2-workers .worker-card:nth-child(3) { animation: hit-3 16s infinite; }
    .v2-workers .worker-card:nth-child(4) { animation: hit-4 16s infinite; }
    .v2-workers .dot-4 { animation: dot-4 16s infinite; }

    @keyframes assign-1 {
      0% { opacity: 0; left: 50%; top: 12%; }
      1.5% { opacity: 1; left: 50%; top: 12%; }
      7% { opacity: 1; left: 50%; top: 12%; }
      13% { opacity: 1; left: 14%; top: 46%; }
      22% { opacity: 1; left: 14%; top: 46%; }
      24% { opacity: 0; left: 14%; top: 46%; }
      24.5% { opacity: 0; left: 50%; top: 12%; }
      100% { opacity: 0; left: 50%; top: 12%; }
    }
    @keyframes assign-2 {
      0% { opacity: 0; left: 50%; top: 12%; }
      25% { opacity: 0; left: 50%; top: 12%; }
      26.5% { opacity: 1; left: 50%; top: 12%; }
      32% { opacity: 1; left: 50%; top: 12%; }
      38% { opacity: 1; left: 38%; top: 46%; }
      47% { opacity: 1; left: 38%; top: 46%; }
      49% { opacity: 0; left: 38%; top: 46%; }
      49.5% { opacity: 0; left: 50%; top: 12%; }
      100% { opacity: 0; left: 50%; top: 12%; }
    }
    @keyframes assign-3 {
      0% { opacity: 0; left: 50%; top: 12%; }
      50% { opacity: 0; left: 50%; top: 12%; }
      51.5% { opacity: 1; left: 50%; top: 12%; }
      57% { opacity: 1; left: 50%; top: 12%; }
      63% { opacity: 1; left: 86%; top: 46%; }
      72% { opacity: 1; left: 86%; top: 46%; }
      74% { opacity: 0; left: 86%; top: 46%; }
      74.5% { opacity: 0; left: 50%; top: 12%; }
      100% { opacity: 0; left: 50%; top: 12%; }
    }
    @keyframes assign-4 {
      0% { opacity: 0; left: 50%; top: 12%; }
      75% { opacity: 0; left: 50%; top: 12%; }
      76.5% { opacity: 1; left: 50%; top: 12%; }
      82% { opacity: 1; left: 50%; top: 12%; }
      88% { opacity: 1; left: 62%; top: 46%; }
      97% { opacity: 1; left: 62%; top: 46%; }
      99% { opacity: 0; left: 62%; top: 46%; }
      99.5% { opacity: 0; left: 50%; top: 12%; }
      100% { opacity: 0; left: 50%; top: 12%; }
    }

    @keyframes hit-1 {
      0%,12% { border-color: var(--line-strong); }
      14%,22% { border-color: var(--wk); }
      24%,100% { border-color: var(--line-strong); }
    }
    @keyframes dot-1 {
      0%,12% { background: var(--good); }
      14%,22% { background: var(--warn); }
      24%,100% { background: var(--good); }
    }
    @keyframes hit-2 {
      0%,37% { border-color: var(--line-strong); }
      39%,47% { border-color: var(--wk); }
      49%,100% { border-color: var(--line-strong); }
    }
    @keyframes dot-2 {
      0%,37% { background: var(--good); }
      39%,47% { background: var(--warn); }
      49%,100% { background: var(--good); }
    }
    @keyframes hit-3 {
      0%,87% { border-color: var(--line-strong); }
      89%,97% { border-color: var(--wk); }
      99%,100% { border-color: var(--line-strong); }
    }
    @keyframes dot-3 {
      0%,87% { background: var(--good); }
      89%,97% { background: var(--warn); }
      99%,100% { background: var(--good); }
    }
    @keyframes hit-4 {
      0%,62% { border-color: var(--line-strong); }
      64%,72% { border-color: var(--wk); }
      74%,100% { border-color: var(--line-strong); }
    }
    @keyframes dot-4 {
      0%,62% { background: var(--good); }
      64%,72% { background: var(--warn); }
      74%,100% { background: var(--good); }
    }

    .v2-workers .st-free-1 { animation: free-1 16s infinite; }
    .v2-workers .st-busy-1 { animation: busy-1 16s infinite; }
    @keyframes free-1 {
      0%,13% { opacity: 1; }
      14%,22% { opacity: 0; }
      23%,100% { opacity: 1; }
    }
    @keyframes busy-1 {
      0%,13% { opacity: 0; }
      14%,22% { opacity: 1; }
      23%,100% { opacity: 0; }
    }
    .v2-workers .st-free-2 { animation: free-2 16s infinite; }
    .v2-workers .st-busy-2 { animation: busy-2 16s infinite; }
    @keyframes free-2 {
      0%,38% { opacity: 1; }
      39%,47% { opacity: 0; }
      48%,100% { opacity: 1; }
    }
    @keyframes busy-2 {
      0%,38% { opacity: 0; }
      39%,47% { opacity: 1; }
      48%,100% { opacity: 0; }
    }
    .v2-workers .st-free-4 { animation: free-4 16s infinite; }
    .v2-workers .st-busy-4 { animation: busy-4 16s infinite; }
    @keyframes free-4 {
      0%,63% { opacity: 1; }
      64%,72% { opacity: 0; }
      73%,100% { opacity: 1; }
    }
    @keyframes busy-4 {
      0%,63% { opacity: 0; }
      64%,72% { opacity: 1; }
      73%,100% { opacity: 0; }
    }

    @media (prefers-reduced-motion: reduce) {
      .v2-workers .assign-task,
      .v2-workers .worker-card,
      .v2-workers .wc-dot,
      .v2-workers .wc-label {
        animation: none !important;
      }
    }

    @media (max-width: 720px) {
      .v2-workers .roster-scene {
        height: 430px;
      }
      .v2-workers .roster {
        grid-template-columns: repeat(2, minmax(0, 1fr));
        bottom: 12px;
      }
      @keyframes assign-1 {
        0% { opacity: 0; left: 50%; top: 8%; }
        1.5% { opacity: 1; left: 50%; top: 8%; }
        7% { opacity: 1; left: 50%; top: 8%; }
        13% { opacity: 1; left: 26%; top: 37%; }
        22% { opacity: 1; left: 26%; top: 37%; }
        24% { opacity: 0; left: 26%; top: 37%; }
        24.5% { opacity: 0; left: 50%; top: 8%; }
        100% { opacity: 0; left: 50%; top: 8%; }
      }
      @keyframes assign-2 {
        0% { opacity: 0; left: 50%; top: 8%; }
        25% { opacity: 0; left: 50%; top: 8%; }
        26.5% { opacity: 1; left: 50%; top: 8%; }
        32% { opacity: 1; left: 50%; top: 8%; }
        38% { opacity: 1; left: 74%; top: 37%; }
        47% { opacity: 1; left: 74%; top: 37%; }
        49% { opacity: 0; left: 74%; top: 37%; }
        49.5% { opacity: 0; left: 50%; top: 8%; }
        100% { opacity: 0; left: 50%; top: 8%; }
      }
      @keyframes assign-3 {
        0% { opacity: 0; left: 50%; top: 8%; }
        50% { opacity: 0; left: 50%; top: 8%; }
        51.5% { opacity: 1; left: 50%; top: 8%; }
        57% { opacity: 1; left: 50%; top: 8%; }
        63% { opacity: 1; left: 74%; top: 67%; }
        72% { opacity: 1; left: 74%; top: 67%; }
        74% { opacity: 0; left: 74%; top: 67%; }
        74.5% { opacity: 0; left: 50%; top: 8%; }
        100% { opacity: 0; left: 50%; top: 8%; }
      }
      @keyframes assign-4 {
        0% { opacity: 0; left: 50%; top: 8%; }
        75% { opacity: 0; left: 50%; top: 8%; }
        76.5% { opacity: 1; left: 50%; top: 8%; }
        82% { opacity: 1; left: 50%; top: 8%; }
        88% { opacity: 1; left: 26%; top: 67%; }
        97% { opacity: 1; left: 26%; top: 67%; }
        99% { opacity: 0; left: 26%; top: 67%; }
        99.5% { opacity: 0; left: 50%; top: 8%; }
        100% { opacity: 0; left: 50%; top: 8%; }
      }
    }
  }
</style>
