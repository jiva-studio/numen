<script setup lang="ts">
/**
 * One record of the connection in PlexLinkInspectorPopover: its direction, its description and
 * the button that removes it.
 */
import { computed, useId } from 'vue'
import { ArrowLeft, ArrowRight, Minus, Trash2 } from '@lucide/vue'
import type { LinkDirection } from '../model/connectionDraft'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  label: string
  author: string
  direction: LinkDirection
  nextDirection: LinkDirection
  firstTitle: string
  secondTitle: string
}>()

const emit = defineEmits<{
  (event: 'cycle-direction' | 'remove' | 'done' | 'add-reverse'): void
  (event: 'update:label', label: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const labelFieldId = useId()
const directionWords = computed(() => {
  const name = (direction: LinkDirection) =>
    words.direction[direction](props.firstTitle, props.secondTitle)
  return words.directionButton(name(props.direction), name(props.nextDirection))
})

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event) {
  emit('update:label', (event.target as HTMLInputElement).value)
}

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
  <div class="plex-link-row" :data-from="props.author">
    <button
      type="button"
      class="plex-link-row__dir-btn plex-link-popover__dir-btn"
      :data-direction="props.direction"
      :title="directionWords"
      :aria-label="directionWords"
      @click="emit('cycle-direction')"
    >
      <ArrowRight
        v-if="props.direction === 'forward'"
        class="plex-link-row__icon plex-link-popover__icon"
      />
      <ArrowLeft
        v-else-if="props.direction === 'backward'"
        class="plex-link-row__icon plex-link-popover__icon"
      />
      <Minus v-else class="plex-link-row__icon plex-link-popover__icon" />
    </button>

    <label :for="labelFieldId" class="sr-only">{{ words.linkDescription }}</label>
    <input
      :id="labelFieldId"
      :value="props.label"
      type="text"
      class="plex-link-row__input plex-link-popover__input"
      :placeholder="words.linkDescriptionPlaceholder"
      @input="onInput"
      @keydown="onInputKeyDown"
    />

    <button
      type="button"
      class="plex-link-row__delete-btn plex-link-popover__delete-btn"
      :title="words.removeLink"
      :aria-label="words.removeLink"
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
  --plex-link-row-gap: 0.375rem;
  --plex-link-row-block-size: 2.125rem;
  --plex-link-row-padding-inline: 0.375rem;
  display: flex;
  align-items: center;
  gap: var(--plex-link-row-gap);
  block-size: var(--plex-link-row-block-size);
  padding-inline: var(--plex-link-row-padding-inline);
  background: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  transition: border-color var(--numen-motion-hover) ease;
  min-inline-size: 0;
}
.plex-link-row:focus-within {
  border-color: var(--numen-ring);
}
.plex-link-row__dir-btn,
.plex-link-row__delete-btn {
  --dir-btn-inline-size: 1.5rem;
  --dir-btn-block-size: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: var(--dir-btn-inline-size);
  block-size: var(--dir-btn-block-size);
  background: transparent;
  color: var(--numen-hushed);
  border: none;
  cursor: pointer;
  flex-shrink: 0;
  padding: 0;
  transition: color var(--numen-motion-hover) ease;
}
.plex-link-row__dir-btn {
  border-radius: var(--numen-radius);
}
.plex-link-row__dir-btn:hover {
  background: var(--numen-raised);
  color: var(--numen-ink);
}
.plex-link-row__dir-btn:focus-visible,
.plex-link-row__delete-btn:focus-visible {
  --dir-btn-focus-visible-outline-offset: 0.0625rem;
  outline: var(--numen-stroke) solid var(--numen-ring);
  outline-offset: var(--dir-btn-focus-visible-outline-offset);
}
.plex-link-row__delete-btn:hover {
  background: transparent;
  color: var(--numen-alarm);
}
.plex-link-row__icon {
  --icon-inline-size: 0.95rem;
  --icon-block-size: 0.95rem;
  inline-size: var(--icon-inline-size);
  block-size: var(--icon-block-size);
  stroke-width: 2;
}
.plex-link-row__icon--trash {
  --icon-trash-inline-size: 0.875rem;
  --icon-trash-block-size: 0.875rem;
  inline-size: var(--icon-trash-inline-size);
  block-size: var(--icon-trash-block-size);
  opacity: 0.6;
  transition: opacity var(--numen-motion-hover) ease;
}
.plex-link-row__delete-btn:hover .plex-link-row__icon--trash {
  opacity: 1;
}
.plex-link-row__input {
  --input-padding: 0.25rem;
  flex: 1;
  min-inline-size: 0;
  block-size: 100%;
  border: none;
  background: transparent;
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  outline: none;
  padding: 0 var(--input-padding);
}
.plex-link-row__input::placeholder {
  color: var(--numen-hushed);
  opacity: 0.75;
}
</style>
