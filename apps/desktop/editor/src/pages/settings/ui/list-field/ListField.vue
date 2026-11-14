<script setup lang="ts">
/**
 * The items held by a plugin setting of type list, one row each with a remove action, and an input
 * that adds one.
 */

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  items: readonly string[]
  labelledBy: string
  describedBy?: string | undefined
  words: {
    readonly listAdd: string
    readonly listNew: string
    readonly listRemove: (item: string) => string
    readonly listItemTwice: (item: string) => string
  }
}>()

defineEmits<{
  add: [item: string]
  remove: [item: string]
}>()
</script>

<template>
  <div role="group" :aria-labelledby="labelledBy" :aria-describedby="describedBy" />
</template>
