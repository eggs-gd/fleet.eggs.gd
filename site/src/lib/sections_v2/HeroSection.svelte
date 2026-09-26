<script lang="ts">
  import DownloadButton from '$lib/download/DownloadButton.svelte';
  import {
    Section,
    SectionFoot,
    SectionText,
    SectionTitle,
    Stack
  } from '$lib/layout';
</script>

<Section class="v2-hero" id="product" aria-labelledby="hero-title">
  {#snippet band()}
    <SectionTitle id="hero-title" level={1} size="l">
      I wanted one manager 
      for coding-agent work.
    </SectionTitle>
  {/snippet}

  <Stack class="hero-copy" gap="s">
    <SectionText size="l">That is pretty much how Fleet started.</SectionText>
    <SectionText size="m">Remote sessions let me use coding agents from my phone.</SectionText>
    <SectionText size="m">
      It worked — until I had Codex in one project,<br/>
      Claude in another, Cursor somewhere else,<br/>
      and my own memory acting as the orchestrator.
    </SectionText>
    <SectionText size="m">I didn't want remote access to a bunch of coding agents.</SectionText>
    <SectionText size="xl"><strong>I wanted one manager.</strong></SectionText>

    <SectionText size="l">
      The manager is an agent you already use.
      Fleet gave you that role: rough intent in, executable work out, real agents underneath.
    </SectionText>

    <div class="v2-manager">
      <div class="manager-scene" aria-hidden="true">
        <div class="scene-pile"></div>

        <div class="scene-chat">
          <span class="scene-label">You</span>
          <div class="chat-window">
            <div class="chat-bubble"></div>
            <div class="chat-bubble short"></div>
            <div class="chat-input"></div>
          </div>
        </div>

        <div class="scene-manager">
          <span class="scene-label">Manager</span>
          <span class="decide" aria-hidden="true"></span>
        </div>

        <div class="scene-backlog">
          <span class="scene-label">Backlog</span>
        </div>

        <div class="traveler queue-item fly-queue-1"></div>
        <div class="traveler queue-item fly-queue-2"></div>
        <div class="traveler queue-item fly-queue-3"></div>
        <div class="traveler queue-item fly-queue-4"></div>

        <div class="scene-bots">
          <div class="bot-badge">
            <span class="bot-icon bot-gemini" aria-hidden="true">
              <svg viewBox="0 0 24 24"
                ><path
                  d="M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81"
                /></svg
              >
            </span>
            <span class="bot-name">gemini</span>
            <i class="bot-status status-1"></i>
          </div>
          <div class="bot-badge">
            <span class="bot-icon bot-claude" aria-hidden="true">
              <svg viewBox="0 0 24 24"
                ><path
                  d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"
                /></svg
              >
            </span>
            <span class="bot-name">claude</span>
            <i class="bot-status status-2"></i>
          </div>
          <div class="bot-badge">
            <span class="bot-icon bot-codex" aria-hidden="true">X</span>
            <span class="bot-name">codex</span>
            <i class="bot-status status-3"></i>
          </div>
          <div class="bot-badge">
            <span class="bot-icon bot-cursor" aria-hidden="true">
              <svg viewBox="0 0 24 24"
                ><path
                  d="M11.503.131 1.891 5.678a.84.84 0 0 0-.42.726v11.188c0 .3.162.575.42.724l9.609 5.55a1 1 0 0 0 .998 0l9.61-5.55a.84.84 0 0 0 .42-.724V6.404a.84.84 0 0 0-.42-.726L12.497.131a1.01 1.01 0 0 0-.996 0M2.657 6.338h18.55c.263 0 .43.287.297.515L12.23 22.918c-.062.107-.229.064-.229-.06V12.335a.59.59 0 0 0-.295-.51l-9.11-5.257c-.109-.063-.064-.23.061-.23"
                /></svg
              >
            </span>
            <span class="bot-name">cursor</span>
            <i class="bot-status status-4"></i>
          </div>
        </div>

        <div class="traveler junk fly-junk-1">
          <span class="chip-wave"><i></i><i></i><i></i><i></i><i></i></span>
          <span>0:14</span>
        </div>
        <div class="traveler junk img fly-junk-2">
          <span class="pixel-thumb thumb-a"><i></i><i></i><i></i><i></i><i></i><i></i></span>
        </div>
        <div class="traveler junk paste fly-junk-3">
          user: also the sidebar collapses<br />claude: got it, reproducing<br />claude: found it, fixing
          now
        </div>
        <div class="traveler junk fly-junk-4">icons broken on mobile</div>
        <div class="traveler junk img fly-junk-5">
          <span class="pixel-thumb thumb-b"><i></i><i></i><i></i><i></i><i></i><i></i></span>
        </div>

        <div class="traveler task fly-task-1">CORE-162</div>
        <div class="traveler task fly-task-2">CORE-163</div>
        <div class="traveler task fly-task-3">CORE-164</div>
        <div class="traveler task fly-task-4">CORE-165</div>

        <div class="traveler reject fly-reject">? which repo</div>
      </div>

      <SectionText size="m" class="scene-caption">
        A voice note, a screenshot, a half-formed sentence — the manager sorts it into the right project,
        shapes it into a scoped task, and queues it for a worker.<br/>
        If the project or the intent is unclear, it comes back with a question instead of a guess.
      </SectionText>

      <SectionFoot>
        <SectionText size="xl">Three sentences in. Usable work out.</SectionText>
      </SectionFoot>
    </div>

    <SectionFoot>
      <div class="cta-row">
        <DownloadButton />
        <span class="early-badge">Early access</span>
      </div>
    </SectionFoot>
  </Stack>
</Section>

<style>
  :global {
    .v2-hero {
      padding-top: 76px;
      padding-bottom: 76px;
      text-align: center;
    }

    .v2-hero .section-title {
      max-width: 920px;
      margin: 0 auto;
    }

    .v2-hero .cta-row {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      justify-content: center;
      gap: 14px;
      margin-top: 10px;
    }

    .v2-hero .hero-copy {
      width: min(100%, 920px);
      margin: 0 auto;
    }

    .v2-hero .v2-manager {
      margin-top: 12px;
    }

    .v2-manager .manager-scene {
      position: relative;
      height: 360px;
      border: 1px solid var(--line);
      border-radius: 6px;
      background: var(--panel);
      overflow: hidden;
    }

    .v2-manager .scene-label {
      position: absolute;
      top: -20px;
      left: 0;
      color: var(--faint);
      font-size: 11px;
      font-weight: 760;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }

    .v2-manager .scene-pile {
      position: absolute;
      left: 4%;
      top: 6%;
      width: 30%;
      height: 46%;
    }

    .v2-manager .scene-chat {
      position: absolute;
      left: 4%;
      top: 68%;
      width: 26%;
      height: 26%;
    }

    .v2-manager .chat-window {
      display: grid;
      align-content: end;
      gap: 6px;
      height: 100%;
      border: 1px solid var(--line-strong);
      border-radius: 4px;
      background: var(--panel-strong);
      padding: 8px;
    }

    .v2-manager .chat-bubble {
      justify-self: start;
      width: 70%;
      height: 8px;
      border-radius: 2px;
      background: rgba(218, 226, 237, 0.1);
    }

    .v2-manager .chat-bubble.short {
      width: 42%;
    }

    .v2-manager .chat-input {
      height: 14px;
      border: 1px solid var(--line-strong);
      border-radius: 3px;
      background: var(--panel);
    }

    .v2-manager .scene-manager {
      position: absolute;
      left: 40%;
      top: 42%;
      width: 16%;
      height: 20%;
      display: grid;
      place-items: center;
      border: 1px solid var(--line-strong);
      border-radius: 4px;
      background: var(--panel-strong);
    }

    .v2-manager .decide {
      position: absolute;
      left: 50%;
      top: 50%;
      z-index: 3;
      width: 44px;
      height: 44px;
      margin: -22px 0 0 -22px;
      border: 2px solid rgba(218, 226, 237, 0.16);
      border-top-color: rgba(218, 226, 237, 0.92);
      border-radius: 50%;
      animation: decide-spin 1.15s linear infinite;
      pointer-events: none;
    }

    @keyframes decide-spin {
      to {
        transform: rotate(360deg);
      }
    }

    .v2-manager .scene-backlog {
      position: absolute;
      left: 68%;
      top: 56%;
      width: 26%;
      height: 30%;
      border: 1px solid var(--line);
      border-radius: 4px;
      background: var(--panel-strong);
    }

    .v2-manager .scene-bots {
      position: absolute;
      left: 68%;
      top: 10%;
      width: 26%;
      display: flex;
      gap: 5px;
    }

    .v2-manager .bot-badge {
      position: relative;
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 5px;
      border: 1px solid var(--line-strong);
      border-radius: 4px;
      background: var(--panel-strong);
      padding: 7px 4px;
    }

    .v2-manager .bot-icon {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 12px;
      height: 12px;
      font-family: var(--font-mono);
      font-size: 10px;
      font-weight: 760;
    }

    .v2-manager .bot-icon svg {
      width: 100%;
      height: 100%;
      fill: currentColor;
    }

    .v2-manager .bot-icon.bot-gemini {
      color: #7a8fae;
    }
    .v2-manager .bot-icon.bot-claude {
      color: #c98a63;
    }
    .v2-manager .bot-icon.bot-codex {
      color: var(--faint);
    }
    .v2-manager .bot-icon.bot-cursor {
      color: #b7b7c2;
    }

    .v2-manager .bot-status {
      position: absolute;
      top: -3px;
      right: -3px;
      width: 7px;
      height: 7px;
      border: 1px solid var(--panel-strong);
      border-radius: 50%;
      background: var(--good);
    }

    .v2-manager .bot-name {
      color: var(--muted);
      font-family: var(--font-mono);
      font-size: 9px;
      text-align: center;
    }

    .v2-manager .status-1 {
      animation: bot-status-pulse 8s infinite;
    }
    .v2-manager .status-2 {
      animation: bot-status-pulse 8s infinite;
      animation-delay: -2s;
    }
    .v2-manager .status-3 {
      animation: bot-status-pulse 8s infinite;
      animation-delay: -4s;
    }
    .v2-manager .status-4 {
      animation: bot-status-pulse 8s infinite;
      animation-delay: -6s;
    }

    @keyframes bot-status-pulse {
      0%,
      79% {
        background: var(--good);
      }
      82%,
      94% {
        background: var(--warn);
      }
      97%,
      100% {
        background: var(--good);
      }
    }

    .v2-manager .traveler {
      position: absolute;
      transform: translate(-50%, -50%);
      border: 1px solid var(--line-strong);
      border-radius: 4px;
      background: var(--panel-strong);
      padding: 5px 8px;
      color: var(--muted);
      font-family: var(--font-mono);
      font-size: 10px;
      white-space: nowrap;
      opacity: 0;
    }

    .v2-manager .traveler.img {
      padding: 4px;
      background: var(--panel-strong);
    }

    .v2-manager .traveler.paste {
      white-space: normal;
      width: 168px;
      line-height: 1.5;
    }

    .v2-manager .traveler.task {
      color: #0d1a12;
      background: var(--good);
      border-color: var(--good);
      font-weight: 700;
    }

    .v2-manager .traveler.reject {
      animation-name: reject-path;
    }

    .v2-manager .traveler.queue-item {
      width: 72px;
      height: 8px;
      padding: 0;
      border-radius: 2px;
      background: rgba(156, 231, 178, 0.22);
      border-color: transparent;
    }

    .v2-manager .chip-wave {
      display: inline-flex;
      align-items: flex-end;
      gap: 1px;
      height: 9px;
      margin-right: 4px;
    }

    .v2-manager .chip-wave i {
      display: block;
      width: 2px;
      background: var(--faint);
      border-radius: 1px;
    }

    .v2-manager .chip-wave i:nth-child(1) {
      height: 4px;
    }
    .v2-manager .chip-wave i:nth-child(2) {
      height: 9px;
    }
    .v2-manager .chip-wave i:nth-child(3) {
      height: 6px;
    }
    .v2-manager .chip-wave i:nth-child(4) {
      height: 8px;
    }
    .v2-manager .chip-wave i:nth-child(5) {
      height: 3px;
    }

    .v2-manager .pixel-thumb {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      grid-template-rows: repeat(2, 1fr);
      width: 32px;
      height: 20px;
      border-radius: 2px;
      overflow: hidden;
    }

    .v2-manager .thumb-a i:nth-child(1) {
      background: #46647f;
    }
    .v2-manager .thumb-a i:nth-child(2) {
      background: #6f8a6b;
    }
    .v2-manager .thumb-a i:nth-child(3) {
      background: #8a7a5c;
    }
    .v2-manager .thumb-a i:nth-child(4) {
      background: #33424d;
    }
    .v2-manager .thumb-a i:nth-child(5) {
      background: #9c8468;
    }
    .v2-manager .thumb-a i:nth-child(6) {
      background: #55684f;
    }

    .v2-manager .thumb-b i:nth-child(1) {
      background: #8a5c46;
    }
    .v2-manager .thumb-b i:nth-child(2) {
      background: #a68a5c;
    }
    .v2-manager .thumb-b i:nth-child(3) {
      background: #6f5c4a;
    }
    .v2-manager .thumb-b i:nth-child(4) {
      background: #40332c;
    }
    .v2-manager .thumb-b i:nth-child(5) {
      background: #b5946a;
    }
    .v2-manager .thumb-b i:nth-child(6) {
      background: #7a6552;
    }

    .v2-manager .fly-junk-1 {
      animation: junk-path-1 4.6s infinite;
    }
    .v2-manager .fly-junk-2 {
      animation: junk-path-2 4.6s infinite;
      animation-delay: -0.9s;
    }
    .v2-manager .fly-junk-3 {
      animation: junk-path-3 4.6s infinite;
      animation-delay: -1.8s;
    }
    .v2-manager .fly-junk-4 {
      animation: junk-path-4 4.6s infinite;
      animation-delay: -2.7s;
    }
    .v2-manager .fly-junk-5 {
      animation: junk-path-5 4.6s infinite;
      animation-delay: -3.6s;
    }

    @keyframes junk-path-1 {
      0%,
      4% {
        opacity: 0;
        left: 17%;
        top: 90%;
      }
      10% {
        opacity: 1;
        left: 17%;
        top: 90%;
      }
      28%,
      70% {
        opacity: 1;
        left: 10%;
        top: 16%;
      }
      92%,
      100% {
        opacity: 0;
        left: 48%;
        top: 52%;
      }
    }

    @keyframes junk-path-2 {
      0%,
      4% {
        opacity: 0;
        left: 17%;
        top: 90%;
      }
      10% {
        opacity: 1;
        left: 17%;
        top: 90%;
      }
      28%,
      70% {
        opacity: 1;
        left: 24%;
        top: 12%;
      }
      92%,
      100% {
        opacity: 0;
        left: 48%;
        top: 52%;
      }
    }

    @keyframes junk-path-3 {
      0%,
      4% {
        opacity: 0;
        left: 17%;
        top: 90%;
      }
      10% {
        opacity: 1;
        left: 17%;
        top: 90%;
      }
      28%,
      70% {
        opacity: 1;
        left: 14%;
        top: 36%;
      }
      92%,
      100% {
        opacity: 0;
        left: 48%;
        top: 52%;
      }
    }

    @keyframes junk-path-4 {
      0%,
      4% {
        opacity: 0;
        left: 17%;
        top: 90%;
      }
      10% {
        opacity: 1;
        left: 17%;
        top: 90%;
      }
      28%,
      70% {
        opacity: 1;
        left: 27%;
        top: 40%;
      }
      92%,
      100% {
        opacity: 0;
        left: 48%;
        top: 52%;
      }
    }

    @keyframes junk-path-5 {
      0%,
      4% {
        opacity: 0;
        left: 17%;
        top: 90%;
      }
      10% {
        opacity: 1;
        left: 17%;
        top: 90%;
      }
      28%,
      70% {
        opacity: 1;
        left: 18%;
        top: 26%;
      }
      92%,
      100% {
        opacity: 0;
        left: 48%;
        top: 52%;
      }
    }

    .v2-manager .fly-task-1 {
      animation: task-path 20s infinite;
    }
    .v2-manager .fly-task-2 {
      animation: task-path 20s infinite;
      animation-delay: -5s;
    }
    .v2-manager .fly-task-3 {
      animation: task-path 20s infinite;
      animation-delay: -10s;
    }
    .v2-manager .fly-task-4 {
      animation: task-path 20s infinite;
      animation-delay: -15s;
    }

    @keyframes task-path {
      0%,
      1.5% {
        opacity: 0;
        left: 48%;
        top: 52%;
      }
      4% {
        opacity: 1;
        left: 48%;
        top: 52%;
      }
      11.25%,
      14.5% {
        opacity: 1;
        left: 81%;
        top: 79%;
      }
      17%,
      100% {
        opacity: 0;
        left: 81%;
        top: 79%;
      }
    }

    .v2-manager .fly-queue-1 {
      animation: queue-path-1 8s infinite;
    }
    .v2-manager .fly-queue-2 {
      animation: queue-path-2 8s infinite;
      animation-delay: -2s;
    }
    .v2-manager .fly-queue-3 {
      animation: queue-path-3 8s infinite;
      animation-delay: -4s;
    }
    .v2-manager .fly-queue-4 {
      animation: queue-path-4 8s infinite;
      animation-delay: -6s;
    }

    @keyframes queue-path-1 {
      0%,
      3% {
        opacity: 0;
        left: 81%;
        top: 79%;
      }
      6%,
      20% {
        opacity: 1;
        left: 81%;
        top: 79%;
      }
      23%,
      36% {
        opacity: 1;
        left: 81%;
        top: 72%;
      }
      39%,
      52% {
        opacity: 1;
        left: 81%;
        top: 65%;
      }
      55%,
      68% {
        opacity: 1;
        left: 81%;
        top: 58%;
      }
      82%,
      94% {
        opacity: 1;
        left: 71.07%;
        top: 14.11%;
      }
      100% {
        opacity: 0;
        left: 71.07%;
        top: 14.11%;
      }
    }

    @keyframes queue-path-2 {
      0%,
      3% {
        opacity: 0;
        left: 81%;
        top: 79%;
      }
      6%,
      20% {
        opacity: 1;
        left: 81%;
        top: 79%;
      }
      23%,
      36% {
        opacity: 1;
        left: 81%;
        top: 72%;
      }
      39%,
      52% {
        opacity: 1;
        left: 81%;
        top: 65%;
      }
      55%,
      68% {
        opacity: 1;
        left: 81%;
        top: 58%;
      }
      82%,
      94% {
        opacity: 1;
        left: 77.82%;
        top: 14.11%;
      }
      100% {
        opacity: 0;
        left: 77.82%;
        top: 14.11%;
      }
    }

    @keyframes queue-path-3 {
      0%,
      3% {
        opacity: 0;
        left: 81%;
        top: 79%;
      }
      6%,
      20% {
        opacity: 1;
        left: 81%;
        top: 79%;
      }
      23%,
      36% {
        opacity: 1;
        left: 81%;
        top: 72%;
      }
      39%,
      52% {
        opacity: 1;
        left: 81%;
        top: 65%;
      }
      55%,
      68% {
        opacity: 1;
        left: 81%;
        top: 58%;
      }
      82%,
      94% {
        opacity: 1;
        left: 84.31%;
        top: 14.11%;
      }
      100% {
        opacity: 0;
        left: 84.31%;
        top: 14.11%;
      }
    }

    @keyframes queue-path-4 {
      0%,
      3% {
        opacity: 0;
        left: 81%;
        top: 79%;
      }
      6%,
      20% {
        opacity: 1;
        left: 81%;
        top: 79%;
      }
      23%,
      36% {
        opacity: 1;
        left: 81%;
        top: 72%;
      }
      39%,
      52% {
        opacity: 1;
        left: 81%;
        top: 65%;
      }
      55%,
      68% {
        opacity: 1;
        left: 81%;
        top: 58%;
      }
      82%,
      94% {
        opacity: 1;
        left: 90.79%;
        top: 14.11%;
      }
      100% {
        opacity: 0;
        left: 90.79%;
        top: 14.11%;
      }
    }

    .v2-manager .fly-reject {
      animation-duration: 7s;
      animation-iteration-count: infinite;
    }

    @keyframes reject-path {
      0%,
      5% {
        opacity: 0;
        left: 48%;
        top: 52%;
        border-color: var(--warn);
        color: var(--warn);
        background: rgba(245, 212, 107, 0.08);
      }
      14%,
      36% {
        opacity: 1;
        left: 48%;
        top: 52%;
        border-color: var(--warn);
        color: var(--warn);
        background: rgba(245, 212, 107, 0.08);
      }
      44%,
      58% {
        opacity: 1;
        left: 17%;
        top: 90%;
        border-color: var(--warn);
        color: var(--warn);
        background: rgba(245, 212, 107, 0.08);
      }
      72% {
        opacity: 1;
        left: 17%;
        top: 90%;
        border-color: var(--line-strong);
        color: var(--muted);
        background: var(--panel-strong);
      }
      88%,
      96% {
        opacity: 1;
        left: 28%;
        top: 20%;
        border-color: var(--line-strong);
        color: var(--muted);
        background: var(--panel-strong);
      }
      100% {
        opacity: 0;
        left: 28%;
        top: 20%;
        border-color: var(--line-strong);
        color: var(--muted);
        background: var(--panel-strong);
      }
    }

    @media (prefers-reduced-motion: reduce) {
      .v2-manager .traveler,
      .v2-manager .decide {
        animation: none !important;
      }

      .v2-manager .traveler {
        opacity: 0 !important;
      }
    }

    @media (max-width: 720px) {
      .v2-manager .manager-scene {
        height: 375px;
      }

      .v2-manager .scene-pile {
        left: 4%;
        top: 5.12%;
        width: 42%;
        height: 33.28%;
      }

      .v2-manager .scene-chat {
        left: 4%;
        top: 38%;
        width: 42%;
        height: 20%;
      }

      .v2-manager .scene-manager {
        left: 54%;
        top: 38%;
        width: 42%;
        height: 20%;
      }

      .v2-manager .scene-backlog {
        left: 8%;
        top: 66.46%;
        width: 40%;
        height: 28.16%;
      }

      .v2-manager .scene-bots {
        left: 54%;
        top: 66.46%;
        width: 42%;
        flex-wrap: wrap;
      }

      .v2-manager .bot-badge {
        flex: 1 1 40%;
      }

      .v2-manager .traveler.queue-item {
        width: 56px;
      }

      .v2-manager .traveler.paste {
        width: 130px;
        font-size: 9px;
      }

      @keyframes junk-path-1 {
        0%,
        4% {
          opacity: 0;
          left: 25%;
          top: 54%;
        }
        10% {
          opacity: 1;
          left: 25%;
          top: 54%;
        }
        28%,
        70% {
          opacity: 1;
          left: 12%;
          top: 12.8%;
        }
        92%,
        100% {
          opacity: 0;
          left: 75%;
          top: 48%;
        }
      }

      @keyframes junk-path-2 {
        0%,
        4% {
          opacity: 0;
          left: 25%;
          top: 54%;
        }
        10% {
          opacity: 1;
          left: 25%;
          top: 54%;
        }
        28%,
        70% {
          opacity: 1;
          left: 30%;
          top: 10.24%;
        }
        92%,
        100% {
          opacity: 0;
          left: 75%;
          top: 48%;
        }
      }

      @keyframes junk-path-3 {
        0%,
        4% {
          opacity: 0;
          left: 25%;
          top: 54%;
        }
        10% {
          opacity: 1;
          left: 25%;
          top: 54%;
        }
        28%,
        70% {
          opacity: 1;
          left: 10%;
          top: 25.6%;
        }
        92%,
        100% {
          opacity: 0;
          left: 75%;
          top: 48%;
        }
      }

      @keyframes junk-path-4 {
        0%,
        4% {
          opacity: 0;
          left: 25%;
          top: 54%;
        }
        10% {
          opacity: 1;
          left: 25%;
          top: 54%;
        }
        28%,
        70% {
          opacity: 1;
          left: 38%;
          top: 23.04%;
        }
        92%,
        100% {
          opacity: 0;
          left: 75%;
          top: 48%;
        }
      }

      @keyframes junk-path-5 {
        0%,
        4% {
          opacity: 0;
          left: 25%;
          top: 54%;
        }
        10% {
          opacity: 1;
          left: 25%;
          top: 54%;
        }
        28%,
        70% {
          opacity: 1;
          left: 20%;
          top: 15.36%;
        }
        92%,
        100% {
          opacity: 0;
          left: 75%;
          top: 48%;
        }
      }

      @keyframes task-path {
        0%,
        1.5% {
          opacity: 0;
          left: 75%;
          top: 48%;
        }
        4% {
          opacity: 1;
          left: 75%;
          top: 48%;
        }
        11.25%,
        14.5% {
          opacity: 1;
          left: 28%;
          top: 92.05%;
        }
        17%,
        100% {
          opacity: 0;
          left: 28%;
          top: 92.05%;
        }
      }

      @keyframes queue-path-1 {
        0%,
        3% {
          opacity: 0;
          left: 28%;
          top: 92.05%;
        }
        6%,
        20% {
          opacity: 1;
          left: 28%;
          top: 92.05%;
        }
        23%,
        36% {
          opacity: 1;
          left: 28%;
          top: 84.37%;
        }
        39%,
        52% {
          opacity: 1;
          left: 28%;
          top: 76.69%;
        }
        55%,
        68% {
          opacity: 1;
          left: 28%;
          top: 69.01%;
        }
        82%,
        94% {
          opacity: 1;
          left: 64.05%;
          top: 70.06%;
        }
        100% {
          opacity: 0;
          left: 64.05%;
          top: 70.06%;
        }
      }

      @keyframes queue-path-2 {
        0%,
        3% {
          opacity: 0;
          left: 28%;
          top: 92.05%;
        }
        6%,
        20% {
          opacity: 1;
          left: 28%;
          top: 92.05%;
        }
        23%,
        36% {
          opacity: 1;
          left: 28%;
          top: 84.37%;
        }
        39%,
        52% {
          opacity: 1;
          left: 28%;
          top: 76.69%;
        }
        55%,
        68% {
          opacity: 1;
          left: 28%;
          top: 69.01%;
        }
        82%,
        94% {
          opacity: 1;
          left: 85.65%;
          top: 70.06%;
        }
        100% {
          opacity: 0;
          left: 85.65%;
          top: 70.06%;
        }
      }

      @keyframes queue-path-3 {
        0%,
        3% {
          opacity: 0;
          left: 28%;
          top: 92.05%;
        }
        6%,
        20% {
          opacity: 1;
          left: 28%;
          top: 92.05%;
        }
        23%,
        36% {
          opacity: 1;
          left: 28%;
          top: 84.37%;
        }
        39%,
        52% {
          opacity: 1;
          left: 28%;
          top: 76.69%;
        }
        55%,
        68% {
          opacity: 1;
          left: 28%;
          top: 69.01%;
        }
        82%,
        94% {
          opacity: 1;
          left: 64.05%;
          top: 78.86%;
        }
        100% {
          opacity: 0;
          left: 64.05%;
          top: 78.86%;
        }
      }

      @keyframes queue-path-4 {
        0%,
        3% {
          opacity: 0;
          left: 28%;
          top: 92.05%;
        }
        6%,
        20% {
          opacity: 1;
          left: 28%;
          top: 92.05%;
        }
        23%,
        36% {
          opacity: 1;
          left: 28%;
          top: 84.37%;
        }
        39%,
        52% {
          opacity: 1;
          left: 28%;
          top: 76.69%;
        }
        55%,
        68% {
          opacity: 1;
          left: 28%;
          top: 69.01%;
        }
        82%,
        94% {
          opacity: 1;
          left: 85.65%;
          top: 78.86%;
        }
        100% {
          opacity: 0;
          left: 85.65%;
          top: 78.86%;
        }
      }

      @keyframes reject-path {
        0%,
        5% {
          opacity: 0;
          left: 75%;
          top: 48%;
          border-color: var(--warn);
          color: var(--warn);
          background: rgba(245, 212, 107, 0.08);
        }
        14%,
        36% {
          opacity: 1;
          left: 75%;
          top: 48%;
          border-color: var(--warn);
          color: var(--warn);
          background: rgba(245, 212, 107, 0.08);
        }
        44%,
        58% {
          opacity: 1;
          left: 25%;
          top: 54%;
          border-color: var(--warn);
          color: var(--warn);
          background: rgba(245, 212, 107, 0.08);
        }
        72% {
          opacity: 1;
          left: 25%;
          top: 54%;
          border-color: var(--line-strong);
          color: var(--muted);
          background: var(--panel-strong);
        }
        88%,
        96% {
          opacity: 1;
          left: 34%;
          top: 30.72%;
          border-color: var(--line-strong);
          color: var(--muted);
          background: var(--panel-strong);
        }
        100% {
          opacity: 0;
          left: 34%;
          top: 30.72%;
          border-color: var(--line-strong);
          color: var(--muted);
          background: var(--panel-strong);
        }
      }
    }
  }
</style>
