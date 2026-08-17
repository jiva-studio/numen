<script setup lang="ts">
/**
 * The window: one vault, one note in focus, and the plex around it.
 *
 * Choosing a node asks for that note's neighbourhood and hands it back to the
 * plex, which travels there by itself. What decides when to ask is in
 * `showing.ts`.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Activity, AgentPanel, Plex, remainingWord } from '@numen/ui'
import '@numen/ui/styles.css'
import { core } from './vault'
import { showing } from './showing'
import { footOf, type Phase } from './foot'
import { asPlex } from './plex'
import { core as agent } from './agent'
import { conversation } from './conversation'

const window = showing(core)
const { neighbourhood, indexing, failure, notice, trouble, unwatched, go } = window
const { chunks, embedded, reading, embedding, books, booksRead, learning, rate } = window
/** What the vault says about having work in hand. The counts do not say it. */
const { working: reads } = window

/** Everything this window says in its own voice. */
const words = {
  ask: 'Ask about this note',
  thinking: 'Thinking',
  unreachable: 'The agent could not be reached.',
  nothing: 'The agent finished without saying anything.',
  unsent: 'Did not send',
  stopped: 'The agent stopped here',
  reading: 'Reading',
  learning: 'Preparing search by meaning',
  words: 'Searching by words only — no model set',
}

/** What the foot of the window says, one sentence per phase. */
const saying: Record<Phase, string> = {
  reading: words.reading,
  learning: words.learning,
  wordsOnly: words.words,
  idle: '',
}

const activity = computed(() => {
  const foot = footOf({
    busy: reads.value,
    learning: learning.value,
    reading: reading.value,
    books: books.value,
    booksRead: booksRead.value,
    chunks: chunks.value,
    embedded: embedded.value,
    embedding: embedding.value,
    rate: rate.value,
  })
  return {
    says: saying[foot.phase],
    about: foot.about,
    working: foot.working,
    left: remainingWord(foot.left, foot.perSecond),
    tally: foot.tally,
  }
})

const { turns, working, ask, close } = conversation(agent, words)
const asked = ref('')

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

    <div class="below">
      <Plex
        v-if="neighbourhood && !failure && !indexing"
        class="plex"
        :neighbourhood="asPlex(neighbourhood)"
        :creatable="[]"
        @activate="go"
      />
      <div v-else class="plex" />

      <AgentPanel
        v-model="asked"
        class="agent"
        :turns="turns"
        :working="working"
        :placeholder="words.ask"
        @submit="send"
      >
        <template #failure="{ turn }">
          {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
        </template>
      </AgentPanel>
    </div>

    <Activity
      class="activity"
      :says="activity.says"
      :about="activity.about"
      :working="activity.working"
      :left="activity.left"
      :tally="activity.tally"
    />
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.below {
  display: flex;
  flex: 1;
  min-height: 0;
}

.plex {
  flex: 1;
  min-width: 0;
}

/* Floating: clear of the top, the bottom and the trailing edge. */
.agent {
  flex: none;
  inline-size: clamp(24rem, 32vw, 40rem);
  margin-block: var(--numen-inset-wide);
  margin-inline-end: var(--numen-inset-wide);
}

/* The foot of the window: clear of the plex, quiet when there is no work. */
.activity {
  flex: none;
  padding: 0.3rem 1rem;
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
