<script setup lang="ts">
/**
 * A deck tab: the cards as a grid under the sections they stand in, and the
 * questions the file puts.
 *
 * The grid takes cards and hands identities back, and is handed what is wrong
 * with each of them under the same identities, so a card under no stencil and
 * two cards of one mark are each marked where they were read from.
 */
import { computed, ref } from 'vue'
import { cardBlanks, cardFields, Deck as DeckView, Menu } from '@numen/ui'
import type { CardLanding, Point } from '@numen/ui'
import { ChevronDown } from '@lucide/vue'
import FileConflictPrompt from '../FileConflictPrompt.vue'
import type { DeckTabState } from './deck'
import { WORDS as words } from './words'

const props = defineProps<{ held: DeckTabState }>()

// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const { bands, choices, drawn, marks, saying, scheduled, shown, stencils } = props.held

/** What the grid draws against the cards it was handed. */
const wrong = computed(() => ({ at: marks.value.at, under: marks.value.under }))

/** The untyped values a card cut by that stencil is made with. */
const empty = (stencil: string) =>
  cardBlanks(
    cardFields(stencils.value.find((one) => one.name === stencil)?.fields ?? []),
  )

/**
 * The presets on offer. The defaults stand in a band of their own, so the line
 * between them and the notes says which is which.
 */
const offered = computed(() =>
  choices.value.map((one) => ({
    id: one.path,
    text: one.name,
    band: one.path === '' ? 'defaults' : 'presets',
  })),
)

/** Where the presets were asked for, and nothing while they are not. */
const asking = ref<{ at: Point; from: HTMLElement } | null>(null)

/** The line saying which preset schedules this deck opens the presets under it. */
const asks = (event: Event) => {
  const line = event.currentTarget
  if (!(line instanceof HTMLElement)) return
  const box = line.getBoundingClientRect()
  asking.value = { at: { x: box.left, y: box.bottom }, from: line }
}

const chose = (path: string) => {
  asking.value = null
  props.held.schedules(path)
}
</script>

<template>
  <div class="deck-tab">
    <FileConflictPrompt
      :saying="saying"
      :state="shown.state"
      :words="words"
      @keep="props.held.keep()"
      @take="props.held.take()"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <!-- Which preset schedules this deck, and the presets under it when it is
         asked. What the deck names and the vault does not hold is said here. -->
    <p class="deck-tab__scheduled">
      <span id="deck-scheduled" class="deck-tab__by">{{ words.scheduledBy }}</span>
      <button
        type="button"
        class="deck-tab__choice"
        aria-haspopup="menu"
        aria-labelledby="deck-scheduled"
        @click="asks"
      >
        {{ scheduled.name }}
        <ChevronDown class="deck-tab__icon" aria-hidden="true" />
      </button>
      <span v-if="scheduled.saying" role="status" class="deck-tab__wrong">
        {{ scheduled.saying }}
      </span>
    </p>

    <DeckView
      class="deck-tab__grid"
      :cards="drawn"
      :sections="bands"
      :stencils="stencils"
      :name="words.deck"
      :wrong="wrong"
      @add="(stencil: string, section: string | null) => props.held.adds(stencil, empty(stencil), section)"
      @remove="(id: string) => props.held.removes(id)"
      @move="(id: string, at: CardLanding) => props.held.moves(id, at)"
      @write="
        (id: string, field: string, nth: number, text: string) =>
          props.held.writes(id, field, nth, text)
      "
      @add-section="(name: string) => props.held.addsSection(name)"
      @rename-section="(id: string, name: string) => props.held.namesSection(id, name)"
      @remove-section="(id: string) => props.held.removesSection(id)"
    />

    <Menu
      v-if="asking"
      :items="offered"
      :at="asking.at"
      :from="asking.from"
      :current="scheduled.path"
      open
      opening="keyboard"
      :name="words.scheduledBy"
      @choose="chose"
      @dismiss="asking = null"
    />
  </div>
</template>

<style scoped>
/* The grid takes what the bands above it leave, and scrolls inside itself. */
.deck-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.deck-tab__grid {
  flex: 1;
  min-block-size: 0;
}

/* The line above the grid, which takes the height it needs and no more. */
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

/* A line of text carrying the mark that says it opens. */
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
  outline: var(--numen-ring-width) solid var(--numen-ring);
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

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
