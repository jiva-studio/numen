<script setup lang="ts">
/**
 * In-Plex popover for inspecting, managing direction, dual descriptions and deletion of links.
 */
import { ref, useTemplateRef } from 'vue'
import { ArrowLeftRight, X } from '@lucide/vue'
import { useOutsidePress } from '@numen/ui'
import type { DirectionalLinkItem, LinkInspectorRequest } from '../types'
import {
  addReverseLink,
  getLinksToSave,
  removeLink,
  setLinkLabel,
  toggleLinkDirection,
} from '../model/linkInspector'
import { usePopoverPosition } from '../model/usePopoverPosition'
import { WORDS as words } from '../words'
import PlexLinkInspectorRow from './PlexLinkInspectorRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  request: LinkInspectorRequest
}>()

const emit = defineEmits<{
  (event: 'save', rows: readonly DirectionalLinkItem[]): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const popover = useTemplateRef<HTMLDivElement>('popover')
const localRows = ref<readonly DirectionalLinkItem[]>(props.request.links)

const { positionStyle } = usePopoverPosition({
  target: popover,
  at: () => props.request.at,
  watchTriggers: [() => props.request, () => localRows.value.length],
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
      onAddReverseRow()
    } else {
      onDone()
    }
  }
}

function onToggleDirection() {
  localRows.value = toggleLinkDirection(localRows.value, props.request.nodeA.path)
}

function onLabelChanged(row: DirectionalLinkItem, label: string) {
  localRows.value = setLinkLabel(localRows.value, row.id, label)
}

function onAddReverseRow() {
  localRows.value = addReverseLink(
    localRows.value,
    props.request.links,
    props.request.nodeA.path,
    props.request.nodeB.path,
  )
}

function onRemoveRow(row: DirectionalLinkItem) {
  localRows.value = removeLink(localRows.value, row.id)
}

function onDone() {
  emit('save', getLinksToSave(localRows.value))
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
      <div v-if="localRows.length === 0" class="plex-link-popover__empty-state">
        {{ words.linkWillBeRemoved }}
      </div>
      <PlexLinkInspectorRow
        v-for="(row, index) in localRows"
        :key="index"
        :row="row"
        :is-static="localRows.length > 1"
        @toggle-direction="onToggleDirection"
        @update:label="onLabelChanged(row, $event)"
        @remove="onRemoveRow(row)"
        @done="onDone"
        @add-reverse="onAddReverseRow"
      />
    </div>

    <div class="plex-link-popover__footer">
      <button
        v-if="localRows.length < 2"
        type="button"
        class="plex-link-popover__btn plex-link-popover__btn--add"
        :title="localRows.length === 0 ? words.addLink : words.addReverseDirection"
        @click="onAddReverseRow"
      >
        <span>+ {{ localRows.length === 0 ? words.addLink : words.addReverseDirection }}</span>
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
