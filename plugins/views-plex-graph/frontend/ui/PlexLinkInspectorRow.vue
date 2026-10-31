<script setup lang="ts">
/**
 * One record of the connection in PlexLinkInspectorPopover: the note that writes it, whether it
 * is written, its label and whether it draws an arrow at the other note.
 */
import { useId } from 'vue'
import { ArrowLeft, ArrowRight, Minus } from '@lucide/vue'
import type { ConnectionSide } from '@numen/plugin-sdk'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  side: ConnectionSide | null
  /** The title of the note that writes the record. */
  from: string
  /** The title of the note the record is about. */
  to: string
  /** Whether the record points back, against the order the two notes are shown in. */
  isBackward: boolean
}>()

const emit = defineEmits<{
  (event: 'toggle-arrow' | 'done'): void
  (event: 'update:present', isPresent: boolean): void
  (event: 'update:label', label: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const presentFieldId = useId()
const labelFieldId = useId()

/* -------------------------------- Handlers -------------------------------- */
function onPresentChange(event: Event) {
  emit('update:present', (event.target as HTMLInputElement).checked)
}

function onInput(event: Event) {
  emit('update:label', (event.target as HTMLInputElement).value)
}

function onInputKeyDown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    event.stopPropagation()
    event.preventDefault()
    emit('done')
  }
}
</script>

<template>
  <div class="plex-link-row" :class="{ 'plex-link-row--absent': props.side === null }">
    <label :for="presentFieldId" class="plex-link-row__writer">
      <input
        :id="presentFieldId"
        type="checkbox"
        class="plex-link-row__present"
        :checked="props.side !== null"
        @change="onPresentChange"
      />
      <span class="plex-link-row__names">
        <span class="plex-link-row__name">{{ props.from }}</span>
        <ArrowLeft v-if="props.isBackward" class="plex-link-row__icon" />
        <ArrowRight v-else class="plex-link-row__icon" />
        <span class="plex-link-row__name">{{ props.to }}</span>
      </span>
    </label>

    <div class="plex-link-row__fields">
      <label :for="labelFieldId" class="sr-only">{{ words.linkDescription }}</label>
      <input
        :id="labelFieldId"
        :value="props.side?.label ?? ''"
        :disabled="props.side === null"
        type="text"
        class="plex-link-row__input"
        :placeholder="words.linkDescriptionPlaceholder"
        @input="onInput"
        @keydown="onInputKeyDown"
      />
      <button
        type="button"
        class="plex-link-row__arrow-btn"
        :class="{ 'plex-link-row__arrow-btn--on': props.side?.marker === 'arrow' }"
        :disabled="props.side === null"
        :aria-pressed="props.side?.marker === 'arrow'"
        :title="words.toggleArrow"
        :aria-label="words.toggleArrow"
        @click="emit('toggle-arrow')"
      >
        <Minus v-if="props.side?.marker !== 'arrow'" class="plex-link-row__icon" />
        <ArrowLeft v-else-if="props.isBackward" class="plex-link-row__icon" />
        <ArrowRight v-else class="plex-link-row__icon" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.plex-link-row {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-inline-size: 0;
}
.plex-link-row__writer {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
  cursor: pointer;
}
.plex-link-row--absent .plex-link-row__writer {
  color: var(--numen-hushed);
}
.plex-link-row__names {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  min-inline-size: 0;
}
.plex-link-row__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-inline-size: 9rem;
}
.plex-link-row__fields {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  block-size: 2.125rem;
  padding-inline: 0.375rem;
  background: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  transition: border-color var(--numen-motion-hover) ease;
  min-inline-size: 0;
}
.plex-link-row__fields:focus-within {
  border-color: var(--numen-ring);
}
.plex-link-row--absent .plex-link-row__fields {
  opacity: 0.5;
}
.plex-link-row__arrow-btn {
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
  transition: color var(--numen-motion-hover) ease;
}
.plex-link-row__arrow-btn:hover:not(:disabled),
.plex-link-row__arrow-btn--on {
  color: var(--numen-ink);
}
.plex-link-row__arrow-btn:disabled {
  cursor: default;
}
.plex-link-row__icon {
  inline-size: 0.95rem;
  block-size: 0.95rem;
  stroke-width: 2;
  flex-shrink: 0;
}
.plex-link-row__input {
  flex: 1;
  min-inline-size: 0;
  block-size: 100%;
  border: none;
  background: transparent;
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  outline: none;
  padding: 0 0.25rem;
}
.plex-link-row__input::placeholder {
  color: var(--numen-hushed);
  opacity: 0.75;
}
</style>
