<script setup lang="ts">
/**
 * A tab with nothing in it yet, and the few things it can be told to be.
 *
 * What is offered is what the kinds of the window offer, each under the word
 * that kind is asked for by.
 */
import type { Tab } from '@numen/ui'
import { WORDS as words } from './words'

defineProps<{ becomes: readonly Tab[] }>()
defineEmits<{ (event: 'choose', kind: string): void }>()
</script>

<template>
  <div class="blank">
    <p class="blank__says">{{ words.choose }}</p>
    <ul class="blank__choices">
      <li v-for="one in becomes" :key="one.id">
        <button type="button" class="blank__choice" @click="$emit('choose', one.id)">
          {{ one.title }}
        </button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.blank {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  block-size: 100%;
  padding: var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: 0.85rem;
}

.blank__says {
  margin: 0;
  color: var(--numen-edge-label);
}

.blank__choices {
  display: flex;
  flex-direction: column;
  align-items: start;
  gap: 0.25rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.blank__choice {
  padding: 0.3rem 0.6rem;
  border: 0;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-node-fg);
  font: inherit;
  cursor: pointer;
}

.blank__choice:hover {
  background: var(--numen-bubble-bg);
}

.blank__choice:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: 1px;
}
</style>
