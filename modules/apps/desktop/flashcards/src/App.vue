<script setup lang="ts">
/**
 * The flashcards window: the vaults and what each owes, and the cards
 * themselves.
 *
 * It opens on what a person owes today and nothing else. They need not know an
 * editor exists.
 *
 * What it holds is which screen is on and which vault is open. The rules are
 * beside it: what a sitting is, what a keystroke asks for, what the vaults come
 * to, and what the window has to say.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Notices, following } from '@numen/ui'
import '@numen/ui/styles.css'

import Vaults from './Vaults.vue'
import Decks from './Decks.vue'
import Session from './Session.vue'
import Finished from './Finished.vue'
import { VERSION } from './version'
import { cards, deckName } from './core'
import { counting } from './counting'
import { asks, swallows } from './keying'
import { raising } from './notices'
import { reviewed } from './reviewed'
import { session } from './session'
import type { Report } from './session'

/** Which of the three screens the window is on. */
const on = ref<'vaults' | 'decks' | 'session'>('vaults')

/** The vault whose decks are open, and whose cards are being asked. */
const vault = ref('')

const { notices, says, failed, putAway } = raising()
const { vaults, counting: busy, count } = counting({ cards, failed })
const sat = session({ cards, failed })
const done = reviewed({ cards, failed })

const chosen = computed(() => vaults.value.find((one) => one.vaultId === vault.value) ?? null)

const choose = (id: string) => {
  vault.value = id
  on.value = 'decks'
  void done.read(id)
}

/**
 * What the sitting could not act on, said once as it opens: a deck whose cards
 * could not be given marks holds cards this sitting does not ask, and a line of
 * the vault's answers that could not be read is a card standing where the rest
 * of its history left it.
 */
const reported = (said: Report) => {
  if (said.unwritten.length) {
    says(
      `Not asked from ${said.unwritten.map(deckName).join(', ')}: the deck could not be written.`,
      'caution',
    )
  }
  if (said.skipped > 0) {
    says(`${said.skipped} answers in this vault could not be read.`, 'caution')
  }
}

const start = async (deck: string) => {
  const said = await sat.start(vault.value, deck)
  if (!said) return
  on.value = 'session'
  reported(said)
}

/**
 * Out of a sitting and back to the decks, with the counts as they now stand and
 * the days too: what a person just answered is part of what they have done.
 */
const leave = async () => {
  sat.forget()
  on.value = 'decks'
  void done.read(vault.value)
  await count()
}

/** Back to the vaults, which is where a person picks another collection. */
const vaultsAgain = async () => {
  sat.forget()
  done.forget()
  on.value = 'vaults'
  vault.value = ''
  await count()
}

const keyed = (press: KeyboardEvent) => {
  if (on.value !== 'session') return
  const asked = asks(press, { shown: sat.shown.value })
  if (!asked) return
  if (swallows(asked)) press.preventDefault()

  switch (asked.does) {
    case 'show':
      sat.show()
      break
    case 'answer':
      void sat.answer(asked.how)
      break
    case 'takeBack':
      void sat.takeBack()
      break
    case 'leave':
      void leave()
      break
  }
}

/** Whether the window is still open, which is how long anything is followed. */
let open = true

const follows = following({
  open: () => open,
  lost: failed,
  wait: (ms) => new Promise((then) => setTimeout(then, ms)),
})

onMounted(() => {
  window.addEventListener('keydown', keyed)
  void count()
  // A deck written or a card changed underneath the window is counted again
  // without a person asking. A sitting is left alone: its cards were laid out
  // when it opened, and what a deck says now is read at the next one.
  void follows(
    () => cards.moving({}),
    async () => {
      if (on.value === 'session') return
      await count()
      if (vault.value) await done.read(vault.value)
    },
  )
})
onUnmounted(() => {
  open = false
  window.removeEventListener('keydown', keyed)
})
</script>

<template>
  <main class="flashcards">
    <Vaults
      v-if="on === 'vaults'"
      :vaults="vaults"
      :counting="busy"
      :version="VERSION"
      @choose="choose"
    />

    <Decks
      v-else-if="on === 'decks' && chosen"
      :vault="chosen"
      :days="done.days.value"
      :due="done.due.value"
      @start="start"
      @back="vaultsAgain"
    />

    <Finished v-else-if="sat.over.value" :done="sat.done.value" @leave="leave" />

    <Session
      v-else-if="sat.card.value"
      :card="sat.card.value"
      :shown="sat.shown.value"
      :left="sat.left.value"
      :taken-back="sat.answers.value.length > 0"
      @show="sat.show"
      @answer="sat.answer"
      @take-back="sat.takeBack"
      @leave="leave"
    />
  </main>

  <!-- What the window has to say, in the corner every window says it in. -->
  <Notices :notices="notices" @gone="putAway" />
</template>

<style scoped>
.flashcards {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  padding: var(--numen-inset-wide);
  gap: var(--numen-inset);
}
</style>
