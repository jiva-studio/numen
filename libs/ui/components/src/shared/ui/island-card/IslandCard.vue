<script setup lang="ts">
/** A titled group: a card whose tinted header holds the title and may carry an aside at its right, with the rows below it. The body pads like a card of settings rows. */
defineProps<{ title: string }>()
</script>

<template>
  <section class="island-card numen">
    <header class="island-card__header">
      <h3 class="island-card__title">{{ title }}</h3>
      <div v-if="$slots.aside" class="island-card__aside"><slot name="aside" /></div>
    </header>
    <div class="island-card__body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.island-card {
  overflow: hidden;
  box-sizing: border-box;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.island-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  container: island-card-header / inline-size;
  padding: 0.375rem var(--numen-card-pad);
  background-color: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 6%);
}

.island-card__title {
  margin: 0;
  min-inline-size: 0;
  overflow-wrap: anywhere;
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.island-card__aside {
  display: flex;
  flex: none;
  align-items: center;
  gap: 0.5rem;
  font-size: var(--numen-text-2);
  color: var(--numen-hushed);
}

.island-card__body {
  box-sizing: border-box;
  padding: var(--numen-card-inset) var(--numen-card-pad);
}
</style>

<style>
/* The last action moves right so its box stands as far from the right edge as from the top. */
.island-card__aside:has(button, [role="button"]) {
  margin-inline-end: calc(0.375rem - var(--numen-card-pad));
}
</style>
