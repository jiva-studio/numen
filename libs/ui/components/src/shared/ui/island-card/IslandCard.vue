<script setup lang="ts">
/** A titled group: the heading is above a card that holds only the rows, so a group of one row stays one row tall. The heading may carry an aside at its right. The card pads like a card of settings rows; compact draws the tighter pad of a list of draggable rows. */
defineProps<{ title: string; isCompact?: boolean }>()
</script>

<template>
  <section class="island-card numen">
    <header class="island-card__header">
      <h3 class="island-card__title">{{ title }}</h3>
      <div v-if="$slots.aside" class="island-card__aside"><slot name="aside" /></div>
    </header>
    <div class="island-card__body" :class="{ 'island-card__body--compact': isCompact }">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.island-card {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.island-card__header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
  padding-inline: 0;
}

.island-card__title {
  margin: 0;
  min-inline-size: 0;
  font-size: var(--numen-text-3);
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
  padding: var(--numen-card-inset) var(--numen-card-pad);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.island-card__body--compact {
  padding: 0.25rem 0.5rem;
}
</style>
