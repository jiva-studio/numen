<script setup lang="ts">
/**
 * Renders the deck preset schedule bar and selector menu.
 */
import { computed, ref } from 'vue'
import { Menu } from '@numen/ui'
import type { Position } from '@numen/ui'
import { ChevronDown } from '@lucide/vue'
import type { Choice, DeckPreset } from '../composables/useDeckSchedule'
import { WORDS as words } from '../../../entities/deck/words'

// --- Props & Emits ---
const props = defineProps<{
  scheduled: DeckPreset
  choices: readonly Choice[]
}>()

const emit = defineEmits<{
  choose: [preset: string]
}>()

// --- State ---
const offeredPresets = computed(() =>
  props.choices.map((one) => ({
    id: one.path,
    text: one.name,
    group: one.path === '' ? 'defaults' : 'presets',
  })),
)

const scheduleMenuTarget = ref<{ at: Position; from: HTMLElement } | null>(null)

// --- Handlers ---
function onOpenScheduleMenu(event: Event) {
  const line = event.currentTarget
  if (!(line instanceof HTMLElement)) return
  const box = line.getBoundingClientRect()
  scheduleMenuTarget.value = { at: { x: box.left, y: box.bottom }, from: line }
}

function onChooseSchedule(path: string) {
  scheduleMenuTarget.value = null
  emit('choose', path)
}

function onDismissScheduleMenu() {
  scheduleMenuTarget.value = null
}
</script>

<template>
  <p class="deck-tab__scheduled">
    <span id="deck-scheduled" class="deck-tab__by">{{ words.scheduledBy }}</span>
    <button
      type="button"
      class="deck-tab__choice"
      aria-haspopup="menu"
      aria-labelledby="deck-scheduled"
      @click="onOpenScheduleMenu"
    >
      {{ props.scheduled.name }}
      <ChevronDown class="deck-tab__icon" aria-hidden="true" />
    </button>
    <span v-if="props.scheduled.errorMessage" role="status" class="deck-tab__wrong">
      {{ props.scheduled.errorMessage }}
    </span>

    <Menu
      v-if="scheduleMenuTarget"
      :items="offeredPresets"
      :at="scheduleMenuTarget.at"
      :from="scheduleMenuTarget.from"
      :current="props.scheduled.path"
      open
      opening="keyboard"
      :name="words.scheduledBy"
      @choose="onChooseSchedule"
      @dismiss="onDismissScheduleMenu"
    />
  </p>
</template>

<style scoped>
.deck-tab__scheduled {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  padding: 0.75rem 1rem 0.6rem;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
}

.deck-tab__by {
  color: var(--numen-hushed);
}

.deck-tab__choice {
  display: inline-flex;
  align-items: center;
  gap: var(--numen-node-gap);
  padding: 0.15rem 0.4rem;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  line-height: 1.2;
  cursor: pointer;
}

.deck-tab__choice:focus-visible {
  outline: none;
  outline-offset: var(--numen-stroke);
}

.deck-tab__icon {
  inline-size: 1em;
  block-size: 1em;
}

.deck-tab__wrong {
  color: var(--numen-alarm);
  overflow-wrap: anywhere;
}
</style>
