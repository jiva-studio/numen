<script setup lang="ts">
/**
 * One card of a deck, as a tile: a strip it is dragged by and removed from, and
 * under it the card's values, each in a box that is always open to typing.
 *
 * Every field of the card model is placed under its own name and holds as many lines
 * as a person writes. What is wrong with the card is said under its strip, and
 * what is wrong with one value is said under that value.
 */
import { computed, useId } from 'vue'
import { ErrorMessage } from '../../error-message'
import { CardHeader } from '../../card-header'
import { RemoveButton } from '../../remove-button'
import { CardValue } from './card-value'
import { DECK_WORDS, type CardWords } from '../../../lib/deck'
import type { PlacedFieldValue, Tile } from '../../../lib/grid'
import { createSealedMap, type StepDirection } from '../../../lib/order'

const props = withDefaults(
  defineProps<{
    /** The card, laid out against the card model that cuts it. */
    tile: Tile
    /** What is wrong with this card, said under its heading. */
    wrong?: readonly string[]
    /** What is wrong with each of its values, under the field the value is in. */
    wrongUnder?: ReadonlyMap<string, readonly string[]>
    /** The words it is drawn with. */
    words?: CardWords
  }>(),
  { wrong: () => [], wrongUnder: () => createSealedMap(), words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'remove'): void
  /** The card taken up by the pointer, and let go again. */
  (event: 'lift', press: DragEvent): void
  (event: 'release'): void
  /** The card asked to go one place along the order. */
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
}>()

/** What this card's boxes are named by, which is this card's alone. */
const uid = useId()

/**
 * What the card is announced as. A card holds no name of its own: what it is
 * called is the line its first field evaluates to, and the file writes that.
 */
const called = computed(() => `${props.words.cardStem} ${props.tile.at}`)

/** What the strip says cut the card, and where nothing cut it, that nothing did. */
const cardModelTitle = computed(() => props.tile.cardModel ?? props.words.unknown(null))

const boxId = (value: PlacedFieldValue): string => `${uid}-${encodeURIComponent(value.key)}`

/** What is wrong with one value, said once, under the last box standing for its field. */
const getProblems = (value: PlacedFieldValue): readonly string[] =>
  value.isLast ? (props.wrongUnder.get(value.field) ?? []) : []
</script>

<template>
  <article
    class="card rounded-node bg-raised flex flex-col"
    :aria-label="called"
    :aria-posinset="tile.at"
    :aria-setsize="tile.of"
    :data-card="tile.id"
    :data-section="tile.section ?? undefined"
    :data-dragged="tile.isDragged || undefined"
  >
    <CardHeader
      :drag="`${words.drag}: ${called}`"
      @dragstart="emit('lift', $event)"
      @dragend="emit('release')"
      @step="(direction, press) => emit('step', direction, press)"
    >
      <!-- A deck holds cards cut by more than one cardModel, so the strip says
           which cut this one, and where nothing did, that nothing did. A name
           is exposed by nothing standing on a paragraph, so the text takes a
           role that carries one. -->
      <p
        class="card__cut text-small text-hushed truncate"
        role="group"
        :aria-label="words.cut"
        data-cut-of
      >
        {{ cardModelTitle }}
      </p>

      <template #actions>
        <RemoveButton :label="`${words.remove}: ${called}`" @press="emit('remove')" />
      </template>
    </CardHeader>

    <div class="card__body flex flex-col">
      <!-- A card is waiting for a cardModel only where it names one. -->
      <ErrorMessage
        v-if="!tile.isKnown && tile.cardModel !== null"
        class="card__objections"
        role="alert"
        :said="words.unknown(tile.cardModel)"
      />

      <ErrorMessage
        v-if="wrong.length"
        class="card__objections"
        data-wrong
        :said="wrong"
        :label="words.wrong"
      />

      <CardValue
        v-for="value in tile.filled"
        :key="value.key"
        :card="tile.id"
        :value="value"
        :box-id="boxId(value)"
        :wrong="getProblems(value)"
        :words="words"
      />

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

.card__body {
  flex: 1;
  gap: var(--numen-inset);
  padding: var(--numen-box-air);
}

.card[data-dragged] {
  opacity: 0.5;
}

.card__cut {
  position: absolute;
  inset-inline: 2rem;
  inset-block-start: 50%;
  translate: 0 -50%;
  margin: 0;
  text-align: center;
  pointer-events: none;
}

.card__objections {
  margin: 0;
  padding-inline: var(--box-pad-inline);
}

/* A card holding nothing says so in the middle of the room it is given. */
.card__silence {
  margin: auto;
  padding-inline: var(--box-pad-inline);
  text-align: center;
}
</style>
