<script setup lang="ts">
/**
 * One card of a deck, as a tile: a strip it is carried by and removed from, and
 * under it the card's values, each in a box that is always open to typing.
 *
 * The field naming the card stands first among them and holds one line, because
 * the name is written in a heading. What is wrong with the card is said under
 * that heading, and what is wrong with one value is said under that value.
 */
import { useId } from 'vue'
import Bar from './Bar.vue'
import Glyph from './Glyph.vue'
import Grown from './Grown.vue'
import Rule from '../rule/Rule.vue'
import { Button } from '../components/ui/button'
import { DECK_WORDS, oneLine, type DeckWords, type Stood, type Tile, type Way } from './model'

const props = withDefaults(
  defineProps<{
    /** The card, laid out against the stencil that cuts it. */
    tile: Tile
    /** How many cards the deck holds, and this one's place among them. */
    of?: number
    /** What is wrong with this card, said under its heading. */
    wrong?: readonly string[]
    /** What is wrong with each of its values, under the field the value is in. */
    wrongUnder?: ReadonlyMap<string, readonly string[]>
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { of: 1, wrong: () => [], wrongUnder: () => new Map(), words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'remove'): void
  /** The card taken up by the pointer, and let go again. */
  (event: 'lift', press: DragEvent): void
  (event: 'release'): void
  /** The card asked to go one place along the order. */
  (event: 'step', way: Way, press: KeyboardEvent): void
  /**
   * One value as it now reads. A card writing a field twice is writing two
   * values, and `nth` says which of them was typed in.
   */
  (event: 'write', field: string, nth: number, text: string): void
}>()

/** What this card's boxes are named by, which is this card's alone. */
const uid = useId()

const boxId = (value: Stood): string => `${uid}-${encodeURIComponent(value.key)}`

/**
 * What is wrong with one value, said once, under the last of the boxes standing
 * for that field, which is the one a card writing it twice put there.
 */
const wrongIn = (value: Stood): readonly string[] => {
  const under = props.tile.filled.filter((each) => each.field === value.field)
  if (under[under.length - 1] !== value) return []
  return props.wrongUnder.get(value.field) ?? []
}

/**
 * A break struck in the box the card is named by. The name is written in a
 * heading, so the box holds one line; a break struck while a word is being
 * composed belongs to the composing.
 */
const breaking = (value: Stood, press: KeyboardEvent): void => {
  if (value.names && !press.isComposing) press.preventDefault()
}

const write = (value: Stood, text: string): void => {
  emit('write', value.field, value.nth, value.names ? oneLine(text) : text)
}
</script>

<template>
  <article
    class="card flex flex-col rounded-node bg-raised"
    :aria-label="tile.name"
    :aria-posinset="tile.at"
    :aria-setsize="of"
    :data-card="tile.id"
    :data-carried="tile.carried || undefined"
  >
    <Bar
      :carry="`${words.carry}: ${tile.name}`"
      @dragstart="emit('lift', $event)"
      @dragend="emit('release')"
      @step="(way, press) => emit('step', way, press)"
    >
      <template #deeds>
        <Button
          variant="ghost"
          size="icon-small"
          class="size-6"
          draggable="false"
          :aria-label="`${words.remove}: ${tile.name}`"
          @click="emit('remove')"
        >
          <Glyph shows="bin" />
        </Button>
      </template>
    </Bar>

    <div class="card__body flex flex-col">
      <!-- A name is exposed by nothing standing on a paragraph, so the text
           takes a role that carries one. -->
      <p v-if="!tile.named" class="card__said truncate" role="group" :aria-label="words.name">
        {{ tile.name }}
      </p>

      <p v-if="!tile.known" class="card__objects text-small text-alarm" role="alert">
        {{ words.unknown(tile.stencil) }}
      </p>

      <ul
        v-if="wrong.length"
        class="card__objects text-small text-alarm"
        :aria-label="words.wrong"
        data-wrong
      >
        <li v-for="(text, at) in wrong" :key="at">{{ text }}</li>
      </ul>

      <div
        v-for="value in tile.filled"
        :key="value.key"
        class="card__value"
        :data-names="value.names || undefined"
        :data-twice="value.twice || undefined"
      >
        <Rule at="start">
          <label class="card__field text-small text-hushed" :for="boxId(value)">
            {{ value.field }}
          </label>
        </Rule>

        <Grown :text="value.text">
          <textarea
            :id="boxId(value)"
            class="card__written"
            :data-value="value.field"
            rows="1"
            :value="value.text"
            @keydown.enter="breaking(value, $event)"
            @input="write(value, ($event.target as HTMLTextAreaElement).value)"
          ></textarea>
        </Grown>

        <p v-if="value.twice" class="card__objects text-small text-alarm">{{ words.twice }}</p>

        <ul
          v-if="wrongIn(value).length"
          class="card__objects text-small text-alarm"
          :aria-label="words.wrong"
          :data-wrong-value="value.field"
        >
          <li v-for="(text, at) in wrongIn(value)" :key="at">{{ text }}</li>
        </ul>
      </div>

      <p v-if="!tile.filled.length" class="card__silence text-small text-hushed">
        {{ words.nothing }}
      </p>
    </div>
  </article>
</template>

<style scoped>
.card {
  --pad: 0.625rem;
  /* The name of a value, the box it is typed in and what is said to be wrong
     with it all stand over one edge. */
  --box-pad-inline: 0.625rem;

  position: relative;
  min-inline-size: 0;
  border: var(--numen-stroke) solid var(--numen-node-border);
  overflow: hidden;
  overflow-wrap: anywhere;
}

/* The strip runs the whole width, and the body keeps the clearance. */
.card__body {
  gap: var(--numen-inset);
  padding: var(--pad);
}

/* A rule divides the whole tile, so it runs to both edges of it. */
.card__value > .rule {
  inline-size: auto;
  margin-inline: calc(-1 * var(--pad));
}

.card[data-carried] {
  opacity: 0.5;
}

/* A card whose stencil names no field is named by what it was handed. */
.card__said {
  margin: 0;
  padding-inline: var(--box-pad-inline);
}

/*
 * The box and the ground behind it are set the same text, in the same type, at
 * the same measure, and share one cell. The ground is what the cell is sized by,
 * so the box is exactly as tall as what it holds and never scrolls.
 */
.card__written {
  inline-size: 100%;
  min-inline-size: 0;
}

/* Everything the body holds stands over one edge: the card's own name, what is
   wrong with it, what each value is called, and the box the value is typed in. */
.card__objects,
.card__silence {
  margin: 0;
  padding-inline: var(--box-pad-inline);
}

.card__silence {
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}
</style>
