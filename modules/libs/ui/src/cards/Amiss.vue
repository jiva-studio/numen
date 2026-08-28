<script setup lang="ts">
/**
 * What is wrong with something, said under it: one thing in a paragraph, and
 * several in a list.
 *
 * Whatever it is said in is the one element, so an `id` handed to this lands on
 * what a box beside it names in `aria-describedby`, and so does a role.
 */
defineProps<{
  /** What is wrong: one thing, or each of several. */
  said: string | readonly string[]
  /** What a list of several is called to a reader. */
  label?: string
}>()
</script>

<template>
  <p v-if="typeof said === 'string'" class="amiss text-small text-alarm">{{ said }}</p>

  <ul v-else class="amiss text-small text-alarm" :aria-label="label">
    <li v-for="(text, at) in said" :key="at">{{ text }}</li>
  </ul>
</template>

<style scoped>
/* A line or two saying what is wrong, and never a list to read: no bullet, and
   no room kept for one, so it stands over the edge the host draws it on. */
.amiss {
  margin: 0;
  padding-inline-start: 0;
  list-style: none;
}
</style>
