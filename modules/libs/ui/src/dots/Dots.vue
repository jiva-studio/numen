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
  display: inline-flex;
  align-items: center;
  gap: var(--numen-dot-gap);
}

/* The colour is whatever they stand on, so the same dots read on a plain
   surface and inside a filled button. */
.dots__dot {
  inline-size: var(--numen-dot-size);
  block-size: var(--numen-dot-size);
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.45;
  animation: dots-rise var(--numen-dot-cycle) ease-in-out infinite;
}

.dots__dot:nth-child(2) {
  animation-delay: calc(var(--numen-dot-cycle) / 3);
}

.dots__dot:nth-child(3) {
  animation-delay: calc(var(--numen-dot-cycle) / 3 * 2);
}

@keyframes dots-rise {
  0%,
  60%,
  100% {
    transform: translateY(0);
    opacity: 0.45;
  }
  30% {
    transform: translateY(calc(-1 * var(--numen-dot-rise)));
    opacity: 1;
  }
}
</style>
