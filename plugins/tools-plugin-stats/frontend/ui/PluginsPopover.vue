<script setup lang="ts">
/**
 * The plugins that run, one line each: whether it is in use, its name, the worst health item of an
 * unhealthy plugin, and when the core stops an idle one. How long it has run is in the tooltip of
 * the line. Clicking a line shows who started the plugin and who holds it now.
 */
import { computed, ref } from 'vue'
import { Tooltip, useTooltip } from '@numen/ui'
import { ChevronIcon } from '../model/icons'
import type { PluginProcess } from '../types'
import { getRows } from '../lib/rows'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  processes: readonly PluginProcess[]
  now: number
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const tip = useTooltip()
const rows = computed(() => getRows(props.processes, props.now))
const expandedIds = ref<ReadonlySet<string>>(new Set())

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  emit('close')
}

function onToggleHolders(id: string) {
  const next = new Set(expandedIds.value)
  if (!next.delete(id)) next.add(id)
  expandedIds.value = next
}

function onEnterRow(event: Event, id: string) {
  if (
    event.target instanceof Element &&
    event.target.closest('.plugin-popover__countdown') !== null
  ) {
    return
  }
  tip.open(event, () => getRowTitle(id))
}

/* -------------------------------- Helpers --------------------------------- */
function getRowTitle(id: string): string {
  return rows.value.find((one) => one.id === id)?.title ?? ''
}

function getCountdownHint(id: string): string {
  return rows.value.find((one) => one.id === id)?.countdownHint ?? ''
}
</script>

<template>
  <div class="plugin-popover" role="dialog" aria-modal="false" :aria-label="words.title">
    <div class="plugin-popover__header">
      <span class="plugin-popover__title">{{ words.title }}</span>
      <button
        type="button"
        class="plugin-popover__close-btn"
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

    <p v-if="rows.length === 0" class="plugin-popover__empty">{{ words.none }}</p>
    <ul v-else class="plugin-popover__body">
      <li
        v-for="row in rows"
        :key="row.id"
        class="plugin-popover__plugin"
        :aria-label="row.name"
        :aria-expanded="expandedIds.has(row.id)"
        tabindex="0"
        @mouseenter="onEnterRow($event, row.id)"
        @mouseleave="tip.close"
        @focus="onEnterRow($event, row.id)"
        @blur="tip.close"
        @click="onToggleHolders(row.id)"
        @keydown.enter.space.prevent="onToggleHolders(row.id)"
      >
        <span
          class="plugin-popover__status"
          :class="{ 'plugin-popover__status--in-use': row.isInUse }"
          role="img"
          :aria-label="row.status"
        ></span>
        <span class="plugin-popover__name-cell">
          <span class="plugin-popover__name">{{ row.name }}</span>
          <span v-if="row.isInCore" class="plugin-popover__tag">
            {{ words.inCore }}
          </span>
          <span
            v-if="row.healthText"
            class="plugin-popover__health"
            :class="`plugin-popover__health--${row.health}`"
          >
            {{ row.healthText }}
          </span>
        </span>
        <span
          class="plugin-popover__countdown"
          @mouseenter="tip.open($event, () => getCountdownHint(row.id))"
          @mouseleave="tip.close"
        >
          {{ row.countdown }}
        </span>
        <ChevronIcon class="plugin-popover__chevron" aria-hidden="true" />
        <dl v-if="expandedIds.has(row.id)" class="plugin-popover__holders">
          <dt class="plugin-popover__holders-term">{{ words.startedBy }}</dt>
          <dd class="plugin-popover__holders-value">{{ row.startedBy }}</dd>
          <dt class="plugin-popover__holders-term">{{ words.heldBy }}</dt>
          <dd v-if="row.holders.length === 0" class="plugin-popover__holders-value">
            {{ words.notHeld }}
          </dd>
          <dd
            v-for="(holder, index) in row.holders"
            :key="index"
            class="plugin-popover__holders-value"
          >
            {{ holder }}
          </dd>
        </dl>
      </li>
    </ul>
    <Teleport to="body">
      <Tooltip v-if="tip.at.value && tip.text.value" :id="tip.id" :at="tip.at.value">
        {{ tip.text.value }}
      </Tooltip>
    </Teleport>
  </div>
</template>
