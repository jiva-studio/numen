<script setup lang="ts">
/**
 * The menu that moves an entry of a bar, drawn once for the window at the pointer.
 */
import { Menu, type MenuItem, type Position } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  items: readonly MenuItem[]
  at: Position
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  choose: [id: string]
  dismiss: []
}>()
</script>

<template>
  <Menu
    :items="items"
    :at="at"
    is-open
    name="Move"
    @choose="emit('choose', $event)"
    @dismiss="emit('dismiss')"
  />
</template>
