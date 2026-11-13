<script setup lang="ts">
/**
 * The plugins that run: for each its name, its health, how long it has run, and whether it is in use
 * or when the core stops it for idleness.
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
      <li v-for="row in rows" :key="row.id" class="plugin-popover__plugin" :aria-label="row.name">
        <div class="plugin-popover__head" :title="row.id">
          <span class="plugin-popover__name">{{ row.name }}</span>
          <span
            class="plugin-popover__health"
            :class="`plugin-popover__health--${row.health}`"
            :title="row.healthText"
          >
            {{ row.healthText }}
          </span>
        </div>
        <dl class="plugin-popover__facts">
          <div
            v-for="one in row.facts"
            :key="one.id"
            class="plugin-popover__fact"
            :title="one.hint || undefined"
          >
            <dt class="plugin-popover__label">{{ one.label }}</dt>
            <dd class="plugin-popover__value">{{ one.value }}</dd>
          </div>
        </dl>
        <p v-if="row.isInCore" class="plugin-popover__note">{{ words.inCore }}</p>
      </li>
    </ul>
  </div>
</template>
