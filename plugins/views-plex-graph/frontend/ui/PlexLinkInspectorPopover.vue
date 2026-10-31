<script setup lang="ts">
/**
 * In-Plex popover for editing the connection between two notes: the kind of link, and for each of
 * the two notes whether it writes a record, its label and whether it draws an arrow.
 */
import { computed, ref, useId, useTemplateRef } from 'vue'
import { ArrowLeftRight, Trash2, X } from '@lucide/vue'
import { useOutsidePress } from '@numen/ui'
import type { Connection, ConnectionRole } from '@numen/plugin-sdk'
import type { LinkInspectorRequest } from '../types'
import {
  setRole,
  setSideLabel,
  setSidePresent,
  toggleSideArrow,
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
const roleFieldId = useId()

const rows = computed(() => [
  {
    key: 'first' as const,
    side: draft.value.firstSide,
    from: props.request.nodeA.title,
    to: props.request.nodeB.title,
    isBackward: false,
  },
  {
    key: 'second' as const,
    side: draft.value.secondSide,
    from: props.request.nodeB.title,
    to: props.request.nodeA.title,
    isBackward: true,
  },
])

const { positionStyle } = usePopoverPosition({
  target: popover,
  at: () => props.request.at,
  watchTriggers: [() => props.request],
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
    onDone()
  }
}

function onRoleChange(event: Event) {
  draft.value = setRole(draft.value, (event.target as HTMLSelectElement).value as ConnectionRole)
}

function onPresentChanged(key: SideKey, isPresent: boolean) {
  draft.value = setSidePresent(draft.value, key, isPresent)
}

function onLabelChanged(key: SideKey, label: string) {
  draft.value = setSideLabel(draft.value, key, label)
}

function onArrowToggled(key: SideKey) {
  draft.value = toggleSideArrow(draft.value, key)
}

function onDone() {
  emit('save', draft.value)
}

function onRemove() {
  emit('remove')
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
      <div class="plex-link-popover__kind">
        <label :for="roleFieldId" class="plex-link-popover__kind-label">{{ words.linkKind }}</label>
        <select
          :id="roleFieldId"
          class="plex-link-popover__kind-select"
          :value="draft.role ?? ''"
          @change="onRoleChange"
        >
          <option v-if="draft.role === null" value="" disabled>{{ words.noLinkKind }}</option>
          <option v-for="role in props.request.roles" :key="role" :value="role">
            {{ words.linkKindOption(props.request.nodeB.title, role, props.request.nodeA.title) }}
          </option>
        </select>
      </div>

      <PlexLinkInspectorRow
        v-for="row in rows"
        :key="row.key"
        :side="row.side"
        :from="row.from"
        :to="row.to"
        :is-backward="row.isBackward"
        @update:present="onPresentChanged(row.key, $event)"
        @update:label="onLabelChanged(row.key, $event)"
        @toggle-arrow="onArrowToggled(row.key)"
        @done="onDone"
      />
    </div>

    <div class="plex-link-popover__footer">
      <button
        type="button"
        class="plex-link-popover__btn plex-link-popover__btn--remove"
        @click="onRemove"
      >
        <Trash2 class="plex-link-popover__icon" />
        <span>{{ words.removeConnection }}</span>
      </button>

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
