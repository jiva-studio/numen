<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { TypingIndicator } from '../typing-indicator'

withDefaults(
  defineProps<{
    /** What the tool is called, in the words it is to be shown by. */
    tool: string
    /** What it is working on, when that is worth saying. */
    subject?: string
    /**
     * What is true of it beside its name: how much has been written, how long
     * the wait has lasted. It is what moves while nothing else does.
     */
    aside?: string
    /** Still in hand. */
    isWorking?: boolean
  }>(),
  { subject: '', aside: '', isWorking: false },
)
</script>

<template>
  <p class="tool-call numen text-hushed flex items-center gap-2 font-sans text-base">
    <span class="tool-call__mark" :data-working="isWorking || undefined" />
    <span class="min-w-0 truncate">{{ tool }}</span>
    <span v-if="subject" class="min-w-0 flex-1 truncate opacity-70">{{ subject }}</span>
    <span v-else class="flex-1" />
    <span v-if="aside" class="flex-none tabular-nums opacity-70">{{ aside }}</span>
    <TypingIndicator v-if="isWorking" class="ml-0.5" />
  </p>
</template>

<style scoped>
.tool-call__mark {
  flex: none;
  inline-size: 0.45em;
  block-size: 0.45em;
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.4;
  transition:
    opacity 0.2s,
    transform 0.2s;
}

.tool-call__mark[data-working] {
  opacity: 1;
  animation: tool-call-pulse 1.4s ease-in-out infinite;
}

@keyframes tool-call-pulse {
  0%,
  100% {
    transform: scale(0.85);
    opacity: 0.4;
  }
  50% {
    transform: scale(1.15);
    opacity: 1;
  }
}
</style>
