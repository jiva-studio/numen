<script setup lang="ts">
/**
 * A rule with something standing on it.
 *
 * What it holds stands in the middle, or at the start where the rule is given
 * `at="start"`.
 *
 * The line is drawn either side of what it holds and never under it: it is the
 * rule's own decoration, so nothing is announced around what stands there, and
 * what stands there keeps its own name. The line gives way as the room runs
 * out, and the rule stays one line however narrow it is drawn.
 */
withDefaults(defineProps<{ at?: 'middle' | 'start' }>(), { at: 'middle' })
</script>

<template>
  <div class="rule" role="presentation" :data-at="at">
    <span class="rule__held"><slot /></span>
  </div>
</template>

<style scoped>
.rule {
  /* The air the line keeps from what stands in the middle. */
  --rule-gap: 0.625rem;

  display: flex;
  align-items: center;
  gap: var(--rule-gap);
  inline-size: 100%;
  min-inline-size: 0;
}

/* Each side takes what is left of the width, down to nothing. */
.rule::before,
.rule::after {
  content: '';
  flex: 1 1 0;
  min-inline-size: 0;
  block-size: 0;
  border-block-start: var(--numen-stroke) dashed var(--numen-node-border);
}

/* A rule that leads with what it holds keeps only a stub of line before it. */
.rule[data-at='start']::before {
  flex: 0 0 var(--rule-gap);
}

/* What stands on the rule is drawn at its own width while there is room for
   it, and is what the width goes to first. */
.rule__held {
  display: flex;
  align-items: center;
  flex: 0 1 auto;
  min-inline-size: 0;
  overflow: hidden;
}
</style>
