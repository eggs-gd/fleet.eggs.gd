<script lang="ts">
  import { projectAccordionKey } from './lib/dashboardState';
  import { projectColor, projectInitial } from './lib/projectTree';
  import Icon from './Icon.svelte';
  import SidebarBranch from './SidebarBranch.svelte';
  import type { Project } from './lib/types';

  interface TreeNode {
    project: Project;
    children: TreeNode[];
    taskCount: number;
  }

  export let node: TreeNode;
  export let depth = 1;
  export let selectedProjectId = '';
  export let accordionOpen: Record<string, boolean> = {};
  export let onActivate: (
    projectId: string,
    key: string,
    isOpen: boolean,
    hasChildren: boolean
  ) => void = () => {};

  $: key = projectAccordionKey(node.project);
  $: open = accordionOpen[key] ?? node.taskCount > 0;
  $: hasChildren = node.children.length > 0;
</script>

<li class="sidebar-branch" style="--tree-depth: {depth}">
  <button
    type="button"
    class="sidebar-project"
    class:active={selectedProjectId === node.project.id}
    aria-expanded={hasChildren ? open : undefined}
    on:click={() => onActivate(node.project.id, key, open, hasChildren)}
  >
    <span
      class="project-mark"
      class:project-mark--space={hasChildren}
      class:project-mark--project={!hasChildren}
      style="background: {projectColor(node.project.id)}"
      >{projectInitial(node.project.title, node.project.id)}</span
    >
    <span class="sidebar-project-title">{node.project.title || node.project.id}</span>
    <span class="sidebar-project-count">{node.taskCount}</span>
    {#if hasChildren}
      <span class="tree-toggle" class:is-open={open} aria-hidden="true">
        <Icon name="chevron-right" size={12} />
      </span>
    {/if}
  </button>
  {#if hasChildren && open}
    <ul>
      {#each node.children as child (child.project.id)}
        <SidebarBranch
          node={child}
          depth={depth + 1}
          {selectedProjectId}
          {accordionOpen}
          {onActivate}
        />
      {/each}
    </ul>
  {/if}
</li>

<style>
  .sidebar-branch {
    padding-left: calc(var(--tree-depth, 1) * var(--space-lg));
    min-width: 0;
  }
</style>
