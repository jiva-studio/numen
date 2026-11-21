<script setup lang="ts">
/** A titled group: the heading sits above a card that holds only the rows, so a group of one row stays one row tall. The heading may carry an aside at its right. */
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
  --body-pad: 0 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.island-card__header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
  padding-inline: 0.25rem;
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
  overflow: hidden;
  box-sizing: border-box;
  padding: var(--body-pad);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}
</style>
