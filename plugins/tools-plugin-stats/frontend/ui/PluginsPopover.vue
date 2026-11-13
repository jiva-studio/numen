<script setup lang="ts">
/**
 * The plugins that run, one line each: its name, its health, and whether it is in use or when the
 * core stops it for idleness. How long it has run is in the tooltip of the line.
 */
import { computed } from 'vue'
import type { PluginProcess } from '../types'
import { getRows } from '../lib/rows'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  processes: readonly PluginProcess[]
  now: number
  locale?: string | undefined
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const rows = computed(() => getRows(props.processes, props.now, props.locale))

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  emit('close')
}
</script>

<template>
  <div class="plugin-popover" role="dialog" aria-modal="false" :aria-label="words.title">
    <div class="plugin-popover__header">
      <span class="plugin-popover__title">{{ words.title }}</span>
      <button
        type="button"
        class="plugin-popover__close-btn"
        :title="words.close"
        :aria-label="words.close"
        @click="onClose"
      >
        ✕
      </button>
    </div>

    <p v-if="rows.length === 0" class="plugin-popover__empty">{{ words.none }}</p>
    <ul v-else class="plugin-popover__body">
      <li
        v-for="row in rows"
        :key="row.id"
        class="plugin-popover__plugin"
        :aria-label="row.name"
        :title="row.title"
      >
        <span class="plugin-popover__name-cell">
          <span class="plugin-popover__name">{{ row.name }}</span>
          <span v-if="row.isInCore" class="plugin-popover__tag" :title="words.inCoreHint">
            {{ words.inCore }}
          </span>
        </span>
        <span
          class="plugin-popover__health"
          :class="`plugin-popover__health--${row.health}`"
          :title="row.healthLabel"
        >
          <span class="plugin-popover__dot" role="img" :aria-label="row.healthLabel"></span>
          <span v-if="row.healthText" class="plugin-popover__health-text" aria-hidden="true">
            {{ row.healthText }}
          </span>
        </span>
        <span class="plugin-popover__state">{{ row.state }}</span>
        <span class="plugin-popover__detail" :title="row.detailHint || undefined">
          <span v-if="row.detail" class="plugin-popover__separator" aria-hidden="true">·</span>
          {{ row.detail }}
        </span>
      </li>
    </ul>
  </div>
</template>
