<script setup lang="ts">
/**
 * The card models a new card may be cut by, offered at a pressed plus: one button
 * for each, under the question they answer.
 */
import { Button } from '@numen/ui'
import { DECK_WORDS, type DeckWords } from '../../../lib/deck'
import type { CardModel } from '../../../lib/card'

withDefaults(
  defineProps<{
    /** The card models a card may be cut by. */
    cardModels: readonly CardModel[]
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'choose', cardModel: CardModel): void
}>()
</script>

<template>
  <div class="deck__asking flex flex-col items-center">
    <p class="deck__silence caps-numen text-small text-hushed">{{ words.cut }}</p>
    <div class="deck__cuts flex flex-wrap justify-center">
      <Button
        v-for="cardModel in cardModels"
        :key="cardModel.name"
        variant="outline"
        size="small"
        :data-cut="cardModel.name"
        @click="emit('choose', cardModel)"
      >
        {{ cardModel.name }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.deck__asking {
  gap: var(--numen-inset);
}

.deck__cuts {
  gap: var(--numen-inset);
}

.deck__silence {
  margin: 0;
}
</style>
