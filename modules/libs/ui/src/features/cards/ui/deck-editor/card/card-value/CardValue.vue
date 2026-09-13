<script setup lang="ts">
/**
 * One value of a card: the name of the field it stands in, the box it is typed
 * in, and what is wrong with it said underneath.
 */
import { ErrorMessage } from '../../../error-message'
import { AutosizeTextarea } from '../../../autosize-textarea'
import { Divider } from '../../../divider'
import { DECK_WORDS, type CardWords } from '../../../../lib/deck'
import type { PlacedFieldValue } from '../../../../lib/grid'

withDefaults(
  defineProps<{
    /** The value, laid out against the stencil that cuts the card. */
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

const emit = defineEmits<{
  /** The value as it now reads. */
  (event: 'write', text: string): void
}>()
</script>

<template>
  <div class="card__value">
    <Divider at="start">
      <label v-if="value.declared" class="card__field text-small text-hushed" :for="boxId">
        {{ value.field }}
      </label>
      <span v-else class="card__field text-small text-hushed">{{ value.field }}</span>
    </Divider>

    <AutosizeTextarea
      v-if="value.declared"
      :id="boxId"
      :text="value.text"
      :data-value="value.field"
      @write="(text: string) => emit('write', text)"
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

/* A value no stencil names is read where it would be typed, and keeps the
   breaks it was written with. */
.card__wrote {
  padding-block: var(--box-air, 0.5rem);
  white-space: pre-wrap;
}
</style>
