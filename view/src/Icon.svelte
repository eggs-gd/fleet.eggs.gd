<script>
  export let name = '';
  export let size = 16;
  export let filled = false;

  const paths = {
    star: 'M12 3.5l2.47 5.24 5.78.7-4.24 4.03 1.12 5.71L12 16.9l-5.13 2.28 1.12-5.71-4.24-4.03 5.78-.7L12 3.5z',
    home: 'M4 11.5L12 4l8 7.5M6 10v9h5v-5h2v5h5v-9',
    users: 'M8 12a3.5 3.5 0 100-7 3.5 3.5 0 000 7zm8 0a3 3 0 100-6 3 3 0 000 6zM2 20c0-3 2.7-5.5 6-5.5s6 2.5 6 5.5m2-5.2c2.4.3 4 2.2 4 5.2',
    gear: 'M7 10h3v-3l-3.5-3.5a6 6 0 018 8l6 6a2 2 0 01-3 3l-6-6a6 6 0 01-8-8l3.5 3.5',
    folder: 'M3 6.5a1 1 0 011-1h4.4l1.6 2H20a1 1 0 011 1V18a1 1 0 01-1 1H4a1 1 0 01-1-1V6.5z',
    plus: 'M12 5v14M5 12h14',
    external: 'M9 6h9v9M18 6L6 18',
    github:
      'M12 2C6.5 2 2 6.6 2 12.2c0 4.5 2.9 8.3 6.9 9.6.5.1.7-.2.7-.5v-1.9c-2.8.6-3.4-1.4-3.4-1.4-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 0 1.5 1 1.5 1 .9 1.6 2.4 1.1 3 .9.1-.7.4-1.1.6-1.4-2.2-.3-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.8 1a9.4 9.4 0 015 0c2-1.3 2.8-1 2.8-1 .5 1.4.2 2.4.1 2.7.6.7 1 1.6 1 2.7 0 3.9-2.3 4.7-4.6 5 .4.3.7 1 .7 2v3c0 .3.2.6.7.5 4-1.3 6.9-5.1 6.9-9.6C22 6.6 17.5 2 12 2z',
    check: 'M5 13l4 4L19 7',
    'check-circle': 'M12 21a9 9 0 100-18 9 9 0 000 18zM8 12.5l2.5 2.5L16 9',
    'checks-circle': [
      'M12 21a9 9 0 100-18 9 9 0 000 18z',
      'M6.6 12.6l2.1 2.1 4.6-5.2',
      'M10.2 12.6l2.1 2.1 4.6-5.2'
    ],
    'x-circle': 'M12 21a9 9 0 100-18 9 9 0 000 18zM9.5 9.5l5 5m0-5l-5 5',
    'help-circle':
      'M12 21a9 9 0 100-18 9 9 0 000 18zM9.5 9.3a2.5 2.5 0 114 2c-.7.5-1.5 1-1.5 2.2M12 16.5v.1',
    search: 'M11 4a7 7 0 100 14 7 7 0 000-14zM21 21l-4.3-4.3',
    mic: 'M12 3a3 3 0 013 3v6a3 3 0 01-6 0V6a3 3 0 013-3zM6 11a6 6 0 0012 0M12 19v2',
    send: 'M4 12h14M14 6l6 6-6 6',
    'chevron-left': 'M15 5l-7 7 7 7',
    'chevron-right': 'M9 5l7 7-7 7',
    dots: 'M6 12h.01M12 12h.01M18 12h.01',
    expand: 'M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7',
    collapse: 'M4 14h6v6M20 10h-6V4M14 10l7-7M10 14l-7 7',
    comment: 'M4 5h16v11H8l-4 4V5z',
    paperclip: 'M8 13.5l6.5-6.5a3 3 0 114.24 4.24L11 19a5 5 0 11-7-7l7.5-7.5',
    refresh: 'M4 4v5h5M20 20v-5h-5M4.5 9a8 8 0 0113.4-3.6L20 8M19.5 15a8 8 0 01-13.4 3.6L4 16',
    doc: 'M7 3h6l5 5v13H7z M13 3v5h5',
    bug: 'M9 8V6a3 3 0 016 0v2M6 10h12M6 14h12M8 10a4 4 0 008 0M5 9.5l2-1.8M19 9.5l-2-1.8M5 17.5l2-1.8M19 17.5l-2-1.8M12 10v8',
    flag: 'M6 3v18M6 4.5h11l-2 3.5 2 3.5H6',
    archive: 'M4 7h16M6 7V5h12v2M6 7v12h12V7M10 12h4',
    play: 'M8 6.5v11l9-5.5-9-5.5z',
    undo: 'M9 8H4V3M4 8c2.4-3.2 7-5 11-3.2S21 13 18.5 17',
    sun: 'M12 17a5 5 0 100-10 5 5 0 000 10zM12 2v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4l1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4m11.4-11.4l1.4-1.4',
    moon: 'M16.5 13.5A7 7 0 119.2 5.4 5.5 5.5 0 0016.5 13.5z',
    desktop: 'M4 5h16v11H4zM8 20h8M12 16v4'
  };

  $: glyph = paths[name];
  $: segments = Array.isArray(glyph) ? glyph : [glyph || ''];
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill={filled ? 'currentColor' : 'none'}
  stroke="currentColor"
  stroke-width="1.8"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
  focusable="false"
>
  {#each segments as d}
    <path {d} />
  {/each}
</svg>
