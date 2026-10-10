<script setup lang="ts">
/**
 * In-Plex popover for inspecting, managing direction, dual descriptions and deletion of links.
 */
import { computed, onMounted, onScopeDispose, ref, useTemplateRef } from 'vue'
import type { PlexRelatedSeat } from '@numen/ui'
import type {
  DirectionalLinkItem,
  LinkDirectionMode,
  LinkInspectorRequest,
  LinkInspectorSavePayload,
} from '../types'
import { getInverseSeat } from '../model/usePlexTab'
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

const positionStyle = computed(() => ({
  left: `${props.request.at.x}px`,
  top: `${props.request.at.y}px`,
}))

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  window.addEventListener('pointerdown', onWindowPointerDown)
})

onScopeDispose(() => {
  window.removeEventListener('pointerdown', onWindowPointerDown)
})

/* -------------------------------- Handlers -------------------------------- */
function onWindowPointerDown(event: PointerEvent) {
  if (!popover.value?.contains(event.target as Node)) {
    onDone()
  }
}

function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.stopPropagation()
    event.preventDefault()
    emit('dismiss')
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
  const nextDirection: Record<LinkDirectionMode, LinkDirectionMode> = {
    undirected: 'forward',
    forward: 'reverse',
    reverse: 'undirected',
  }
  const next = nextDirection[row.direction]
  const mutableRow = row as {
    from: string
    to: string
    role: PlexRelatedSeat
    direction: LinkDirectionMode
  }

  const baseRole = row.from === props.request.nodeA.path ? row.role : getInverseSeat(row.role)
  mutableRow.direction = next

  if (next === 'reverse') {
    if (initialLinkIds.has(row.id) && row.from === props.request.nodeA.path) {
      removedLinks.value.push({
        from: props.request.nodeA.path,
        to: props.request.nodeB.path,
        role: baseRole,
      })
    }
    mutableRow.from = props.request.nodeB.path
    mutableRow.to = props.request.nodeA.path
    mutableRow.role = getInverseSeat(baseRole)
  } else {
    if (initialLinkIds.has(row.id) && row.from === props.request.nodeB.path) {
      removedLinks.value.push({
        from: props.request.nodeB.path,
        to: props.request.nodeA.path,
        role: getInverseSeat(baseRole),
      })
    }
    mutableRow.from = props.request.nodeA.path
    mutableRow.to = props.request.nodeB.path
    mutableRow.role = baseRole
  }
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
  let finalRows = [...localRows.value]
  const autoRemoved: { from: string; to: string; role?: PlexRelatedSeat }[] = []

  if (finalRows.length === 2) {
    const row0Empty = !finalRows[0]!.description.trim()
    const row1Empty = !finalRows[1]!.description.trim()

    if (row0Empty && !row1Empty) {
      if (initialLinkIds.has(finalRows[0]!.id)) {
        autoRemoved.push({
          from: finalRows[0]!.from,
          to: finalRows[0]!.to,
          role: finalRows[0]!.role,
        })
      }
      finalRows = [finalRows[1]!]
    } else if (row1Empty && !row0Empty) {
      if (initialLinkIds.has(finalRows[1]!.id)) {
        autoRemoved.push({
          from: finalRows[1]!.from,
          to: finalRows[1]!.to,
          role: finalRows[1]!.role,
        })
      }
      finalRows = [finalRows[0]!]
    }
  }

  emit('save', {
    pairKey: props.request.pairKey,
    rows: finalRows,
    removedLinks: [...removedLinks.value, ...autoRemoved],
  })
}
</script>

<template>
  <div ref="popover" class="plex-link-popover" :style="positionStyle" @keydown="onKeyDown">
    <div class="plex-link-popover__header">
      <span class="plex-link-popover__title">
        {{ props.request.nodeA.title }} ⟷ {{ props.request.nodeB.title }}
      </span>
    </div>

    <div class="plex-link-popover__body">
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
        :title="words.addReverseDirection"
        @click="onAddReverseRow"
      >
        <span>+ {{ words.addReverseDirection }}</span>
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

<style scoped>
.plex-link-popover {
  position: absolute;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  min-width: 280px;
  background: var(--surface-1);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
  transform: translate(-50%, -100%) translateY(-12px);
  user-select: none;
}

.plex-link-popover__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: var(--space-1);
  border-bottom: 1px solid var(--border-subtle);
}

.plex-link-popover__title {
  font-size: var(--text-xs);
  font-weight: 500;
  color: var(--text-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 240px;
}

.plex-link-popover__body {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.plex-link-popover__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding-top: var(--space-1);
}

.plex-link-popover__add-spacer {
  flex: 1;
}

.plex-link-popover__btn {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  height: var(--size-icon-button);
  padding: 0 var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  cursor: pointer;
  border: none;
  transition: all 0.15s ease;
}

.plex-link-popover__btn--add {
  background: var(--surface-2);
  color: var(--text-2);
}

.plex-link-popover__btn--add:hover {
  background: var(--surface-3);
  color: var(--text-1);
}

.plex-link-popover__btn--done {
  background: var(--accent);
  color: var(--accent-foreground, #fff);
  font-weight: 500;
  margin-left: auto;
}

.plex-link-popover__btn--done:hover {
  opacity: 0.9;
}

.plex-link-popover__shortcut {
  font-size: 10px;
  opacity: 0.6;
}
</style>
