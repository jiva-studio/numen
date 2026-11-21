<script setup lang="ts">
/** A card of a titled header over a body, set off by a hairline: the header names it and may carry actions or a description, and the body holds a list of rows. */
defineProps<{ title: string }>()
</script>

<template>
  <section class="island-card numen">
    <header class="island-card__header">
      <h3 class="island-card__title">{{ title }}</h3>
      <div v-if="$slots.aside" class="island-card__aside"><slot name="aside" /></div>
    </header>
    <div class="island-card__body"><slot /></div>
  </section>
</template>

<style scoped>
.island-card {
  --header-pad: 0.25rem 0.75rem;
  --body-pad: 0 0.75rem;
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
  padding: var(--header-pad);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
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
  padding: var(--body-pad);
}
</style>
