<script>
  import { typeLabel, typePillClass } from './lib/taskTags.js';

  export let type = '';
</script>

<span class={typePillClass(type)}>{typeLabel(type)}</span>
