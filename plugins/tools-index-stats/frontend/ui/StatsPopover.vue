<script setup lang="ts">
/**
 * The details of the search index: the counts, the model, the profile, the speed and the sizes.
 */
import { computed } from 'vue'
import { Tooltip, useTooltip } from '@numen/ui'
import type { Stats } from '../api/stats'
import { getRows } from '../lib/rows'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  indexStats: Stats
  locale?: string | undefined
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const tip = useTooltip()
const rows = computed(() => getRows(props.indexStats, props.locale))

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  emit('close')
}
</script>

<template>
  <div class="index-popover" role="dialog" aria-modal="false" :aria-label="words.title">
    <div class="index-popover__header">
      <span class="index-popover__title">{{ words.title }}</span>
      <button
        type="button"
        class="index-popover__close-btn"
        :aria-label="words.close"
        @mouseenter="tip.open($event, words.close)"
        @mouseleave="tip.close"
        @focus="tip.open($event, words.close)"
        @blur="tip.close"
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
        @mouseenter="tip.open($event, row.hint)"
        @mouseleave="tip.close"
      >
        <dt class="index-popover__label">{{ row.label }}</dt>
        <dd class="index-popover__value">{{ row.value }}</dd>
      </div>
    </dl>
    <Teleport to="body">
      <Tooltip v-if="tip.at.value && tip.text.value" :id="tip.id" :at="tip.at.value">
        {{ tip.text.value }}
      </Tooltip>
    </Teleport>
  </div>
</template>
