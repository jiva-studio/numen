<script setup lang="ts">
/**
 * One card of a deck, as a tile: a strip it is carried by and removed from, and
 * under it the card's values, each in a box that is always open to typing.
 *
 * Every field of the stencil stands under its own name and holds as many lines
 * as a person writes. What is wrong with the card is said under its strip, and
 * what is wrong with one value is said under that value.
 */
import { computed, useId } from 'vue'
import ErrorMessage from './ErrorMessage.vue'
import CardHeader from './CardHeader.vue'
import RemoveButton from './RemoveButton.vue'
import AutosizeTextarea from './AutosizeTextarea.vue'
import Divider from '../divider/Divider.vue'
import { DECK_WORDS, sealed, type CardWords, type PlacedFieldValue, type Tile } from './deck'
import type { StepDirection } from './order'

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
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
  /**
   * One value as it now reads. A card writing a field twice is writing two
   * values, of which `nth` says which was typed in.
   */
  (event: 'write', field: string, nth: number, text: string): void
}>()

/** What this card's boxes are named by, which is this card's alone. */
const uid = useId()

/**
 * What the card is announced as. A card holds no name of its own: what it is
 * called is the line its first field comes to, and the file writes that.
 */
const called = computed(() => `${props.words.cardStem} ${props.tile.at}`)

/** What the strip says cut the card, and where nothing cut it, that nothing did. */
const cut = computed(() => props.tile.stencil ?? props.words.unknown(null))

const boxId = (value: PlacedFieldValue): string => `${uid}-${encodeURIComponent(value.key)}`

/** What is wrong with one value, said once, under the last box standing for its field. */
const wrongIn = (value: PlacedFieldValue): readonly string[] =>
  value.last ? (props.wrongUnder.get(value.field) ?? []) : []
</script>

<template>
  <article
    class="card flex flex-col rounded-node bg-raised"
    :aria-label="called"
    :aria-posinset="tile.at"
    :aria-setsize="tile.of"
    :data-card="tile.id"
    :data-section="tile.section ?? undefined"
    :data-carried="tile.carried || undefined"
  >
    <CardHeader
      :carry="`${words.carry}: ${called}`"
      @dragstart="emit('lift', $event)"
      @dragend="emit('release')"
      @step="(direction, press) => emit('step', direction, press)"
    >
      <!-- A deck holds cards cut by more than one stencil, so the strip says
           which cut this one, and where nothing did, that nothing did. A name
           is exposed by nothing standing on a paragraph, so the text takes a
           role that carries one. -->
      <p
        class="card__cut truncate text-small text-hushed"
        role="group"
        :aria-label="words.cut"
        data-cut-of
      >
        {{ cut }}
      </p>

      <template #deeds>
        <RemoveButton :label="`${words.remove}: ${called}`" @press="emit('remove')" />
      </template>
    </CardHeader>

    <div class="card__body flex flex-col">
      <!-- A card is waiting for a stencil only where it names one. -->
      <ErrorMessage
        v-if="!tile.known && tile.stencil !== null"
        class="card__objects"
        role="alert"
        :said="words.unknown(tile.stencil)"
      />

      <ErrorMessage
        v-if="wrong.length"
        class="card__objects"
        data-wrong
        :said="wrong"
        :label="words.wrong"
      />

      <div v-for="value in tile.filled" :key="value.key" class="card__value">
        <Divider at="start">
          <label
            v-if="value.declared"
            class="card__field text-small text-hushed"
            :for="boxId(value)"
          >
            {{ value.field }}
          </label>
          <span v-else class="card__field text-small text-hushed">{{ value.field }}</span>
        </Divider>

        <AutosizeTextarea
          v-if="value.declared"
          :id="boxId(value)"
          :text="value.text"
          :data-value="value.field"
          @write="(text: string) => emit('write', value.field, value.nth, text)"
        />

        <!-- No stencil names a slot for this value, so what was written is read
             where it would be typed. -->
        <p
          v-else
          class="card__wrote"
          role="group"
          :aria-label="value.field"
          :data-wrote="value.field"
        >{{ value.text }}</p>

        <ErrorMessage
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
  border: var(--numen-stroke) solid var(--numen-rule);
  overflow: hidden;
  overflow-wrap: anywhere;
}

/* The strip runs the whole width, and the body keeps the clearance. The body
   takes the rest of the tile, so what stands in place of values has the room. */
.card__body {
  flex: 1;
  gap: var(--numen-inset);
  padding: var(--numen-box-air);
}

/* A divider divides the whole tile, so it runs to both edges of it. */
.card__value > .divider {
  inline-size: auto;
  margin-inline: calc(-1 * var(--numen-box-air));
}

.card[data-carried] {
  opacity: 0.5;
}

/* What cut the card stands in the middle of the strip itself, and keeps clear
   of the button at its end. */
.card__cut {
  position: absolute;
  inset-inline: 2rem;
  inset-block-start: 50%;
  translate: 0 -50%;
  margin: 0;
  text-align: center;
  pointer-events: none;
}

/* Everything the body holds stands over one edge: what is wrong with the card,
   what each value is called, and the value itself. */
.card__objects,
.card__wrote {
  margin: 0;
  padding-inline: var(--box-pad-inline);
}

/* A value no stencil names is read where it would be typed, and keeps the
   breaks it was written with. */
.card__wrote {
  padding-block: var(--box-air, 0.5rem);
  white-space: pre-wrap;
}

/* A card holding nothing says so in the middle of the room it is given. */
.card__silence {
  margin: auto;
  padding-inline: var(--box-pad-inline);
  text-align: center;
}
</style>
