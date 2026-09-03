<script setup lang="ts">
/**
 * The band a tab says something in, and the answers that ride with it.
 *
 * What it is is the caller's: a refusal is an alert and a question a person is
 * being put is a status, and the role comes down with the rest of the
 * attributes.
 */
defineProps<{
  /** Whether answers stand on the line the sentence is on. */
  answering?: boolean
}>()
</script>

<template>
  <p class="warning" :class="{ 'warning--answering': answering }">
    <slot />
  </p>
</template>
