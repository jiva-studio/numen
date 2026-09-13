<script setup lang="ts">
/**
 * The stencils a new card may be cut by, offered at a pressed plus: one button
 * for each, under the question they answer.
 */
import { Button } from '@/shared/ui/button'
import { DECK_WORDS, type DeckWords } from '../../../lib/deck'
import type { Stencil } from '../../../lib/card'

withDefaults(
  defineProps<{
    /** The stencils a card may be cut by. */
    stencils: readonly Stencil[]
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'choose', stencil: Stencil): void
}>()
</script>

<template>
  <div class="deck__asking flex flex-col items-center">
    <p class="deck__silence caps-numen text-small text-hushed">{{ words.cut }}</p>
    <div class="deck__cuts flex flex-wrap justify-center">
      <Button
        v-for="stencil in stencils"
        :key="stencil.name"
        variant="outline"
        size="small"
        :data-cut="stencil.name"
        @click="emit('choose', stencil)"
      >
        {{ stencil.name }}
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
