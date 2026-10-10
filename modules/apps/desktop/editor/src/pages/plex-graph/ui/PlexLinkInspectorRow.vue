<script setup lang="ts">
/**
 * Single direction and description row in PlexLinkInspectorPopover.
 */
import { ArrowLeft, ArrowRight, Minus, Trash2 } from '@lucide/vue'
import type { DirectionalLinkItem } from '../types'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  row: DirectionalLinkItem
  isStatic: boolean
}>()

const description = defineModel<string>('description', { required: true })

const emit = defineEmits<{
  (event: 'toggle-direction'): void
  (event: 'remove'): void
  (event: 'done'): void
  (event: 'add-reverse'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onInputKeyDown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    event.stopPropagation()
    event.preventDefault()
    if (event.ctrlKey || event.metaKey) {
      emit('add-reverse')
    } else {
      emit('done')
    }
  }
}
</script>

<template>
  <div class="plex-link-row">
    <button
      type="button"
      class="plex-link-row__dir-btn plex-link-popover__dir-btn"
      :class="{
        'plex-link-row__dir-btn--static': props.isStatic,
        'plex-link-popover__dir-btn--static': props.isStatic,
      }"
      :disabled="props.isStatic"
      :title="props.isStatic ? undefined : words.changeDirection"
      @click="emit('toggle-direction')"
    >
      <ArrowRight
        v-if="props.row.direction === 'forward'"
        class="plex-link-row__icon plex-link-popover__icon"
      />
      <ArrowLeft
        v-else-if="props.row.direction === 'reverse'"
        class="plex-link-row__icon plex-link-popover__icon"
      />
      <Minus v-else class="plex-link-row__icon plex-link-popover__icon" />
    </button>

    <input
      v-model="description"
      type="text"
      class="plex-link-row__input plex-link-popover__input"
      :placeholder="words.linkDescriptionPlaceholder"
      @keydown="onInputKeyDown"
    />

    <button
      type="button"
      class="plex-link-row__delete-btn plex-link-popover__delete-btn"
      :title="words.removeLink"
      @click="emit('remove')"
    >
      <Trash2
        class="plex-link-row__icon plex-link-row__icon--trash plex-link-popover__icon plex-link-popover__icon--trash"
      />
    </button>
  </div>
</template>

<style scoped>
.plex-link-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.plex-link-row__dir-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--size-icon-button);
  height: var(--size-icon-button);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  color: var(--text-2);
  cursor: pointer;
  border: none;
  flex-shrink: 0;
  transition: all 0.15s ease;
}

.plex-link-row__dir-btn:hover:not(:disabled) {
  background: var(--surface-3);
  color: var(--text-1);
}

.plex-link-row__dir-btn:disabled,
.plex-link-row__dir-btn--static {
  cursor: default;
  opacity: 0.8;
}

.plex-link-row__icon {
  width: 14px;
  height: 14px;
}

.plex-link-row__input {
  flex: 1;
  min-width: 0;
  height: var(--size-icon-button);
  padding: 0 var(--space-2);
  background: var(--surface-1);
  color: var(--text-1);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  outline: none;
  transition: border-color 0.15s ease;
}

.plex-link-row__input:focus {
  border-color: var(--accent);
}

.plex-link-row__delete-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--size-icon-button);
  height: var(--size-icon-button);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-3);
  cursor: pointer;
  border: none;
  flex-shrink: 0;
  transition: all 0.15s ease;
}

.plex-link-row__delete-btn:hover {
  background: var(--surface-2);
  color: var(--danger, #e5484d);
}
</style>
