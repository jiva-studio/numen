<script setup lang="ts">
/**
 * A row of metric tiles above or below the curve slider.
 */

// --- Props & Emits ---
defineProps<{
  control: string
  tiles: readonly { figure: string; name: string }[]
}>()

// --- State ---

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <div class="curve-slider__material" :data-control="control">
    <span v-for="one in tiles" :key="one.name" class="curve-slider__tile" data-control="tile">
      <span class="curve-slider__figure" data-control="figure">{{ one.figure }}</span>
      <span class="curve-slider__word" data-control="word">{{ one.name }}</span>
    </span>
  </div>
</template>

<style scoped>
.curve-slider__material {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: 1fr;
  gap: var(--numen-node-gap);
  min-block-size: var(--curve-slider-tile);
  margin: 0;
}
</style>
