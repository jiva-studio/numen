<script setup lang="ts">
import { computed, useId } from 'vue'
import { ErrorMessage } from '../../error-message'
import { CardHeader } from '../../card-header'
import { RemoveButton } from '../../remove-button'
import { CardValue } from './card-value'
import { useCardModelTitle } from '../../../../model/cards/useCardModelTitle'
import { DECK_WORDS, type CardWords } from '../../../../../lib/cards'
import type { PlacedFieldValue, Tile } from '../../../../../lib/cards'
import { getSealedMap, type StepDirection } from '../../../../../lib/cards'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    tile: Tile
    wrong?: readonly string[]
    wrongUnder?: ReadonlyMap<string, readonly string[]>
    words?: CardWords
  }>(),
  { wrong: () => [], wrongUnder: () => getSealedMap(), words: () => DECK_WORDS },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'remove'): void
  (event: 'drag-start', press: DragEvent): void
  (event: 'release'): void
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
}>()

/* --------------------------------- State ---------------------------------- */
const uid = useId()

const called = computed(() => `${props.words.cardStem} ${props.tile.at}`)

const cardModelTitle = useCardModelTitle(
  () => props.tile.cardModel,
  () => props.words.unknown,
)

/* -------------------------------- Helpers --------------------------------- */
const boxId = (value: PlacedFieldValue): string => `${uid}-${encodeURIComponent(value.key)}`

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
      @dragstart="emit('drag-start', $event)"
      @dragend="emit('release')"
      @step="(direction, press) => emit('step', direction, press)"
    >
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
  --box-air: 0.5rem;
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
  --cut-inset-inline: 2rem;
  position: absolute;
  inset-inline: var(--cut-inset-inline);
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

.card__silence {
  margin: auto;
  padding-inline: var(--box-pad-inline);
  text-align: center;
}
</style>
