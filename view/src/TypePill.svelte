<script lang="ts">
  import { typeLabel, typePillClass } from './lib/taskTags';

  export let type = '';
</script>

<span class={typePillClass(type)}>{typeLabel(type)}</span>
