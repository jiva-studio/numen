<script setup lang="ts">
/**
 * How a card came back, said by a person: the four, or the one that turns the
 * card over.
 *
 * The key stands in front of the word. A person answering from the keyboard
 * reads down the row of keys, and one answering with the pointer reads the
 * words either way.
 */
import { Button, KeyCap } from '@numen/ui'
import { called, said } from './core'
import type { Said } from './core'

defineProps<{
  /** Whether the answer is being shown, which is when the four stand. */
  shown: boolean
}>()

defineEmits<{
  (event: 'show'): void
  (event: 'answer', how: Said): void
}>()
</script>

<template>
  <footer class="answers">
    <template v-if="shown">
      <Button
        v-for="(how, i) in said"
        :key="how"
        variant="outline"
        class="answers__one"
        :class="`answers__one--${how}`"
        @click="$emit('answer', how)"
      >
        <KeyCap :keys="{ marks: [], letter: String(i + 1) }" />
        {{ called[how] }}
      </Button>
    </template>
    <Button v-else variant="outline" class="answers__one" @click="$emit('show')">
      <KeyCap :keys="{ marks: [], letter: 'space' }" />
      Show the answer
    </Button>
  </footer>
</template>

<style scoped>
.answers {
  display: flex;
  flex: none;
  gap: var(--numen-inset);
}

/* An answer is a target a person hits without looking, so it takes the whole
   width it can and stands taller than a button in a row of controls. */
.answers__one {
  flex: 1;
  block-size: auto;
  padding-block: var(--numen-inset-wide);
}

/* The answer that says a card was lost is the one worth telling apart at a
   glance, because it is the one a person reaches for without reading. */
.answers__one--again {
  border-color: var(--numen-alarm);
  color: var(--numen-alarm);
}
</style>
