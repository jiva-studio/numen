<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * Choosing a node asks for that note's neighbourhood and hands it back to the
 * plex, which travels there by itself. What decides when to ask is in
 * `showing.ts`; what each tab stands for is settled here and nowhere else.
 */
import { onMounted, onUnmounted, ref } from 'vue'
import { Agent, Plex, Workspace } from '@numen/ui'
import type { WorkspaceLayout } from '@numen/ui'
import '@numen/ui/styles.css'
import { core } from './vault'
import { showing } from './showing'
import { asPlex } from './plex'
import { core as agent } from './agent'
import { conversation } from './conversation'
import { AGENT, PLEX, TABS, opening } from './workspace'

const window = showing(core)
const { neighbourhood, indexing, failure, notice, trouble, unwatched, go } = window

/** Everything this window says in its own voice. */
const words = {
  ask: 'Ask about this note',
  thinking: 'Thinking',
  unreachable: 'The agent could not be reached.',
  nothing: 'The agent finished without saying anything.',
  unsent: 'Did not send',
  stopped: 'The agent stopped here',
}

const { turns, working, ask, close } = conversation(agent, words)
const asked = ref('')
const layout = ref<WorkspaceLayout>(opening())

const send = (text: string) => {
  asked.value = ''
  void ask(text, neighbourhood.value?.focus?.path ?? '')
}

onMounted(window.start)
onUnmounted(() => {
  window.close()
  close()
})
</script>

<template>
  <main>
    <p v-if="unwatched" class="warning">not following the vault — {{ unwatched }}</p>
    <p v-if="trouble" class="warning">the vault could not be read — {{ trouble }}</p>
    <p v-if="notice" class="warning">{{ notice }}</p>

    <p v-if="failure" class="failure">{{ failure }}</p>
    <p v-else-if="indexing" class="waiting">reading the vault…</p>
    <p v-else-if="!neighbourhood && trouble" class="waiting">nothing was read</p>
    <p v-else-if="!neighbourhood" class="waiting">this vault holds no notes</p>

    <Workspace v-model="layout" class="below" :tabs="TABS">
      <template #tab="{ id }">
        <Plex
          v-if="id === PLEX && neighbourhood && !failure && !indexing"
          :neighbourhood="asPlex(neighbourhood)"
          :creatable="[]"
          @activate="go"
        />

        <Agent
          v-else-if="id === AGENT"
          v-model="asked"
          :turns="turns"
          :working="working"
          :placeholder="words.ask"
          @submit="send"
        >
          <template #failure="{ turn }">
            {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
          </template>
        </Agent>

        <div v-else />
      </template>
    </Workspace>
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.below {
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
