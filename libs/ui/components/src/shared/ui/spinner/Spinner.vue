<script setup lang="ts">
/**
 * A rotating ring beside something still loading: more of it is on its way. It
 * does not indicate how much is left.
 *
 * It is decorative, and the component that renders it marks the region busy.
 */
</script>

<template>
  <span class="spinner numen" aria-hidden="true" />
</template>

<style scoped>
.spinner {
  --cycle: var(--numen-wait-spin);

  display: inline-block;
  flex: none;
  inline-size: var(--spinner-size);
  block-size: var(--spinner-size);
  border: var(--spinner-thickness) solid currentColor;
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

/* The mark is static and is displayed beside the name; the band is
   filling, and no motion indicates it. */
@media (prefers-reduced-motion: reduce) {
  .spinner {
    opacity: 0.4;
    animation: none;
  }
}
</style>
