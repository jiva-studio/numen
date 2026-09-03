<script setup lang="ts">
/**
 * Three dots rising in turn: something is being written.
 */
const DOTS = 3
</script>

<template>
  <span class="dots numen">
    <span v-for="dot in DOTS" :key="dot" class="dots__dot" />
  </span>
</template>

<style scoped>
.dots {
  /* How large a dot is drawn, and how far it rises. */
  --size: 0.25rem;
  --rise: 0.1875rem;
  /* Each dot rises a third of a cycle behind the one before it, so they read
     left to right. */
  --cycle: 1200ms;

  display: inline-flex;
  align-items: center;
  gap: var(--numen-dot-gap);
}

/* The colour is whatever they stand on, so the same dots read on a plain
   surface and inside a filled button. */
.dots__dot {
  inline-size: var(--size);
  block-size: var(--size);
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.45;
  animation: dots-rise var(--cycle) ease-in-out infinite;
}

.dots__dot:nth-child(2) {
  animation-delay: calc(var(--cycle) / 3);
}

.dots__dot:nth-child(3) {
  animation-delay: calc(var(--cycle) / 3 * 2);
}

@keyframes dots-rise {
  0%,
  60%,
  100% {
    transform: translateY(0);
    opacity: 0.45;
  }
  30% {
    transform: translateY(calc(-1 * var(--rise)));
    opacity: 1;
  }
}
</style>
