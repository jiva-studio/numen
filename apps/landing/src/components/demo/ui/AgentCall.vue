<script setup lang="ts">
/** One line about the agent's work: the tool it used, on what, and a word aside. */
import type { Turn } from '../model/agent'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ turn: Turn }>()
</script>

<template>
  <div class="agent__call">
    <span class="agent__mark" />
    <span class="agent__tool">{{ turn.text }}</span>
    <span class="agent__subject">{{ turn.subject }}</span>
    <span class="agent__aside">{{ turn.aside }}</span>
  </div>
</template>

<style scoped>
.agent__call {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  inline-size: 100%;
  padding: 0;
  border: 0;
  background: none;
  color: var(--numen-hushed);
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.agent__mark {
  flex: none;
  inline-size: 0.45em;
  block-size: 0.45em;
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.4;
}

.agent__tool {
  color: var(--numen-ink);
  opacity: 0.7;
  white-space: nowrap;
}

.agent__subject {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent__aside {
  margin-inline-start: auto;
  white-space: nowrap;
}
</style>
