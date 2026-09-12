<script setup lang="ts">
/**
 * A tool in hand, in the thread where it was reached for.
 *
 * A line about work: quieter than what is said, and marked.
 */
import { TypingIndicator } from '../typing-indicator'

withDefaults(
  defineProps<{
    /** What the tool is called, in the words it is to be shown by. */
    tool: string
    /** What it is working on, when that is worth saying. */
    about?: string
    /**
     * What is true of it beside its name: how much has been written, how long
     * the wait has lasted. It is what moves while nothing else does.
     */
    aside?: string
    /** Still in hand. */
    working?: boolean
  }>(),
  { about: '', aside: '', working: false },
)
</script>

<template>
  <p class="tool-call numen flex items-baseline gap-2 font-sans text-base text-hushed">
    <span class="tool-call__mark" :data-working="working || undefined" />
    <span class="min-w-0 truncate">{{ tool }}</span>
    <span v-if="about" class="min-w-0 flex-1 truncate opacity-70">{{ about }}</span>
    <span v-else class="flex-1" />
    <span v-if="aside" class="flex-none tabular-nums opacity-70">{{ aside }}</span>
    <TypingIndicator v-if="working" />
  </p>
</template>

<style scoped>
.tool-call__mark {
  flex: none;
  inline-size: 0.4em;
  block-size: 0.4em;
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.5;
}

.tool-call__mark[data-working] {
  opacity: 1;
}
</style>
