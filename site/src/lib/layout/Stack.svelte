<script lang="ts">
	import type { Snippet } from 'svelte';
	import { spaceVar, type Size } from './size';

	let {
		class: className = '',
		gap = '28px',
		children
	}: {
		class?: string;
		gap?: Size | string;
		children: Snippet;
	} = $props();
</script>

<div class="stack {className}" style="--stack-gap: {spaceVar(gap)}">
	{@render children()}
</div>
