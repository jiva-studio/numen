<script setup lang="ts">
/**
 * One value of a card, edited in an auto-sizing text box, with the problems with
 * the value shown underneath.
 */
import { inject } from 'vue'
import { ErrorMessage } from '../../../error-message'
import { AutosizeTextarea } from '../../../autosize-textarea'
import { Divider } from '../../../divider'
import { DECK_WORDS, type CardWords } from '../../../../lib/deck'
import { DECK_CONTEXT } from '../../../../model/context'
import type { PlacedFieldValue } from '../../../../lib/grid'

const props = withDefaults(
  defineProps<{
    /** The card the value belongs to, by its identifier. */
    card: string
    /** The value, laid out against the card model that cuts the card. */
    value: PlacedFieldValue
    /** What the box is named by, which its label points at. */
    boxId: string
    /** What is wrong with this value. */
    wrong?: readonly string[]
    /** The words it is drawn with. */
    words?: CardWords
  }>(),
  { wrong: () => [], words: () => DECK_WORDS },
)

const deck = inject(DECK_CONTEXT)
if (!deck) throw new Error('A card value is drawn outside a deck')

/** Reports the value's new text to the deck, with the card and the place the value is in. */
const onWrite = (text: string): void =>
  deck.write(props.card, { field: props.value.field, index: props.value.index }, text)
</script>

<template>
  <div class="card__value">
    <Divider at="start">
      <label v-if="value.isDeclared" class="card__field text-small text-hushed" :for="boxId">
        {{ value.field }}
      </label>
      <span v-else class="card__field text-small text-hushed">{{ value.field }}</span>
    </Divider>

    <AutosizeTextarea
      v-if="value.isDeclared"
      :id="boxId"
      :text="value.text"
      :data-value="value.field"
      @write="onWrite"
    />

    <!-- No cardModel names a slot for this value, so what was written is read
         where it would be typed. -->
    <p v-else class="card__wrote" role="group" :aria-label="value.field" :data-wrote="value.field">
      {{ value.text }}
    </p>

    <ErrorMessage
      v-if="wrong.length"
      class="card__objections"
      :data-wrong-value="value.field"
      :said="wrong"
      :label="words.wrong"
    />
  </div>
</template>

<style scoped>
/* A divider divides the whole tile, so it runs to both edges of it. */
.card__value > .divider {
  inline-size: auto;
  margin-inline: calc(-1 * var(--numen-box-air));
}

/* What a value is called, the value itself and what is wrong with it all stand
   over one edge. */
.card__objections,
.card__wrote {
  margin: 0;
  padding-inline: var(--box-pad-inline);
}

/* A value no cardModel names is read where it would be typed, and keeps the
   breaks it was written with. */
.card__wrote {
  padding-block: var(--box-air);
  white-space: pre-wrap;
}
</style>
