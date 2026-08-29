<script setup lang="ts">
/**
 * One card, open to be put right where it was met.
 *
 * A card written badly is the one thing a person wants to fix in the middle of
 * answering, and the boxes it is fixed in are the library's own — the same ones
 * the editor's deck is written in.
 */
import { computed } from 'vue'
import { Card } from '@numen/ui'
import type { Stood, Tile } from '@numen/ui'
import type { Held } from './core'

const props = defineProps<{
  /** What the card is known by, and where it stands. */
  card: string
  section: string
  /** Its values, in the order its stencil asks for them. */
  values: readonly Held[]
}>()

const emit = defineEmits<{ (event: 'write', field: string, text: string): void }>()

/** The values as the boxes take them. */
const filled = computed<readonly Stood[]>(() =>
  props.values.map((one, at) => ({
    field: one.field,
    text: one.text,
    at: at + 1,
    key: `${one.field}/0`,
    nth: 1,
    last: true,
    // Every slot here is one the stencil names: the read filled them from it.
    declared: true,
  })),
)

const tile = computed<Tile>(() => ({
  id: props.card,
  section: props.section || null,
  stencil: null,
  filled: filled.value,
  at: 1,
  of: 1,
  known: true,
  carried: false,
}))
</script>

<template>
  <section class="editing">
    <Card :tile="tile" @write="(field, _nth, text) => emit('write', field, text)" />
  </section>
</template>

<style scoped>
/* The boxes are as tall as the fields they hold, and no taller. A card of many
   fields scrolls inside what the frame gave it. */
.editing {
  display: flex;
  flex-direction: column;
  max-block-size: 100%;
  overflow-y: auto;
}
</style>
