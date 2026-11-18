<script setup lang="ts">
/**
 * One row in the Logs viewer: collapsed summary showing timestamp, severity badge, source, message
 * and property key chips; clicking expands the row to display formatted SQL and JSON properties.
 */
import { computed } from 'vue'
import { formatEntryProperties } from '../lib/format'
import type { LogRow } from '../types'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  row: LogRow
  isExpanded: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  toggle: [id: number]
}>()

/* --------------------------------- State ---------------------------------- */
const formattedProperties = computed(() =>
  props.isExpanded ? formatEntryProperties(props.row.entry.properties) : [],
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  emit('toggle', props.row.id)
}
</script>

<template>
  <li class="log-row" :class="`log-row--${row.entry.level}`">
    <button type="button" class="log-row__summary" :aria-expanded="isExpanded" @click="onToggle">
      <span class="log-row__time">{{ row.timeText }}</span>
      <span class="log-row__level" :class="`log-row__level--${row.entry.level}`">
        {{ row.entry.level }}
      </span>
      <span class="log-row__source">{{ row.entry.source }}</span>
      <span class="log-row__message">{{ row.entry.message }}</span>
      <span v-if="row.keys.length > 0" class="log-row__keys">
        <span v-for="key in row.keys" :key="key" class="log-row__key-chip">{{ key }}</span>
      </span>
    </button>

    <div v-if="isExpanded" class="log-row__details">
      <p v-if="formattedProperties.length === 0" class="log-row__empty-props">
        {{ words.noProperties }}
      </p>
      <dl v-else class="log-row__props">
        <div
          v-for="prop in formattedProperties"
          :key="prop.key"
          class="log-row__prop"
          :class="`log-row__prop--${prop.kind}`"
        >
          <dt class="log-row__prop-key">{{ prop.key }}</dt>
          <dd class="log-row__prop-value">
            <pre
              v-if="prop.kind === 'sql' || prop.kind === 'json'"
              class="log-row__code"
            ><code>{{ prop.display }}</code></pre>
            <span v-else class="log-row__scalar">{{ prop.display }}</span>
          </dd>
        </div>
      </dl>
    </div>
  </li>
</template>
