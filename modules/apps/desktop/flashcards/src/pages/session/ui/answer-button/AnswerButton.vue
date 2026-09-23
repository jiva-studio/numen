<script setup lang="ts">
/**
 * One of the four answers: the number it is pressed with, the word it is known
 * by, and how long it would leave the card.
 */
import { Button, KeyCap } from '@numen/ui'

import { called, getTimeAhead } from '@/entities/card'
import type { Grade, Intervals } from '@/entities/card'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  how: Grade
  /** The number the answer is pressed with. */
  at: number
  /** Where each of the four would leave the card. */
  ahead: Intervals | null
}>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  (event: 'answer'): void
}>()
</script>

<template>
  <Button variant="outline" @click="$emit('answer')">
    <KeyCap :keys="{ icons: [], letter: String(at) }" />
    {{ called[how] }}
    <!-- What the answer does to the card, said where the answer is chosen: a
         person picking between the four is picking between these. It is read off
         the screen and not out of the button's own name, which is the word a
         person means to press. -->
    <span v-if="ahead" class="answer__ahead" aria-hidden="true">
      {{ getTimeAhead(ahead[how]) }}
    </span>
  </Button>
</template>

<style scoped>
/* What the answer does, said quietly beside it: it is read once, when a person
   is learning what the four mean, and glanced at after that. */
.answer__ahead {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
}
</style>
