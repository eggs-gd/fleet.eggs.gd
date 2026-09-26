<script lang="ts">
  import {
    Section,
    SectionFoot,
    SectionText,
    SectionTitle,
    Stack
  } from '$lib/layout';
</script>

<Section class="v2-orchestrator" id="orchestrator" aria-labelledby="orchestrator-title">
  {#snippet band()}
    <SectionTitle id="orchestrator-title" size="m">
      Parallel across projects. Serialized where code can collide.
    </SectionTitle>
  {/snippet}

  <Stack gap="l">
    <div class="v2-ownership" id="ownership">
    <Stack gap="m">
    <SectionText size="m">
      Fleet deliberately keeps one active worker on one repository at a time.<br/>
      Different repositories can move in parallel.<br/>
      The same repository cannot be silently rewritten by two workers at once.
    </SectionText>
    <SectionText size="m">
      If an agent is running, waiting for human input,
      or needs review, the work is still owned.
    </SectionText>



    <div class="task-flow" aria-hidden="true">
      <div class="flow-panel">
        <div class="flow-head"><strong>perceptrail</strong></div>
        <div class="flow-queue flow-queue-a">
          <div class="task-chip chip-a1">
            <span class="task-tagcell"><em class="task-tag tag-bug">bug</em></span>
            <span class="task-main"><span class="task-id">#341</span><span class="task-title">Fix session timeout</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-gemini"><svg viewBox="0 0 24 24"><path d="M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
          <div class="task-chip chip-a2">
            <span class="task-tagcell"><em class="task-tag tag-feature">feat</em></span>
            <span class="task-main"><span class="task-id">#342</span><span class="task-title">Add repo picker</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-cursor"><svg viewBox="0 0 24 24"><path d="M11.503.131 1.891 5.678a.84.84 0 0 0-.42.726v11.188c0 .3.162.575.42.724l9.609 5.55a1 1 0 0 0 .998 0l9.61-5.55a.84.84 0 0 0 .42-.724V6.404a.84.84 0 0 0-.42-.726L12.497.131a1.01 1.01 0 0 0-.996 0M2.657 6.338h18.55c.263 0 .43.287.297.515L12.23 22.918c-.062.107-.229.064-.229-.06V12.335a.59.59 0 0 0-.295-.51l-9.11-5.257c-.109-.063-.064-.23.061-.23" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
        </div>
      </div>

      <div class="flow-panel">
        <div class="flow-head"><strong>fleet</strong></div>
        <div class="flow-queue flow-queue-b">
          <div class="task-chip chip-b1">
            <span class="task-tagcell"><em class="task-tag tag-chore">chore</em></span>
            <span class="task-main"><span class="task-id">#118</span><span class="task-title">Refactor task store</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-claude"><svg viewBox="0 0 24 24"><path d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
          <div class="task-chip chip-b2">
            <span class="task-tagcell"><em class="task-tag tag-feature">feat</em></span>
            <span class="task-main"><span class="task-id">#119</span><span class="task-title">Dark mode toggle</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-codex">X</span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
          <div class="task-chip chip-b3">
            <span class="task-tagcell"><em class="task-tag tag-bug">bug</em></span>
            <span class="task-main"><span class="task-id">#120</span><span class="task-title">Crash on empty repo</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-gemini"><svg viewBox="0 0 24 24"><path d="M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
        </div>
      </div>

      <div class="flow-panel">
        <div class="flow-head"><strong>journal</strong></div>
        <div class="flow-queue flow-queue-c">
          <div class="task-chip chip-c1">
            <span class="task-tagcell"><em class="task-tag tag-feature">feat</em></span>
            <span class="task-main"><span class="task-id">#77</span><span class="task-title">Voice note parsing</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-claude"><svg viewBox="0 0 24 24"><path d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
          <div class="task-chip chip-c2">
            <span class="task-tagcell"><em class="task-tag tag-bug">bug</em></span>
            <span class="task-main"><span class="task-id">#78</span><span class="task-title">Sidebar collapse bug</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-cursor"><svg viewBox="0 0 24 24"><path d="M11.503.131 1.891 5.678a.84.84 0 0 0-.42.726v11.188c0 .3.162.575.42.724l9.609 5.55a1 1 0 0 0 .998 0l9.61-5.55a.84.84 0 0 0 .42-.724V6.404a.84.84 0 0 0-.42-.726L12.497.131a1.01 1.01 0 0 0-.996 0M2.657 6.338h18.55c.263 0 .43.287.297.515L12.23 22.918c-.062.107-.229.064-.229-.06V12.335a.59.59 0 0 0-.295-.51l-9.11-5.257c-.109-.063-.064-.23.061-.23" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
          <div class="task-chip chip-c3">
            <span class="task-tagcell"><em class="task-tag tag-feature">feat</em></span>
            <span class="task-main"><span class="task-id">#79</span><span class="task-title">Export to markdown</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-gemini"><svg viewBox="0 0 24 24"><path d="M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
          <div class="task-chip chip-c4">
            <span class="task-tagcell"><em class="task-tag tag-bug">bug</em></span>
            <span class="task-main"><span class="task-id">#80</span><span class="task-title">Slow search index</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-codex">X</span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
          <div class="task-chip chip-c5">
            <span class="task-tagcell"><em class="task-tag tag-chore">chore</em></span>
            <span class="task-main"><span class="task-id">#81</span><span class="task-title">Clean up old drafts</span></span>
            <span class="task-status">
              <em class="status-chip st-running">Running</em>
              <em class="status-chip st-attention">Needs attention</em>
              <em class="status-chip st-review">Review</em>
              <em class="status-chip st-accepted">Accepted</em>
            </span>
            <span class="task-agent agent-claude"><svg viewBox="0 0 24 24"><path d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z" /></svg></span>
            <span class="task-progress"><span class="task-progress-fill"></span></span>
          </div>
        </div>
      </div>
    </div>

    <SectionFoot>
      <SectionText size="l">
        Blocked is not done.
        Needs attention is not done.
        A stopped chat is not accepted work.
      </SectionText>
      <SectionText size="xl">Done means reviewed and accepted.</SectionText>
    </SectionFoot>
    </Stack>
    </div>
  </Stack>
</Section>

<style>
  :global {
    .v2-orchestrator {
      text-align: center;
    }

    .v2-orchestrator .section-title {
      max-width: 920px;
      margin: 0 auto;
    }

    .v2-orchestrator .flow-panel {
      text-align: left;
    }

    .v2-orchestrator .layer-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px;
    }

    .v2-orchestrator .layer-grid article {
      display: grid;
      gap: 12px;
      border: 1px solid var(--line);
      border-radius: 6px;
      background: var(--panel);
      padding: 22px;
    }

    .v2-orchestrator .layer-grid span {
      color: var(--faint);
      font-size: 12px;
      font-weight: 780;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }

    .v2-orchestrator .layer-grid strong {
      color: #e6edf6;
      font-size: var(--type-text-l);
      font-weight: 620;
      line-height: 1.34;
    }

    .v2-orchestrator .task-flow {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      align-items: start;
      gap: 12px;
    }

    .v2-orchestrator .flow-panel {
      display: grid;
      gap: 10px;
      align-content: start;
      min-width: 0;
      height: 264px;
      box-sizing: border-box;
      border: 1px solid var(--line);
      border-radius: 6px;
      background: var(--panel);
      padding: 14px;
    }

    .v2-orchestrator .flow-head strong {
      color: var(--text);
      font-size: 13px;
      font-weight: 760;
    }

    .v2-orchestrator .flow-queue {
      position: relative;
      overflow: hidden;
    }

    .v2-orchestrator .flow-queue-a {
      height: 76px;
    }
    .v2-orchestrator .flow-queue-b {
      height: 114px;
    }
    .v2-orchestrator .flow-queue-c {
      height: 190px;
    }

    .v2-orchestrator .task-chip {
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      box-sizing: border-box;
      display: flex;
      align-items: center;
      gap: 5px;
      height: 30px;
      padding: 0 7px;
      border: 1px solid var(--line-strong);
      border-radius: 4px;
      background: var(--panel-strong);
      opacity: 0;
      overflow: hidden;
    }

    .v2-orchestrator .task-tagcell {
      flex: none;
      width: 34px;
      text-align: right;
    }

    .v2-orchestrator .task-main {
      display: flex;
      flex: 1;
      align-items: baseline;
      gap: 6px;
      min-width: 0;
      text-align: left;
    }

    .v2-orchestrator .task-id {
      flex: none;
      color: var(--faint);
      font-family: var(--font-mono);
      font-size: 10px;
      font-weight: 700;
    }

    .v2-orchestrator .task-title {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      color: var(--muted);
      font-size: 11px;
      white-space: nowrap;
      text-overflow: ellipsis;
    }

    .v2-orchestrator .task-status {
      position: relative;
      flex: none;
      width: 80px;
      height: 14px;
    }

    .v2-orchestrator .task-tag,
    .v2-orchestrator .status-chip {
      border-radius: 3px;
      padding: 2px 4px;
      font-style: normal;
      font-size: 8px;
      font-weight: 760;
      letter-spacing: 0.02em;
      text-transform: uppercase;
      white-space: nowrap;
    }

    .v2-orchestrator .task-tag {
      display: inline-block;
    }

    .v2-orchestrator .status-chip {
      position: absolute;
      top: 0;
      right: 0;
    }

    .v2-orchestrator .tag-bug {
      color: var(--blocked);
      background: rgba(255, 139, 139, 0.12);
    }

    .v2-orchestrator .tag-feature {
      color: var(--good);
      background: rgba(156, 231, 178, 0.12);
    }

    .v2-orchestrator .tag-chore {
      color: var(--faint);
      background: rgba(218, 226, 237, 0.08);
    }

    .v2-orchestrator .status-chip {
      opacity: 0;
    }

    .v2-orchestrator .st-running,
    .v2-orchestrator .st-accepted {
      color: var(--good);
      background: rgba(156, 231, 178, 0.12);
    }

    .v2-orchestrator .st-attention {
      color: var(--warn);
      background: rgba(245, 212, 107, 0.12);
    }

    .v2-orchestrator .st-review {
      color: var(--muted);
      background: rgba(218, 226, 237, 0.08);
    }

    .v2-orchestrator .task-agent {
      display: flex;
      flex: none;
      align-items: center;
      justify-content: center;
      width: 14px;
      height: 14px;
      font-family: var(--font-mono);
      font-size: 11px;
      font-weight: 760;
    }

    .v2-orchestrator .task-agent svg {
      width: 100%;
      height: 100%;
      fill: currentColor;
    }

    .v2-orchestrator .agent-gemini {
      color: #7a8fae;
    }
    .v2-orchestrator .agent-claude {
      color: #c98a63;
    }
    .v2-orchestrator .agent-codex {
      color: var(--faint);
    }
    .v2-orchestrator .agent-cursor {
      color: #b7b7c2;
    }

    .v2-orchestrator .task-progress {
      position: absolute;
      left: 1px;
      right: 1px;
      bottom: 1px;
      height: 2px;
      border-radius: 0 0 3px 3px;
      background: rgba(218, 226, 237, 0.08);
      overflow: hidden;
    }

    .v2-orchestrator .task-progress-fill {
      position: absolute;
      top: 0;
      left: 0;
      bottom: 0;
      width: 0%;
      background: var(--good);
    }

    .v2-orchestrator .chip-a1 {
      animation: chip-flow-a 12s infinite;
    }
    .v2-orchestrator .chip-a2 {
      animation: chip-flow-a 12s infinite;
      animation-delay: -6s;
    }

    @keyframes chip-flow-a {
      0%,
      2% {
        opacity: 0;
        transform: translateY(38px);
        border-color: var(--line-strong);
      }
      4%,
      44% {
        opacity: 0.6;
        transform: translateY(38px);
        border-color: var(--line-strong);
      }
      50%,
      97% {
        opacity: 1;
        transform: translateY(0);
        border-color: var(--good);
      }
      99%,
      100% {
        opacity: 0;
        transform: translateY(0);
        border-color: var(--good);
      }
    }


    /* perceptrail / chip-a1: breezes through clean */
    .v2-orchestrator .chip-a1 .st-running {
      animation: status-a1-running 12s infinite;
    }
    .v2-orchestrator .chip-a1 .st-review {
      animation: status-a1-review 12s infinite;
    }
    .v2-orchestrator .chip-a1 .st-accepted {
      animation: status-a1-accepted 12s infinite;
    }
    .v2-orchestrator .chip-a1 .task-progress-fill {
      animation: progress-a1 12s infinite;
    }

    @keyframes status-a1-running {
      0%,
      51% {
        opacity: 0;
      }
      53%,
      60% {
        opacity: 1;
      }
      62%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-a1-review {
      0%,
      64% {
        opacity: 0;
      }
      66%,
      83% {
        opacity: 1;
      }
      85%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-a1-accepted {
      0%,
      86% {
        opacity: 0;
      }
      88%,
      96% {
        opacity: 1;
      }
      98%,
      100% {
        opacity: 0;
      }
    }
    @keyframes progress-a1 {
      0%,
      49% {
        width: 0%;
        background: var(--good);
      }
      52% {
        width: 8%;
        background: var(--good);
      }
      62% {
        width: 45%;
        background: var(--good);
      }
      66% {
        background: var(--muted);
      }
      82% {
        width: 85%;
        background: var(--muted);
      }
      86%,
      100% {
        width: 100%;
        background: var(--good);
      }
    }

    /* perceptrail / chip-a2: gets stuck on needs-attention */
    .v2-orchestrator .chip-a2 .st-running {
      animation: status-a2-running 12s infinite;
      animation-delay: -6s;
    }
    .v2-orchestrator .chip-a2 .st-attention {
      animation: status-a2-attention 12s infinite;
      animation-delay: -6s;
    }
    .v2-orchestrator .chip-a2 .st-accepted {
      animation: status-a2-accepted 12s infinite;
      animation-delay: -6s;
    }
    .v2-orchestrator .chip-a2 .task-progress-fill {
      animation: progress-a2 12s infinite;
      animation-delay: -6s;
    }

    @keyframes status-a2-running {
      0%,
      51% {
        opacity: 0;
      }
      53%,
      58% {
        opacity: 1;
      }
      60%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-a2-attention {
      0%,
      60% {
        opacity: 0;
      }
      62%,
      88% {
        opacity: 1;
      }
      90%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-a2-accepted {
      0%,
      90% {
        opacity: 0;
      }
      92%,
      96% {
        opacity: 1;
      }
      98%,
      100% {
        opacity: 0;
      }
    }
    @keyframes progress-a2 {
      0%,
      51% {
        width: 0%;
        background: var(--good);
      }
      55% {
        width: 15%;
        background: var(--good);
      }
      60%,
      88% {
        width: 15%;
        background: var(--warn);
      }
      92%,
      100% {
        width: 100%;
        background: var(--good);
      }
    }

    .v2-orchestrator .chip-b1 {
      animation: chip-flow-b 15s infinite;
    }
    .v2-orchestrator .chip-b2 {
      animation: chip-flow-b 15s infinite;
      animation-delay: -5s;
    }
    .v2-orchestrator .chip-b3 {
      animation: chip-flow-b 15s infinite;
      animation-delay: -10s;
    }

    @keyframes chip-flow-b {
      0%,
      2% {
        opacity: 0;
        transform: translateY(76px);
        border-color: var(--line-strong);
      }
      4%,
      29% {
        opacity: 0.6;
        transform: translateY(76px);
        border-color: var(--line-strong);
      }
      33%,
      62% {
        opacity: 0.6;
        transform: translateY(38px);
        border-color: var(--line-strong);
      }
      67%,
      96% {
        opacity: 1;
        transform: translateY(0);
        border-color: var(--good);
      }
      99%,
      100% {
        opacity: 0;
        transform: translateY(0);
        border-color: var(--good);
      }
    }


    /* fleet: review takes a while */
    .v2-orchestrator .chip-b1 .st-running,
    .v2-orchestrator .chip-b2 .st-running,
    .v2-orchestrator .chip-b3 .st-running {
      animation: status-b-running 15s infinite;
    }
    .v2-orchestrator .chip-b1 .st-attention,
    .v2-orchestrator .chip-b2 .st-attention,
    .v2-orchestrator .chip-b3 .st-attention {
      animation: status-b-attention 15s infinite;
    }
    .v2-orchestrator .chip-b1 .st-review,
    .v2-orchestrator .chip-b2 .st-review,
    .v2-orchestrator .chip-b3 .st-review {
      animation: status-b-review 15s infinite;
    }
    .v2-orchestrator .chip-b1 .st-accepted,
    .v2-orchestrator .chip-b2 .st-accepted,
    .v2-orchestrator .chip-b3 .st-accepted {
      animation: status-b-accepted 15s infinite;
    }
    .v2-orchestrator .chip-b1 .task-progress-fill,
    .v2-orchestrator .chip-b2 .task-progress-fill,
    .v2-orchestrator .chip-b3 .task-progress-fill {
      animation: progress-b 15s infinite;
    }
    .v2-orchestrator .chip-b2 .st-running,
    .v2-orchestrator .chip-b2 .st-attention,
    .v2-orchestrator .chip-b2 .st-review,
    .v2-orchestrator .chip-b2 .st-accepted,
    .v2-orchestrator .chip-b2 .task-progress-fill {
      animation-delay: -5s;
    }
    .v2-orchestrator .chip-b3 .st-running,
    .v2-orchestrator .chip-b3 .st-attention,
    .v2-orchestrator .chip-b3 .st-review,
    .v2-orchestrator .chip-b3 .st-accepted,
    .v2-orchestrator .chip-b3 .task-progress-fill {
      animation-delay: -10s;
    }

    @keyframes status-b-running {
      0%,
      68% {
        opacity: 0;
      }
      70%,
      71% {
        opacity: 1;
      }
      73%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-b-attention {
      0%,
      73% {
        opacity: 0;
      }
      75%,
      76% {
        opacity: 1;
      }
      78%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-b-review {
      0%,
      78% {
        opacity: 0;
      }
      80%,
      90% {
        opacity: 1;
      }
      92%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-b-accepted {
      0%,
      92% {
        opacity: 0;
      }
      94%,
      95% {
        opacity: 1;
      }
      97%,
      100% {
        opacity: 0;
      }
    }
    @keyframes progress-b {
      0%,
      68% {
        width: 0%;
        background: var(--good);
      }
      71% {
        width: 10%;
        background: var(--good);
      }
      75% {
        background: var(--warn);
      }
      78% {
        width: 20%;
        background: var(--warn);
      }
      80% {
        background: var(--muted);
      }
      90% {
        width: 85%;
        background: var(--muted);
      }
      94%,
      100% {
        width: 100%;
        background: var(--good);
      }
    }

    .v2-orchestrator .chip-c1 {
      animation: chip-flow-c 14.4s infinite;
    }
    .v2-orchestrator .chip-c2 {
      animation: chip-flow-c 14.4s infinite;
      animation-delay: -2.88s;
    }
    .v2-orchestrator .chip-c3 {
      animation: chip-flow-c 14.4s infinite;
      animation-delay: -5.76s;
    }
    .v2-orchestrator .chip-c4 {
      animation: chip-flow-c 14.4s infinite;
      animation-delay: -8.64s;
    }
    .v2-orchestrator .chip-c5 {
      animation: chip-flow-c 14.4s infinite;
      animation-delay: -11.52s;
    }

    @keyframes chip-flow-c {
      0%,
      1% {
        opacity: 0;
        transform: translateY(152px);
        border-color: var(--line-strong);
      }
      3%,
      17% {
        opacity: 0.6;
        transform: translateY(152px);
        border-color: var(--line-strong);
      }
      20%,
      37% {
        opacity: 0.6;
        transform: translateY(114px);
        border-color: var(--line-strong);
      }
      40%,
      57% {
        opacity: 0.6;
        transform: translateY(76px);
        border-color: var(--line-strong);
      }
      60%,
      77% {
        opacity: 0.6;
        transform: translateY(38px);
        border-color: var(--line-strong);
      }
      80%,
      97% {
        opacity: 1;
        transform: translateY(0);
        border-color: var(--good);
      }
      99%,
      100% {
        opacity: 0;
        transform: translateY(0);
        border-color: var(--good);
      }
    }


    /* journal: flies through fast, no blocking */
    .v2-orchestrator .chip-c1 .st-running,
    .v2-orchestrator .chip-c2 .st-running,
    .v2-orchestrator .chip-c3 .st-running,
    .v2-orchestrator .chip-c4 .st-running,
    .v2-orchestrator .chip-c5 .st-running {
      animation: status-c-running 14.4s infinite;
    }
    .v2-orchestrator .chip-c1 .st-review,
    .v2-orchestrator .chip-c2 .st-review,
    .v2-orchestrator .chip-c3 .st-review,
    .v2-orchestrator .chip-c4 .st-review,
    .v2-orchestrator .chip-c5 .st-review {
      animation: status-c-review 14.4s infinite;
    }
    .v2-orchestrator .chip-c1 .st-accepted,
    .v2-orchestrator .chip-c2 .st-accepted,
    .v2-orchestrator .chip-c3 .st-accepted,
    .v2-orchestrator .chip-c4 .st-accepted,
    .v2-orchestrator .chip-c5 .st-accepted {
      animation: status-c-accepted 14.4s infinite;
    }
    .v2-orchestrator .chip-c1 .task-progress-fill,
    .v2-orchestrator .chip-c2 .task-progress-fill,
    .v2-orchestrator .chip-c3 .task-progress-fill,
    .v2-orchestrator .chip-c4 .task-progress-fill,
    .v2-orchestrator .chip-c5 .task-progress-fill {
      animation: progress-c 14.4s infinite;
    }
    .v2-orchestrator .chip-c2 .st-running,
    .v2-orchestrator .chip-c2 .st-review,
    .v2-orchestrator .chip-c2 .st-accepted,
    .v2-orchestrator .chip-c2 .task-progress-fill {
      animation-delay: -2.88s;
    }
    .v2-orchestrator .chip-c3 .st-running,
    .v2-orchestrator .chip-c3 .st-review,
    .v2-orchestrator .chip-c3 .st-accepted,
    .v2-orchestrator .chip-c3 .task-progress-fill {
      animation-delay: -5.76s;
    }
    .v2-orchestrator .chip-c4 .st-running,
    .v2-orchestrator .chip-c4 .st-review,
    .v2-orchestrator .chip-c4 .st-accepted,
    .v2-orchestrator .chip-c4 .task-progress-fill {
      animation-delay: -8.64s;
    }
    .v2-orchestrator .chip-c5 .st-running,
    .v2-orchestrator .chip-c5 .st-review,
    .v2-orchestrator .chip-c5 .st-accepted,
    .v2-orchestrator .chip-c5 .task-progress-fill {
      animation-delay: -11.52s;
    }

    @keyframes status-c-running {
      0%,
      81% {
        opacity: 0;
      }
      82%,
      86% {
        opacity: 1;
      }
      88%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-c-review {
      0%,
      88% {
        opacity: 0;
      }
      89%,
      92% {
        opacity: 1;
      }
      94%,
      100% {
        opacity: 0;
      }
    }
    @keyframes status-c-accepted {
      0%,
      94% {
        opacity: 0;
      }
      95%,
      96.5% {
        opacity: 1;
      }
      98%,
      100% {
        opacity: 0;
      }
    }
    @keyframes progress-c {
      0%,
      81% {
        width: 0%;
        background: var(--good);
      }
      84% {
        width: 30%;
        background: var(--good);
      }
      88% {
        background: var(--muted);
      }
      92% {
        width: 80%;
        background: var(--muted);
      }
      95%,
      100% {
        width: 100%;
        background: var(--good);
      }
    }

    @media (prefers-reduced-motion: reduce) {
      .v2-orchestrator .task-chip,
      .v2-orchestrator .status-chip,
      .v2-orchestrator .task-progress-fill {
        animation: none !important;
      }

      .v2-orchestrator .chip-a1,
      .v2-orchestrator .chip-b1,
      .v2-orchestrator .chip-c1 {
        opacity: 1;
        transform: translateY(0);
        border-color: var(--good);
      }
      .v2-orchestrator .chip-a1 .st-running,
      .v2-orchestrator .chip-b1 .st-running,
      .v2-orchestrator .chip-c1 .st-running {
        opacity: 1;
      }
      .v2-orchestrator .chip-a1 .task-progress-fill,
      .v2-orchestrator .chip-b1 .task-progress-fill,
      .v2-orchestrator .chip-c1 .task-progress-fill {
        width: 40%;
        background: var(--good);
      }

      .v2-orchestrator .chip-a2 {
        opacity: 0.6;
        transform: translateY(38px);
      }

      .v2-orchestrator .chip-b2 {
        opacity: 0.6;
        transform: translateY(38px);
      }
      .v2-orchestrator .chip-b3 {
        opacity: 0.6;
        transform: translateY(76px);
      }

      .v2-orchestrator .chip-c2 {
        opacity: 0.6;
        transform: translateY(38px);
      }
      .v2-orchestrator .chip-c3 {
        opacity: 0.6;
        transform: translateY(76px);
      }
      .v2-orchestrator .chip-c4 {
        opacity: 0.6;
        transform: translateY(114px);
      }
      .v2-orchestrator .chip-c5 {
        opacity: 0.6;
        transform: translateY(152px);
      }
    }

    @media (max-width: 820px) {
      .v2-orchestrator .layer-grid {
        grid-template-columns: 1fr;
      }
    }

    @media (max-width: 820px) {
      .v2-orchestrator .task-flow {
        grid-template-columns: 1fr;
      }
    }
  }
</style>
