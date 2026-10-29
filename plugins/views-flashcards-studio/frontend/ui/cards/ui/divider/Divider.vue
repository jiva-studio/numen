<script setup lang="ts">
/**
 * A line across the full width with content on it, centered or at the start of
 * the line when `at="start"`.
 *
 * The line is the divider's own decoration: nothing is announced around the
 * content, and the content keeps its own accessible name. The line shortens as
 * the available width shrinks, and the divider stays one line however narrow it
 * is.
 */
withDefaults(defineProps<{ at?: 'middle' | 'start' }>(), { at: 'middle' })
</script>

<template>
  <div class="divider" role="presentation" :data-at="at">
    <span class="divider__pending"><slot /></span>
  </div>
</template>

<style scoped>
.divider {
  display: flex;
  align-items: center;
  /* The air the line keeps from what relies on it. */
  gap: var(--numen-box-air);
  inline-size: 100%;
  min-inline-size: 0;
}

/* Each side takes what is left of the width, down to nothing. */
.divider::before,
.divider::after {
  content: '';
  flex: 1 1 0;
  min-inline-size: 0;
  block-size: 0;
  border-block-start: var(--numen-stroke) dashed var(--numen-rule);
}

/* A divider that leads with what it holds keeps only a stub of line before it. */
.divider[data-at='start']::before {
  flex: 0 0 var(--numen-box-air);
}

/* The pending content is drawn at its own width while there is space for
   it, and is what the width goes to first. */
.divider__pending {
  display: flex;
  align-items: center;
  flex: 0 1 auto;
  min-inline-size: 0;
  overflow: hidden;
}
</style>
