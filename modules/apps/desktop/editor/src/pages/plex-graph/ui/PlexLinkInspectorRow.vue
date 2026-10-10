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
  gap: 0.375rem;
  block-size: 2.125rem;
  padding-inline: 0.375rem;
  background: var(--numen-field-bg, var(--numen-surface));
  border: var(--numen-stroke, 1px) solid var(--numen-field-border, var(--numen-rule));
  border-radius: var(--numen-radius, 0.375rem);
  transition: border-color var(--numen-motion-hover, 110ms) ease;
  min-inline-size: 0;
}
.plex-link-row:focus-within {
  border-color: var(--numen-ring, var(--numen-accent));
}
.plex-link-row__dir-btn,
.plex-link-row__delete-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: 1.5rem;
  block-size: 1.5rem;
  background: transparent;
  color: var(--numen-hushed);
  border: none;
  cursor: pointer;
  flex-shrink: 0;
  padding: 0;
  transition: color var(--numen-motion-hover, 110ms) ease;
}
.plex-link-row__dir-btn:hover:not(:disabled) {
  background: transparent;
  color: var(--numen-ink);
}
.plex-link-row__dir-btn:disabled,
.plex-link-row__dir-btn--static {
  cursor: default;
  opacity: 0.5;
}
.plex-link-row__delete-btn:hover {
  background: transparent;
  color: var(--numen-alarm);
}
.plex-link-row__icon {
  inline-size: 0.95rem;
  block-size: 0.95rem;
  stroke-width: 2;
}
.plex-link-row__icon--trash {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  opacity: 0.6;
  transition: opacity var(--numen-motion-hover, 110ms) ease;
}
.plex-link-row__delete-btn:hover .plex-link-row__icon--trash {
  opacity: 1;
}
.plex-link-row__input {
  flex: 1;
  min-inline-size: 0;
  block-size: 100%;
  border: none;
  background: transparent;
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2, 0.8125rem);
  outline: none;
  padding: 0 0.25rem;
}
.plex-link-row__input::placeholder {
  color: var(--numen-hushed);
  opacity: 0.75;
}
</style>
