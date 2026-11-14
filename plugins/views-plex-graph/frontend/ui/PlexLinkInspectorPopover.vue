<script setup lang="ts">
/**
 * In-Plex popover for the connection between two notes: the direction and description of each
 * record, adding the reverse direction, and removing a record or the whole link.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { ArrowLeftRight, X } from '@lucide/vue'
import { useOutsidePress } from '@numen/ui'
import type { Connection } from '@numen/plugin-sdk'
import type { LinkInspectorRequest } from '../types'
import {
  addReverseSide,
  cycleDirection,
  getNextDirection,
  getRows,
  isConnectionEmpty,
  removeSide,
  setSideLabel,
  type SideKey,
} from '../model/connectionDraft'
import { usePopoverPosition } from '../model/usePopoverPosition'
import { WORDS as words } from '../words'
import PlexLinkInspectorRow from './PlexLinkInspectorRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  request: LinkInspectorRequest
}>()

const emit = defineEmits<{
  (event: 'save', connection: Connection): void
  (event: 'remove' | 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const popover = useTemplateRef<HTMLDivElement>('popover')
const draft = ref<Connection>(props.request.connection)
const rows = computed(() => getRows(draft.value))

const { positionStyle } = usePopoverPosition({
  target: popover,
  at: () => props.request.at,
  watchTriggers: [() => props.request, () => rows.value.length],
})

useOutsidePress(popover, onDone)

/* -------------------------------- Handlers -------------------------------- */
function onCancel() {
  emit('dismiss')
}

function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.stopPropagation()
    event.preventDefault()
    onCancel()
  } else if (event.key === 'Enter') {
    event.stopPropagation()
    event.preventDefault()
    if (event.ctrlKey || event.metaKey) {
      onAddReverse()
    } else {
      onDone()
    }
  }
}

function onCycleDirection(key: SideKey) {
  draft.value = cycleDirection(draft.value, key)
}

function onLabelChanged(key: SideKey, label: string) {
  draft.value = setSideLabel(draft.value, key, label)
}

function onAddReverse() {
  draft.value = addReverseSide(draft.value)
}

function onRemoveRow(key: SideKey) {
  draft.value = removeSide(draft.value, key)
}

function onDone() {
  if (isConnectionEmpty(draft.value)) {
    emit('remove')
  } else {
    emit('save', draft.value)
  }
}
</script>

<template>
  <div ref="popover" class="plex-link-popover" :style="positionStyle" @keydown="onKeyDown">
    <div class="plex-link-popover__header">
      <div class="plex-link-popover__title">
        <span class="plex-link-popover__node-name">{{ props.request.nodeA.title }}</span>
        <ArrowLeftRight class="plex-link-popover__title-icon" />
        <span class="plex-link-popover__node-name">{{ props.request.nodeB.title }}</span>
      </div>
      <button
        type="button"
        class="plex-link-popover__close-btn"
        :title="words.cancel"
        :aria-label="words.cancel"
        @click="onCancel"
      >
        <X class="plex-link-popover__close-icon" />
      </button>
    </div>

    <div class="plex-link-popover__body">
      <div v-if="rows.length === 0" class="plex-link-popover__empty-state">
        {{ words.linkWillBeRemoved }}
      </div>
      <!-- Keyed by place: the one row keeps its button and its focus while its
           record moves from one note to the other. -->
      <PlexLinkInspectorRow
        v-for="(row, index) in rows"
        :key="index"
        :label="row.side.label"
        :author="row.key === 'first' ? props.request.nodeA.title : props.request.nodeB.title"
        :direction="row.direction"
        :next-direction="getNextDirection(row.direction, row.key, rows.length)"
        :first-title="props.request.nodeA.title"
        :second-title="props.request.nodeB.title"
        @cycle-direction="onCycleDirection(row.key)"
        @update:label="onLabelChanged(row.key, $event)"
        @remove="onRemoveRow(row.key)"
        @done="onDone"
        @add-reverse="onAddReverse"
      />
    </div>

    <div class="plex-link-popover__footer">
      <button
        v-if="rows.length < 2"
        type="button"
        class="plex-link-popover__btn plex-link-popover__btn--add"
        :title="rows.length === 0 ? words.addLink : words.addReverseDirection"
        @click="onAddReverse"
      >
        <span>+ {{ rows.length === 0 ? words.addLink : words.addReverseDirection }}</span>
        <span class="plex-link-popover__shortcut">Ctrl ↵</span>
      </button>
      <div v-else class="plex-link-popover__add-spacer" />

      <button
        type="button"
        class="plex-link-popover__btn plex-link-popover__btn--done"
        @click="onDone"
      >
        <span>{{ words.apply }}</span>
        <span class="plex-link-popover__shortcut">↵</span>
      </button>
    </div>
  </div>
</template>

<style scoped src="./plex-link-popover.css"></style>
