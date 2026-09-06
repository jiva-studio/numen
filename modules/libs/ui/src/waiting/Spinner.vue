<script setup lang="ts">
/**
 * A ring turning beside something still filling: more of it is on its way. It
 * says nothing about how much is left.
 *
 * It is drawn for the eye alone, and whoever draws it marks the region busy.
 */
</script>

<template>
  <span class="spinner numen" aria-hidden="true" />
</template>

<style scoped>
/* As tall as one line of the text it stands beside, so it sits on that line's
   own middle. The colour is whatever it stands on, and `--spinner-size` and
   `--spinner-thickness` are whatever holds it. */
.spinner {
  --cycle: 800ms;

  display: inline-block;
  flex: none;
  inline-size: var(--spinner-size, 0.9em);
  block-size: var(--spinner-size, 0.9em);
  border: var(--spinner-thickness, 1.5px) solid currentColor;
  /* One arc left open is what makes the turning visible on a ring. */
  border-block-start-color: transparent;
  border-radius: 50%;
  opacity: 0.55;
  animation: spinner-turn var(--cycle) linear infinite;
}

@keyframes spinner-turn {
  to {
    transform: rotate(1turn);
  }
}

/* Still, it is a mark that is simply there beside the name: the band is
   filling, and nothing moves to say so. */
@media (prefers-reduced-motion: reduce) {
  .spinner {
    opacity: 0.4;
    animation: none;
  }
}
</style>
