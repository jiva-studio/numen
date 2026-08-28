<script setup lang="ts">
/**
 * One row, drawn as one thing.
 *
 * The line around it and the ground under it are the row's own, so a handle, a
 * box to type in and a way to remove it read as one block. Where the keyboard
 * lands inside, the row is what says so.
 */
withDefaults(
  defineProps<{
    /** What the row is drawn as. */
    as?: string
    /** A row standing in a list, or the strip at the head of a block. */
    tone?: 'field' | 'bar'
  }>(),
  { as: 'div', tone: 'field' },
)
</script>

<template>
  <component :is="as" class="slab flex items-center" :data-tone="tone"><slot /></component>
</template>

<style scoped>
.slab {
  --slab-gap: 0.25rem;
  /* The air a row keeps at its ends: the same before the first thing in it as
     after the last, and enough for a ring to be drawn inside the row. */
  --slab-pad-block: 0.125rem;
  --slab-pad-inline: 0.375rem;

  gap: var(--slab-gap);
  padding: var(--slab-pad-block) var(--slab-pad-inline);
  min-inline-size: 0;
}

/* A row in a list is a box: the line goes all the way round it. */
.slab[data-tone='field'] {
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}

/* A strip at the head of a block is ruled off from what it heads. */
.slab[data-tone='bar'] {
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
  background: var(--numen-code-bg);
}
</style>
