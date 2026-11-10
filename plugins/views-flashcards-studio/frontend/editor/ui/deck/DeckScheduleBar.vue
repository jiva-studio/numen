<script setup lang="ts">
/**
 * Renders the deck preset schedule bar and selector menu.
 */
import { computed, ref, useId } from 'vue'
import { Menu, type Position, measureRect } from '@numen/ui'
import { ChevronDown } from '@lucide/vue'
import type { Choice, DeckPreset } from '../../model/deck/useDeckSchedule'
import { WORDS as words } from '../../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  scheduled: DeckPreset
  choices: readonly Choice[]
}>()

const emit = defineEmits<{
  choose: [preset: string]
}>()

/* --------------------------------- State ---------------------------------- */
const labelId = useId()
const choiceId = useId()

const presetMenuItems = computed(() =>
  props.choices.map((one) => ({
    id: one.path,
    text: one.name,
    group: one.path === '' ? 'defaults' : 'presets',
  })),
)

const scheduleMenuTarget = ref<{ at: Position; from: HTMLElement } | null>(null)

/* -------------------------------- Handlers -------------------------------- */
function onOpenScheduleMenu(event: Event) {
  const line = event.currentTarget
  if (!(line instanceof HTMLElement)) return
  const box = measureRect(line)
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
    <span :id="labelId" class="deck-tab__by">{{ words.scheduledBy }}</span>
    <button
      :id="choiceId"
      type="button"
      class="deck-tab__choice"
      aria-haspopup="menu"
      :aria-labelledby="`${labelId} ${choiceId}`"
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
      :items="presetMenuItems"
      :at="scheduleMenuTarget.at"
      :from="scheduleMenuTarget.from"
      :current="props.scheduled.path"
      is-open
      trigger="keyboard"
      :name="words.scheduledBy"
      @choose="onChooseSchedule"
      @dismiss="onDismissScheduleMenu"
    />
  </p>
</template>

<style scoped>
.deck-tab__scheduled {
  --scheduled-gap: 0.5rem;
  --scheduled-padding-block-start: 0.75rem;
  --scheduled-padding-inline: 1rem;
  --scheduled-padding-block-end: 0.6rem;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--scheduled-gap);
  margin: 0;
  padding: var(--scheduled-padding-block-start) var(--scheduled-padding-inline)
    var(--scheduled-padding-block-end);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
}

.deck-tab__by {
  color: var(--numen-hushed);
}

.deck-tab__choice {
  --choice-padding-block: 0.15rem;
  --choice-padding-inline: 0.4rem;
  display: inline-flex;
  align-items: center;
  gap: var(--numen-node-gap);
  padding: var(--choice-padding-block) var(--choice-padding-inline);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  line-height: 1.2;
  cursor: pointer;
}

.deck-tab__choice:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}

.deck-tab__icon {
  --icon-inline-size: 1em;
  --icon-block-size: 1em;
  inline-size: var(--icon-inline-size);
  block-size: var(--icon-block-size);
}

.deck-tab__wrong {
  color: var(--numen-alarm);
  overflow-wrap: anywhere;
}
</style>
