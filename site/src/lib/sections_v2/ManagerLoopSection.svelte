<script lang="ts">
  import { Section, SectionFoot, SectionText, SectionTitle, Stack } from '$lib/layout';
</script>

<Section class="v2-loop" id="manager-loop" aria-labelledby="manager-loop-title">
  {#snippet band()}
    <SectionTitle id="manager-loop-title" size="m">
      The manager is an agent you already use.
    </SectionTitle>
  {/snippet}

  <Stack gap="l">
    <SectionText size="l">One conversation with the whole operation.</SectionText>

    <div class="loop-scene" aria-hidden="true">
      <div class="loop-chat">
        <span class="loop-label">Manager</span>
        <div class="loop-window">
          <p class="loop-who">Claude Code · dedicated session</p>
          <div class="thread">
            <p class="bubble human b-send">
              Add Windows support. Start with an investigation if needed.
            </p>
            <p class="bubble fleet b-attn">
              <b>Fleet · Needs Attention</b>
              Worker needs a decision before continuing.
            </p>
            <p class="bubble human b-reply">Use native process APIs.</p>
            <p class="bubble fleet b-review">
              <b>Fleet · Needs Review</b>
              Windows support is ready for review.
            </p>
          </div>
          <div class="composer">
            <span class="draft draft-send">
              Add Windows support. Start with an investigation if needed.
            </span>
            <span class="draft draft-reply">Use native process APIs.</span>
            <span class="send-mark"></span>
          </div>
        </div>
      </div>

      <div class="loop-side">
        <span class="loop-label">Fleet</span>
        <div class="loop-board">
          <p class="dock">
            <i>Review</i>
            <span>Ready for review</span>
          </p>
        </div>
      </div>

      <div class="shuttle">
        <p class="face as-msg">Add Windows support. Start with an investigation if needed.</p>
        <p class="face as-task">
          <i>Running</i>
          <span>Task created · Codex</span>
        </p>
        <p class="face as-warn">
          <i>Needs Attention</i>
          <span>Worker needs a decision before continuing.</span>
        </p>
      </div>

      <div class="green">
        <p class="face as-go">
          <i>Resumed</i>
          <span>Context updated · worker continues</span>
        </p>
        <p class="face as-review">
          <i>Review</i>
          <span>Ready for review</span>
        </p>
      </div>
    </div>

    <SectionFoot>
      <SectionText size="xl">
        When the work needs you, it comes back to the same conversation.
      </SectionText>
    </SectionFoot>
  </Stack>
</Section>

<style>
  :global {
    .v2-loop {
      text-align: center;
    }

    .v2-loop .section-title {
      max-width: 920px;
      margin: 0 auto;
    }

    .v2-loop .loop-scene {
      position: relative;
      display: grid;
      grid-template-columns: minmax(0, 1.4fr) minmax(220px, 0.8fr);
      gap: 28px;
      width: min(100%, 920px);
      margin: 8px auto 0;
      text-align: left;
    }

    .v2-loop .loop-label {
      display: block;
      margin-bottom: 8px;
      color: var(--faint);
      font-size: 11px;
      font-weight: 760;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }

    .v2-loop .loop-window,
    .v2-loop .loop-board {
      border: 1px solid var(--line-strong);
      border-radius: 4px;
      background: var(--panel-strong);
    }

    .v2-loop .loop-window {
      display: flex;
      flex-direction: column;
      gap: 10px;
      height: 400px;
      padding: 14px;
    }

    .v2-loop .thread {
      display: grid;
      gap: 10px;
      align-content: start;
    }

    .v2-loop .composer {
      position: relative;
      display: flex;
      align-items: center;
      gap: 12px;
      margin-top: auto;
      min-height: 42px;
      border: 1px solid var(--line);
      border-radius: 4px;
      padding: 8px 12px;
      background: var(--panel);
    }

    .v2-loop .draft {
      color: var(--text);
      font-size: 15px;
      line-height: 1.35;
      opacity: 0;
    }

    .v2-loop .draft-send {
      animation: draft-send 18s linear infinite;
    }

    .v2-loop .draft-reply {
      position: absolute;
      left: 12px;
      right: 32px;
      animation: draft-reply 18s linear infinite;
    }

    .v2-loop .send-mark {
      width: 8px;
      height: 8px;
      margin-left: auto;
      border-top: 1.5px solid var(--faint);
      border-right: 1.5px solid var(--faint);
      transform: rotate(45deg);
      flex: none;
    }

    .v2-loop .loop-who {
      margin: 0;
      color: var(--faint);
      font-size: 13px;
    }

    .v2-loop .bubble {
      margin: 0;
      max-width: 92%;
      border: 1px solid var(--line);
      border-radius: 4px;
      padding: 10px 12px;
      color: var(--text);
      font-size: 15px;
      line-height: 1.35;
      opacity: 0;
    }

    .v2-loop .human {
      justify-self: end;
      background: var(--panel);
    }

    .v2-loop .fleet {
      justify-self: start;
      background: rgba(245, 212, 107, 0.08);
      border-color: rgba(245, 212, 107, 0.35);
    }

    .v2-loop .fleet b {
      display: block;
      margin-bottom: 4px;
      color: var(--warn);
      font-size: 12px;
      font-weight: 760;
      letter-spacing: 0.06em;
      text-transform: uppercase;
    }

    .v2-loop .b-send {
      animation: hold-send 18s linear infinite;
    }
    .v2-loop .b-attn {
      animation: hold-attn 18s linear infinite;
    }
    .v2-loop .b-reply {
      animation: hold-reply 18s linear infinite;
    }
    .v2-loop .b-review {
      animation: hold-review 18s linear infinite;
    }

    .v2-loop .loop-board {
      height: 400px;
      padding: 14px;
    }

    .v2-loop .dock {
      display: none;
      margin: 0;
      gap: 4px;
      border: 1px solid rgba(156, 231, 178, 0.4);
      border-radius: 4px;
      padding: 10px 12px;
      background: rgba(156, 231, 178, 0.08);
    }

    .v2-loop .dock i,
    .v2-loop .face i {
      font-style: normal;
      font-size: 12px;
      font-weight: 760;
      letter-spacing: 0.06em;
      text-transform: uppercase;
    }

    .v2-loop .dock i {
      color: var(--good);
    }

    .v2-loop .dock span,
    .v2-loop .face span {
      color: var(--text);
      font-size: 15px;
      line-height: 1.35;
    }

    .v2-loop .shuttle,
    .v2-loop .green {
      position: absolute;
      z-index: 4;
      height: 92px;
      margin: 0;
      overflow: hidden;
      pointer-events: none;
    }

    .v2-loop .shuttle {
      animation: shuttle-move 18s linear infinite;
    }

    .v2-loop .green {
      animation: green-move 18s linear infinite;
    }

    .v2-loop .face {
      position: absolute;
      inset: 0;
      display: grid;
      align-content: center;
      gap: 4px;
      margin: 0;
      border: 1px solid var(--line);
      border-radius: 4px;
      padding: 8px 12px;
      background: var(--panel);
      color: var(--text);
      font-size: 15px;
      line-height: 1.35;
      box-shadow: 0 10px 24px rgba(0, 0, 0, 0.28);
    }

    .v2-loop .as-msg {
      animation: face-msg 18s linear infinite;
    }

    .v2-loop .as-task {
      animation: face-task 18s linear infinite;
    }

    .v2-loop .as-task i {
      color: var(--faint);
    }

    .v2-loop .as-warn {
      border-color: rgba(245, 212, 107, 0.7);
      background: rgba(245, 212, 107, 0.12);
      animation: face-warn 18s linear infinite;
    }

    .v2-loop .as-warn i {
      color: var(--warn);
    }

    .v2-loop .as-go {
      border-color: rgba(156, 231, 178, 0.55);
      background: rgba(156, 231, 178, 0.1);
      animation: face-go 18s linear infinite;
    }

    .v2-loop .as-go i,
    .v2-loop .as-review i {
      color: var(--good);
    }

    .v2-loop .as-review {
      border-color: rgba(156, 231, 178, 0.55);
      background: rgba(156, 231, 178, 0.1);
      animation: face-review 18s linear infinite;
    }

    @keyframes draft-send {
      0%,
      7% {
        opacity: 1;
      }
      11%,
      100% {
        opacity: 0;
      }
    }

    @keyframes draft-reply {
      0%,
      58% {
        opacity: 0;
      }
      62%,
      66% {
        opacity: 1;
      }
      70%,
      100% {
        opacity: 0;
      }
    }

    @keyframes hold-send {
      0%,
      7% {
        opacity: 0;
      }
      11%,
      92% {
        opacity: 1;
      }
      100% {
        opacity: 0;
      }
    }

    @keyframes hold-attn {
      0%,
      56% {
        opacity: 0;
      }
      60%,
      92% {
        opacity: 1;
      }
      100% {
        opacity: 0;
      }
    }

    @keyframes hold-reply {
      0%,
      62% {
        opacity: 0;
      }
      66%,
      92% {
        opacity: 1;
      }
      100% {
        opacity: 0;
      }
    }

    @keyframes hold-review {
      0%,
      80% {
        opacity: 0;
      }
      84%,
      92% {
        opacity: 1;
      }
      100% {
        opacity: 0;
      }
    }

    @keyframes shuttle-move {
      0%,
      13% {
        opacity: 0;
        left: 10%;
        top: 15%;
        width: 460px;
        height: 42px;
      }
      15% {
        opacity: 1;
        left: 10%;
        top: 15%;
        width: 460px;
        height: 42px;
      }
      21%,
      30% {
        opacity: 1;
        left: 66.3%;
        top: 9%;
        width: 250px;
        height: 92px;
      }
      42% {
        opacity: 1;
        left: 66.3%;
        top: 9%;
        width: 250px;
        height: 92px;
      }
      45% {
        opacity: 1;
        left: 66.3%;
        top: 27.5%;
        width: 250px;
        height: 92px;
      }
      48% {
        opacity: 1;
        left: 42%;
        top: 27.5%;
        width: 300px;
        height: 92px;
      }
      52% {
        opacity: 1;
        left: 1.6%;
        top: 27.5%;
        width: 340px;
        height: 92px;
      }
      56%,
      100% {
        opacity: 0;
        left: 1.6%;
        top: 27.5%;
        width: 340px;
        height: 92px;
      }
    }

    @keyframes green-move {
      0%,
      66% {
        opacity: 0;
        left: 38%;
        top: 56%;
        width: 250px;
      }
      68% {
        opacity: 1;
        left: 38%;
        top: 56%;
        width: 250px;
      }
      73% {
        opacity: 1;
        left: 66.3%;
        top: 56%;
        width: 250px;
      }
      80%,
      92% {
        opacity: 1;
        left: 66.3%;
        top: 9%;
        width: 250px;
      }
      100% {
        opacity: 0;
        left: 66.3%;
        top: 9%;
        width: 250px;
      }
    }

    @keyframes face-msg {
      0%,
      16% {
        opacity: 1;
      }
      21%,
      100% {
        opacity: 0;
      }
    }

    @keyframes face-task {
      0%,
      16% {
        opacity: 0;
      }
      21%,
      30% {
        opacity: 1;
      }
      36%,
      100% {
        opacity: 0;
      }
    }

    @keyframes face-warn {
      0%,
      30% {
        opacity: 0;
      }
      36%,
      52% {
        opacity: 1;
      }
      56%,
      100% {
        opacity: 0;
      }
    }

    @keyframes face-go {
      0%,
      76% {
        opacity: 1;
      }
      82%,
      100% {
        opacity: 0;
      }
    }

    @keyframes face-review {
      0%,
      76% {
        opacity: 0;
      }
      82%,
      100% {
        opacity: 1;
      }
    }

    @media (prefers-reduced-motion: reduce) {
      .v2-loop .shuttle,
      .v2-loop .green {
        display: none !important;
      }

      .v2-loop .bubble,
      .v2-loop .dock {
        animation: none !important;
        opacity: 1 !important;
      }

      .v2-loop .dock {
        display: grid !important;
      }
    }

    @media (max-width: 720px) {
      .v2-loop .loop-scene {
        grid-template-columns: 1fr;
      }

      .v2-loop .shuttle,
      .v2-loop .green {
        display: none;
      }

      .v2-loop .loop-window,
      .v2-loop .loop-board {
        height: auto;
        min-height: 0;
      }

      .v2-loop .dock {
        display: grid;
        animation: hold-review 18s linear infinite;
      }
    }
  }
</style>
