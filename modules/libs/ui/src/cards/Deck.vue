<script setup lang="ts">
/**
 * A deck, edited: its cards as tiles, and a plus standing last.
 *
 * A tile is a strip it is carried by and removed from, and under it the card's
 * values, each in a box that is always open to typing. The field naming the
 * card stands first among them. What a card stands for is the caller's.
 */
import { computed, shallowRef, useId } from 'vue'
import Bar from './Bar.vue'
import Glyph from './Glyph.vue'
import Grown from './Grown.vue'
import Rule from './Rule.vue'
import { Button } from '../components/ui/button'
import {
  blanks,
  DECK_WORDS,
  declared,
  freeName,
  grid,
  oneLine,
  stepped,
  type Cut,
  type DeckWords,
  type Drawn,
  type Filled,
  type Landing,
  type Stood,
  type Tile,
  type Way,
} from './model'

const props = withDefaults(
  defineProps<{
    /** The cards, in the order they are drawn. */
    cards: readonly Drawn[]
    /** The stencils a card may be cut by. */
    cuts: readonly Cut[]
    /** What the grid is announced as. */
    name?: string
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { name: 'Deck', words: () => DECK_WORDS },
)

const emit = defineEmits<{
  /**
   * A card asked for, cut by a stencil, named by something nothing has taken.
   * The name is the first field's value and stands nowhere among the rest,
   * which stand empty.
   */
  (event: 'add', name: string, stencil: string, filled: readonly Filled[]): void
  (event: 'remove', id: string): void
  /** A card let go somewhere in the order: before another, or at the end. */
  (event: 'move', id: string, at: Landing): void
  /** One value of one card, as it now reads. The first of them names the card. */
  /**
   * One value of one card as it now reads. A card writing a field twice is
   * writing two values, and `nth` says which of them was typed in.
   */
  (event: 'write', id: string, field: string, nth: number, text: string): void
}>()

/** What this deck's boxes are named by, which is this deck's alone. */
const uid = useId()

const boxId = (tile: Tile, value: Stood): string =>
  `${uid}-${tile.id}-${encodeURIComponent(value.key)}`

/** The plus is showing which stencils a new card may be cut by. */
const asking = shallowRef(false)

/** The card under the pointer's hand, and where letting go would put it. */
const carried = shallowRef<string | null>(null)
const at = shallowRef<Landing>(null)

const shown = computed(() => grid(props.cards, props.cuts, carried.value))

const lift = (id: string, event: DragEvent): void => {
  carried.value = id
  event.dataTransfer?.setData('text/plain', id)
}

const over = (landing: Landing, event: DragEvent): void => {
  if (carried.value === null) return
  event.preventDefault()
  at.value = landing
}

/** The carry is over, and nothing was let go. */
const release = (): void => {
  carried.value = null
  at.value = null
}

/** A card let go on a place that takes it. */
const drop = (): void => {
  const held = carried.value
  const landing = at.value
  release()
  if (held !== null && landing !== held) emit('move', held, landing)
}

/** A card asked to go one place along the order, where there is a place that way. */
const step = (id: string, way: Way, press: KeyboardEvent): void => {
  const lands = stepped(props.cards.map((card) => card.id), id, way)
  if (lands === undefined) return
  press.preventDefault()
  emit('move', id, lands)
}

/**
 * A break struck in the box the card is named by. The name is written in a
 * heading, so the box holds one line; a break struck while a word is being
 * composed belongs to the composing.
 */
const breaking = (value: Stood, press: KeyboardEvent): void => {
  if (value.names && !press.isComposing) press.preventDefault()
}

/**
 * One value of one card as it now reads. The field the card is named by is
 * written in a heading, so what is typed into its box comes to one line.
 */
const write = (tile: Tile, value: Stood, text: string): void => {
  emit('write', tile.id, value.field, value.nth, value.names ? oneLine(text) : text)
}

const add = (cut: Cut): void => {
  asking.value = false
  emit(
    'add',
    freeName(props.cards.map((card) => card.name), props.words.cardStem),
    cut.name,
    blanks(declared(cut.fields).slice(1)),
  )
}
</script>

<template>
  <div
    class="deck numen bg-surface font-sans text-base text-ink"
    role="group"
    :aria-label="name"
    @dragover="over(null, $event)"
    @drop="drop"
  >
    <div class="deck__grid">
      <article
        v-for="tile in shown.tiles"
        :key="tile.id"
        class="deck__tile flex flex-col rounded-node bg-raised"
        :aria-label="tile.name"
        :aria-posinset="tile.at"
        :aria-setsize="shown.of"
        :data-card="tile.id"
        :data-carried="tile.carried || undefined"
        :data-before="tile.id === at || undefined"
        @dragover.stop="over(tile.id, $event)"
        @drop.stop="drop"
      >
        <Bar
          :carry="`${words.carry}: ${tile.name}`"
          @dragstart="lift(tile.id, $event)"
          @dragend="release"
          @step="(way, press) => step(tile.id, way, press)"
        >
          <template #deeds>
            <Button
              variant="ghost"
              size="icon-small"
              class="size-6"
              draggable="false"
              :aria-label="`${words.remove}: ${tile.name}`"
              @click="emit('remove', tile.id)"
            >
              <Glyph shows="bin" />
            </Button>
          </template>
        </Bar>

        <div class="deck__body flex flex-col">
          <!-- A name is exposed by nothing standing on a paragraph, so the
               text takes a role that carries one. -->
          <p v-if="!tile.named" class="deck__said truncate" role="group" :aria-label="words.name">
            {{ tile.name }}
          </p>

          <p v-if="!tile.known" class="deck__objects text-small text-alarm" role="alert">
            {{ words.unknown(tile.stencil) }}
          </p>

          <div
            v-for="value in tile.filled"
            :key="value.key"
            class="deck__value"
            :data-names="value.names || undefined"
            :data-twice="value.twice || undefined"
          >
            <Rule at="start">
              <label class="deck__field text-small text-hushed" :for="boxId(tile, value)">
                {{ value.field }}
              </label>
            </Rule>

            <Grown :text="value.text">
              <textarea
                :id="boxId(tile, value)"
                class="deck__written"
                :data-value="value.field"
                rows="1"
                :value="value.text"
                @keydown.enter="breaking(value, $event)"
                @input="write(tile, value, ($event.target as HTMLTextAreaElement).value)"
              ></textarea>
            </Grown>

            <p v-if="value.twice" class="deck__objects text-small text-alarm">{{ words.twice }}</p>
          </div>

          <p v-if="!tile.filled.length" class="deck__silence text-small text-hushed">
            {{ words.nothing }}
          </p>
        </div>
      </article>

      <article
        class="deck__tile deck__plus rounded-node"
        :aria-posinset="shown.plusAt"
        :aria-setsize="shown.of"
        :aria-label="words.add"
        data-plus
      >
        <!-- The plus says what it is for by standing alone in the middle. What
             it is called is read aloud and shown on hovering, and not beside it.
             What it opens takes its place, so it says nothing of being open. -->
        <Button
          v-if="!asking"
          variant="ghost"
          class="deck__ask"
          :aria-label="words.add"
          :title="words.add"
          @click="asking = true"
        >
          <Glyph shows="plus" />
        </Button>

        <div v-else class="deck__asking flex flex-col items-center">
          <p class="deck__silence text-small text-hushed">{{ words.cut }}</p>
          <div class="deck__cuts flex flex-wrap justify-center">
            <Button
              v-for="cut in cuts"
              :key="cut.name"
              variant="outline"
              size="small"
              :data-cut="cut.name"
              @click="add(cut)"
              >{{ cut.name }}</Button
            >
          </div>
        </div>
      </article>
    </div>
  </div>
</template>

<style scoped>
/* The deck is what scrolls. The grid inside it is as tall as its rows, which is
   what lets every row take the height of the tallest tile of the whole deck. */
.deck {
  --tile: 20rem;
  --gap: 0.75rem;
  --pad: 0.625rem;
  /* The line a carried card lands on. */
  --caret: 2px;

  padding: var(--numen-gutter);
  overflow: auto;
}

/*
 * Tiles stand in as many columns as fit at the narrowest one is drawn, and one
 * column where there is not the width for two. Every row is one flexible track,
 * so all of them come to the height of the fullest tile and no tile is drawn to
 * a size of its own.
 */
.deck__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(var(--tile), 100%), 1fr));
  grid-auto-rows: 1fr;
  gap: var(--gap);
}

.deck__tile {
  position: relative;
  min-inline-size: 0;
  border: var(--numen-stroke) solid var(--numen-node-border);
  overflow: hidden;
  overflow-wrap: anywhere;
}

/* The strip runs the whole width, and the body keeps the clearance. */
.deck__body {
  gap: var(--numen-inset);
  padding: var(--pad);
}

/*
 * A value is the room between its own rule and the next: the rule names it and
 * the box under it takes what is left, with nothing drawn around either.
 */
.deck__field {
  padding-inline: var(--box-pad-inline);
}

/* A rule divides the whole tile, so it runs to both edges of it. */
.deck__value > .rule {
  inline-size: auto;
  margin-inline: calc(-1 * var(--pad));
}

.deck__tile[data-carried] {
  opacity: 0.5;
}

.deck__tile[data-before]::before {
  content: '';
  position: absolute;
  inset-block: 0;
  inset-inline-start: calc(-1 * var(--gap) / 2);
  inline-size: var(--caret);
  background: var(--numen-ring);
}

/* The plus is an outline until it is pressed: it holds nothing yet. What it
   holds stands in the middle, which is where it was pressed. */
.deck__plus {
  --plus: 2.5rem;

  display: grid;
  place-items: center;
  min-block-size: 6rem;
  padding: var(--pad);
  border-style: dashed;
}

.deck__ask {
  block-size: auto;
  inline-size: auto;
  padding: var(--numen-inset);
}

.deck__ask :deep(.glyph) {
  inline-size: var(--plus);
  block-size: var(--plus);
  stroke-width: 1;
}

.deck__asking {
  gap: var(--numen-inset);
}

.deck__cuts {
  gap: var(--numen-inset);
}

/* A card whose stencil names no field is named by what it was handed. */
.deck__said {
  margin: 0;
}

/*
 * The box and the ground behind it are set the same text, in the same type, at
 * the same measure, and share one cell. The ground is what the cell is sized
 * by, so the box is exactly as tall as what it holds and never scrolls.
 */
.deck__written {
  inline-size: 100%;
  min-inline-size: 0;
}

.deck__objects,
.deck__silence {
  margin: 0;
}

.deck__silence {
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}
</style>
