<script setup lang="ts">
/**
 * A ring turning beside something still filling: more of it is on its way.
 *
 * It says nothing about how much is left. What is coming is counted somewhere
 * or it is not, and a length that guesses is worse than one making no claim.
 *
 * What is filling is what a screen reader is told about, so this is drawn for
 * the eye alone and whoever draws it marks the region busy.
 */
</script>

<template>
  <span class="waiting numen" aria-hidden="true" />
</template>

<style scoped>
/* As tall as one line of the text it stands beside, so it sits on that line's
   own middle. The colour is whatever it stands on. */
.waiting {
  --size: 0.9em;
  --thickness: 1.5px;
  --cycle: 800ms;

  display: inline-block;
  flex: none;
  inline-size: var(--size);
  block-size: var(--size);
  border: var(--thickness) solid currentColor;
  /* One arc left open is what makes the turning visible on a ring. */
  border-block-start-color: transparent;
  border-radius: 50%;
  opacity: 0.55;
  animation: waiting-turn var(--cycle) linear infinite;
}

@keyframes waiting-turn {
  to {
    transform: rotate(1turn);
  }
}

/* Still, it is a mark that is simply there beside the name: the band is
   filling, and nothing moves to say so. */
@media (prefers-reduced-motion: reduce) {
  .waiting {
    opacity: 0.4;
    animation: none;
  }
}
</style>
