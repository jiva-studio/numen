<script setup lang="ts">
/**
 * The details of the search index: the counts, the model, the profile, the speed and the sizes.
 */
import { computed } from 'vue'
import type { Stats } from './api/stats'
import { getRows } from './lib/rows'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  stats: Stats
  locale?: string | undefined
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const rows = computed(() => getRows(props.stats, props.locale))

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  emit('close')
}
</script>

<template>
  <div class="index-popover">
    <div class="index-popover__header">
      <span class="index-popover__title">{{ words.title }}</span>
      <button
        type="button"
        class="index-popover__close-btn"
        :title="words.close"
        :aria-label="words.close"
        @click="onClose"
      >
        ✕
      </button>
    </div>

    <dl class="index-popover__body">
      <div
        v-for="row in rows"
        :key="row.id"
        class="index-popover__row"
        :class="{ 'index-popover__row--nested': row.isNested }"
        :title="row.hint || undefined"
      >
        <dt class="index-popover__label">{{ row.label }}</dt>
        <dd class="index-popover__value">{{ row.value }}</dd>
      </div>
    </dl>
  </div>
</template>
