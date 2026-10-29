<script setup lang="ts">
/**
 * Error text displayed next to the element it concerns: a paragraph for one
 * message and a list for several.
 *
 * The text is rendered in a single element, so an `id` passed to this component
 * lands on the element that a control references in `aria-describedby`, and so
 * does a role.
 */
defineProps<{
  /** What is wrong: one thing, or each of several. */
  said: string | readonly string[]
  /** What a list of several is called to a reader. */
  label?: string
}>()
</script>

<template>
  <p v-if="typeof said === 'string'" class="error-message text-small text-alarm">{{ said }}</p>

  <ul v-else class="error-message text-small text-alarm" :aria-label="label">
    <li v-for="(text, at) in said" :key="at">{{ text }}</li>
  </ul>
</template>

<style scoped>
.error-message {
  flex-basis: 100%;
  margin: 0;
  padding-inline-start: 0;
  list-style: none;
  /* A name with nothing in it to break at is broken where the line ends. */
  overflow-wrap: anywhere;
}
</style>
