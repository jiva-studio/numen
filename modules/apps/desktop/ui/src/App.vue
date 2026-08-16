<script setup lang="ts">
/**
 * The window: one vault, one note in focus, and the plex around it.
 *
 * Choosing a node asks for that note's neighbourhood and hands it back to the
 * plex, which travels there by itself. What decides when to ask is in
 * `showing.ts`.
 */
import { onMounted, onUnmounted } from 'vue'
import { Plex } from '@numen/ui'
import '@numen/ui/tokens.css'
import { core } from './vault'
import { showing } from './showing'
import { asPlex } from './plex'

const window = showing(core)
const { neighbourhood, name, indexing, failure, notice, trouble, unwatched, go } = window

onMounted(window.start)
onUnmounted(window.close)
</script>

<template>
  <main>
    <header>
      <span class="vault">{{ name || 'numen' }}</span>
      <span v-if="neighbourhood" class="here">{{ neighbourhood.focus?.title }}</span>
    </header>

    <p v-if="unwatched" class="warning">not following the vault — {{ unwatched }}</p>
    <p v-if="trouble" class="warning">the vault could not be read — {{ trouble }}</p>
    <p v-if="notice" class="warning">{{ notice }}</p>

    <p v-if="failure" class="failure">{{ failure }}</p>
    <p v-else-if="indexing" class="waiting">reading the vault…</p>
    <p v-else-if="!neighbourhood && trouble" class="waiting">nothing was read</p>
    <p v-else-if="!neighbourhood" class="waiting">this vault holds no notes</p>

    <Plex
      v-else
      class="plex"
      :neighbourhood="asPlex(neighbourhood)"
      :creatable="[]"
      @activate="go"
    />
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

header {
  display: flex;
  gap: 0.75rem;
  align-items: baseline;
  padding: 0.6rem 1rem;
  border-bottom: 1px solid var(--numen-chrome-rule);
  font: 500 0.85rem system-ui, sans-serif;
}

.vault {
  opacity: 0.55;
}

.plex {
  flex: 1;
  min-height: 0;
}

.waiting,
.failure {
  margin: auto;
  font: 0.9rem system-ui, sans-serif;
  opacity: 0.6;
}

.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font: 0.8rem system-ui, sans-serif;
  background: light-dark(#fff4e5, #3a2e1c);
  color: light-dark(#7a4b00, #f0c890);
}

.failure {
  color: #b3261e;
  opacity: 1;
  max-width: 40rem;
  text-align: center;
}
</style>
