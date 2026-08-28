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
import Amiss from './Amiss.vue'
import Bar from './Bar.vue'
import Deed from './Deed.vue'
import Grown from './Grown.vue'
import Rule from '../rule/Rule.vue'
import { DECK_WORDS, sealed, type CardWords, type Stood, type Tile } from './deck'
import { oneLine, type Way } from './order'

const props = withDefaults(
  defineProps<{
    /** The card, laid out against the stencil that cuts it. */
    tile: Tile
    /** What is wrong with this card, said under its heading. */
    wrong?: readonly string[]
    /** What is wrong with each of its values, under the field the value is in. */
    wrongUnder?: ReadonlyMap<string, readonly string[]>
    /** The words it is drawn with. */
    words?: CardWords
  }>(),
  { wrong: () => [], wrongUnder: () => sealed(), words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'remove'): void
  /** The card taken up by the pointer, and let go again. */
  (event: 'lift', press: DragEvent): void
  (event: 'release'): void
  /** The card asked to go one place along the order. */
  (event: 'step', way: Way, press: KeyboardEvent): void
  /**
   * One value as it now reads. `names` says the card's name was typed in, and
   * a card writing a field twice is writing two values, of which `nth` says
   * which was typed in.
   */
  (event: 'write', field: string, nth: number, names: boolean, text: string): void
}>()

/** What this card's boxes are named by, which is this card's alone. */
const uid = useId()

const boxId = (value: Stood): string => `${uid}-${encodeURIComponent(value.key)}`

/** What is wrong with one value, said once, under the last box standing for its field. */
const wrongIn = (value: Stood): readonly string[] =>
  value.last ? (props.wrongUnder.get(value.field) ?? []) : []

/**
 * A break struck in the box the card is named by. The name is written in a
 * heading, so the box holds one line; a break struck while a word is being
 * composed belongs to the composing.
 */
const breaking = (value: Stood, press: KeyboardEvent): void => {
  if (value.names && !press.isComposing) press.preventDefault()
}

const write = (value: Stood, text: string): void => {
  emit('write', value.field, value.nth, value.names, value.names ? oneLine(text) : text)
}
</script>

<template>
  <article
    class="card flex flex-col rounded-node bg-raised"
    :aria-label="tile.name"
    :aria-posinset="tile.at"
    :aria-setsize="tile.of"
    :data-card="tile.id"
    :data-carried="tile.carried || undefined"
  >
    <Bar
      :carry="`${words.carry}: ${tile.name}`"
      @dragstart="emit('lift', $event)"
      @dragend="emit('release')"
      @step="(way, press) => emit('step', way, press)"
    >
      <!-- A deck holds cards cut by more than one stencil, so the strip says
           which cut this one. A name is exposed by nothing standing on a
           paragraph, so the text takes a role that carries one. -->
      <p
        v-if="tile.stencil !== null"
        class="card__cut truncate text-small text-hushed"
        role="group"
        :aria-label="words.cut"
        data-cut-of
      >
        {{ tile.stencil }}
      </p>

      <template #deeds>
        <Deed :label="`${words.remove}: ${tile.name}`" @press="emit('remove')" />
      </template>
    </Bar>

    <div class="card__body flex flex-col">
      <!-- A name is exposed by nothing standing on a paragraph, so the text
           takes a role that carries one. -->
      <p v-if="!tile.named" class="card__said truncate" role="group" :aria-label="words.name">
        {{ tile.name }}
      </p>

      <Amiss
        v-if="!tile.known"
        class="card__objects"
        role="alert"
        :said="words.unknown(tile.stencil)"
      />

      <Amiss
        v-if="wrong.length"
        class="card__objects"
        data-wrong
        :said="wrong"
        :label="words.wrong"
      />

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

        <Grown
          :id="boxId(value)"
          :text="value.text"
          :data-value="value.field"
          @keydown.enter="breaking(value, $event)"
          @write="(text: string) => write(value, text)"
        />

        <Amiss v-if="value.twice" class="card__objects" :said="words.twice" />

        <Amiss
          v-if="wrongIn(value).length"
          class="card__objects"
          :data-wrong-value="value.field"
          :said="wrongIn(value)"
          :label="words.wrong"
        />
      </div>

      <p v-if="!tile.filled.length" class="card__silence caps-numen text-small text-hushed">
        {{ words.nothing }}
      </p>
    </div>
  </article>
</template>

<style scoped>
.card {
  /* The name of a value, the box it is typed in and what is said to be wrong
     with it all stand over one edge. */
  --box-pad-inline: var(--numen-box-air);

  position: relative;
  min-inline-size: 0;
  border: var(--numen-stroke) solid var(--numen-node-border);
  overflow: hidden;
  overflow-wrap: anywhere;
}

/* The strip runs the whole width, and the body keeps the clearance. */
.card__body {
  gap: var(--numen-inset);
  padding: var(--numen-box-air);
}

/* A rule divides the whole tile, so it runs to both edges of it. */
.card__value > .rule {
  inline-size: auto;
  margin-inline: calc(-1 * var(--numen-box-air));
}

.card[data-carried] {
  opacity: 0.5;
}

/* What cut the card stands in the middle of the strip itself, and keeps clear
   of the deed at its end. */
.card__cut {
  position: absolute;
  inset-inline: 2rem;
  inset-block-start: 50%;
  translate: 0 -50%;
  margin: 0;
  text-align: center;
  pointer-events: none;
}

/* A card whose stencil names no field is named by what it was handed. */
.card__said {
  margin: 0;
  padding-inline: var(--box-pad-inline);
}

/* Everything the body holds stands over one edge: the card's own name, what is
   wrong with it, what each value is called, and the box the value is typed in. */
.card__objects,
.card__silence {
  margin: 0;
  padding-inline: var(--box-pad-inline);
}
</style>
