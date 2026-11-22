<script setup lang="ts">
/** A titled group: a card whose tinted header holds the title and may carry an aside at its right, with the rows below it. The body pads like a card of settings rows; compact draws the tighter pad of a list of draggable rows. */
defineProps<{ title: string; isCompact?: boolean }>()
</script>

<template>
  <section class="island-card numen" :class="{ 'island-card--compact': isCompact }">
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
  --island-pad-x: var(--numen-card-pad);
  --island-pad-y: var(--numen-card-inset);
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
  padding: 0.375rem var(--island-pad-x);
  background-color: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 6%);
}

.island-card--compact {
  --island-pad-x: 0.5rem;
  --island-pad-y: 0.25rem;
}

.island-card__title {
  margin: 0;
  min-inline-size: 0;
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.island-card__aside {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.island-card__body {
  box-sizing: border-box;
  padding: var(--island-pad-y) var(--island-pad-x);
}
</style>
