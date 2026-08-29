<script setup lang="ts">
/**
 * What a person has been doing in this vault: the days they answered on, and
 * how many of them run up to now without a gap.
 *
 * It is read at a glance and stands above what they came here to do.
 */
import { computed } from 'vue'
import { Heatmap } from '@numen/ui'

const props = defineProps<{
  /** How many cards were answered on each day, by the day it was. */
  days: ReadonlyMap<string, number>
  /** How many cards fall on each day still to come, by the day they fall on. */
  due: ReadonlyMap<string, number>
  /** How many days up to now were reviewed without a gap. */
  streak: number
}>()

/** The streak, in the words a person would say it in. */
const kept = computed(() => {
  if (props.streak === 0) return ''
  return props.streak === 1 ? '1 day in a row' : `${props.streak} days in a row`
})
</script>

<template>
  <section class="done">
    <Heatmap :did="days" :due="due" />
    <p v-if="kept" class="done__streak">{{ kept }}</p>
  </section>
</template>

<style scoped>
/* The grid keeps its own height, whatever the list below it does. */
.done {
  display: flex;
  flex: none;
  flex-direction: column;
  gap: var(--numen-inset);
}

.done__streak {
  margin: 0;
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
}
</style>
