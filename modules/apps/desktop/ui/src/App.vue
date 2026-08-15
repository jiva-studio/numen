<script setup lang="ts">
/**
 * The window: one vault, one note in focus, and the plex around it.
 *
 * Choosing a node asks for that note's neighbourhood and hands it back to the
 * plex, which travels there by itself.
 */
import { onMounted, ref } from 'vue'
import { Plex } from '@numen/ui'
import '@numen/ui/tokens.css'
import { vault } from './vault'
import { asPlex, type Neighbourhood } from './plex'

const neighbourhood = ref<Neighbourhood | null>(null)
const name = ref<string>('')
const indexing = ref(true)
const failure = ref<string>('')

async function go(path: string) {
  try {
    neighbourhood.value = await vault.neighbourhood({ path })
  } catch (error) {
    failure.value = String(error)
  }
}

/** Waits for the scan to have stored something, then shows the first note. */
async function start() {
  try {
    for (;;) {
      const state = await vault.state({})
      name.value = state.name
      const { note } = await vault.opening({})
      if (note) {
        indexing.value = false
        await go(note.path)
        return
      }
      if (state.ready) {
        indexing.value = false
        return
      }
      await new Promise((wake) => setTimeout(wake, 100))
    }
  } catch (error) {
    failure.value = String(error)
    indexing.value = false
  }
}

onMounted(start)
</script>

<template>
  <main>
    <header>
      <span class="vault">{{ name || 'numen' }}</span>
      <span v-if="neighbourhood" class="here">{{ neighbourhood.focus?.title }}</span>
    </header>

    <p v-if="failure" class="failure">{{ failure }}</p>
    <p v-else-if="indexing" class="waiting">reading the vault…</p>
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
  border-bottom: 1px solid var(--numen-edge, #d8d8d8);
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

.failure {
  color: #b3261e;
  opacity: 1;
  max-width: 40rem;
  text-align: center;
}
</style>
