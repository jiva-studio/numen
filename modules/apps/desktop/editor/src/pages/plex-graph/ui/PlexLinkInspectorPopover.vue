<script setup lang="ts">
/**
 * In-Plex popover for inspecting, managing direction, dual descriptions and deletion of links.
 */
import { computed, nextTick, onMounted, onScopeDispose, ref, useTemplateRef, watch } from 'vue'
import { ArrowLeftRight, X } from '@lucide/vue'
import type { PlexRelatedSeat } from '@numen/ui'
import type { DirectionalLinkItem, LinkInspectorRequest, LinkInspectorSavePayload } from '../types'
import {
  getInverseSeat,
  resolveInspectorDirectionToggle,
  resolveInspectorDonePayload,
  resolvePopoverPosition,
} from '../model/linkInspector'
import { WORDS as words } from '../words'
import PlexLinkInspectorRow from './PlexLinkInspectorRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  request: LinkInspectorRequest
}>()

const emit = defineEmits<{
  (event: 'save', payload: LinkInspectorSavePayload): void
  (event: 'remove-entire-link', pairKey: string): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const popover = useTemplateRef<HTMLDivElement>('popover')
const localRows = ref<DirectionalLinkItem[]>(props.request.links.map((link) => ({ ...link })))
const removedLinks = ref<{ from: string; to: string; role?: PlexRelatedSeat }[]>([])
const initialLinkIds = new Set(props.request.links.map((link) => link.id))

const placedPosition = ref<{ left: number; top: number }>(
  resolvePopoverPosition({
    at: props.request.at,
    popoverSize: { width: 320, height: 160 },
    containerSize: {
      width: typeof window !== 'undefined' ? window.innerWidth : 800,
      height: typeof window !== 'undefined' ? window.innerHeight : 600,
    },
  }),
)

const positionStyle = computed(() => ({
  left: `${placedPosition.value.left}px`,
  top: `${placedPosition.value.top}px`,
}))

/* --------------------------------- Hooks ---------------------------------- */
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  window.addEventListener('pointerdown', onWindowPointerDown)
  window.addEventListener('resize', onWindowResize)
  void nextTick(() => {
    updatePosition()
    if (typeof ResizeObserver !== 'undefined' && popover.value) {
      resizeObserver = new ResizeObserver(() => {
        updatePosition()
      })
      resizeObserver.observe(popover.value)
      const parent = popover.value.offsetParent || popover.value.parentElement
      if (parent) resizeObserver.observe(parent)
    }
  })
})

onScopeDispose(() => {
  window.removeEventListener('pointerdown', onWindowPointerDown)
  window.removeEventListener('resize', onWindowResize)
  resizeObserver?.disconnect()
})

watch(
  () => props.request,
  () => {
    updatePosition()
  },
  { deep: true },
)

watch(
  () => localRows.value.length,
  async () => {
    await nextTick()
    updatePosition()
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onWindowPointerDown(event: PointerEvent) {
  if (!popover.value?.contains(event.target as Node)) {
    onDone()
  }
}

function onWindowResize() {
  updatePosition()
}

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

function onToggleDirection(row: DirectionalLinkItem) {
  if (localRows.value.length > 1) return
  const res = resolveInspectorDirectionToggle(
    row,
    props.request.nodeA.path,
    props.request.nodeB.path,
    initialLinkIds,
  )
  if (res.removedLink) removedLinks.value.push(res.removedLink)
  const target = row as {
    direction: string
    from: string
    to: string
    role: PlexRelatedSeat
  }
  target.direction = res.direction
  target.from = res.from
  target.to = res.to
  target.role = res.role
}

function onAddReverseRow() {
  if (localRows.value.length >= 2) return
  if (localRows.value.length === 0) {
    const baseRole = props.request.links[0]?.role ?? 'jump'
    localRows.value.push({
      id: `${props.request.nodeA.path}->${props.request.nodeB.path}`,
      from: props.request.nodeA.path,
      to: props.request.nodeB.path,
      role: baseRole,
      direction: 'forward',
      description: '',
    })
    return
  }
  const first = localRows.value[0]!
  const baseRole = first.from === props.request.nodeA.path ? first.role : getInverseSeat(first.role)

  localRows.value = [
    {
      ...first,
      id: `${props.request.nodeA.path}->${props.request.nodeB.path}`,
      from: props.request.nodeA.path,
      to: props.request.nodeB.path,
      role: baseRole,
      direction: 'forward',
    },
    {
      id: `${props.request.nodeB.path}->${props.request.nodeA.path}`,
      from: props.request.nodeB.path,
      to: props.request.nodeA.path,
      role: getInverseSeat(baseRole),
      direction: 'reverse',
      description: '',
    },
  ]
}

function onRemoveRow(row: DirectionalLinkItem) {
  if (initialLinkIds.has(row.id)) {
    removedLinks.value.push({ from: row.from, to: row.to, role: row.role })
  }
  localRows.value = localRows.value.filter((r) => r.id !== row.id)
}

function onDone() {
  const payload = resolveInspectorDonePayload(
    localRows.value,
    props.request.links,
    initialLinkIds,
    props.request.pairKey,
    removedLinks.value,
  )
  emit('save', payload)
}

/* -------------------------------- Helpers --------------------------------- */
function getElementSize(el: HTMLElement): { width: number; height: number } {
  const rect = el.getBoundingClientRect()
  return {
    width: rect.width || el.offsetWidth || 320,
    height: rect.height || el.offsetHeight || 160,
  }
}

function getContainerSize(parent: HTMLElement | null): { width: number; height: number } {
  const rect = parent?.getBoundingClientRect()
  return {
    width: rect?.width || (typeof window !== 'undefined' ? window.innerWidth : 800),
    height: rect?.height || (typeof window !== 'undefined' ? window.innerHeight : 600),
  }
}

function updatePosition() {
  const el = popover.value
  if (!el) return
  const parent = (el.offsetParent as HTMLElement | null) || el.parentElement
  placedPosition.value = resolvePopoverPosition({
    at: props.request.at,
    popoverSize: getElementSize(el),
    containerSize: getContainerSize(parent),
  })
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
        v-for="row in localRows"
        :key="row.id"
        v-model:description="(row as { description: string }).description"
        :row="row"
        :is-static="localRows.length > 1"
        @toggle-direction="onToggleDirection(row)"
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
        <span>{{ words.done }}</span>
        <span class="plex-link-popover__shortcut">↵</span>
      </button>
    </div>
  </div>
</template>

<style scoped src="./plex-link-popover.css"></style>
