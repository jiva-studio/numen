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
    as?: 'div' | 'header'
    /** A row standing in a list, or the strip at the head of a block. */
    tone?: 'field' | 'header'
  }>(),
  { as: 'div', tone: 'field' },
)
</script>

<template>
  <component :is="as" class="card-row flex items-center" :data-tone="tone"><slot /></component>
</template>

<style scoped>
.card-row {
  --card-row-gap: 0.25rem;
  /* The air a row keeps at its ends: the same before the first thing in it as
     after the last, and enough for a ring to be drawn inside the row. */
  --card-row-pad-block: 0.125rem;
  --card-row-pad-inline: 0.375rem;

  gap: var(--card-row-gap);
  padding: var(--card-row-pad-block) var(--card-row-pad-inline);
  min-inline-size: 0;
}

/* A row in a list is a box: the line goes all the way round it. */
.card-row[data-tone='field'] {
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}

/* A strip at the head of a block is ruled off from what it heads. */
.card-row[data-tone='header'] {
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  background: var(--numen-code-bg);
}
</style>
