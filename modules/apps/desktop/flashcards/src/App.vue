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
import Asking from './Asking.vue'
import { VERSION } from './version'
import { cards, deckName } from './core'
import { counting } from './counting'
import { asks, picks, swallows } from './keying'
import { raising } from './notices'
import { reviewed } from './reviewed'
import { session } from './session'
import { asking } from './asking'
import { core as agent } from './agent/core'
import type { Owing, Said } from './core'
import type { Report } from './session'

/** Which of the three screens the window is on. */
const on = ref<'vaults' | 'decks' | 'session'>('vaults')

/** The vault whose decks are open, and whose cards are being asked. */
const vault = ref('')

const { notices, says, failed, putAway } = raising()
const { vaults, counting: busy, count } = counting({ cards, failed })
const sat = session({ cards, failed })
const done = reviewed({ cards, failed })

/** Why nothing can be asked here, empty while something can. */
const unreachable = ref('')

const panel = asking({
  agent,
  card: () => sat.card.value,
  unreachable: () => unreachable.value,
  says: (said) => says(said, 'caution'),
})

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
  panel.ends()
  sat.forget()
  on.value = 'decks'
  void done.read(vault.value)
  await count()
}

/** Back to the vaults, which is where a person picks another collection. */
const vaultsAgain = async () => {
  panel.ends()
  sat.forget()
  done.forget()
  on.value = 'vaults'
  vault.value = ''
  await count()
}

/**
 * The card answered. The conversation the panel was holding is over with the
 * card it was about.
 */
const answered = async (how: Said) => {
  panel.ends()
  await sat.answer(how)
}

const keyed = (press: KeyboardEvent) => {
  if (on.value === 'decks') return chosen.value ? choosing(press, chosen.value) : undefined
  if (on.value !== 'session') return

  const asked = asks(press, { shown: sat.shown.value, asking: panel.open.value })
  if (!asked) return
  if (swallows(asked)) press.preventDefault()

  switch (asked.does) {
    case 'show':
      sat.show()
      break
    case 'answer':
      void answered(asked.how)
      break
    case 'takeBack':
      void sat.takeBack()
      break
    case 'leave':
      void leave()
      break
    case 'ask':
      panel.opens()
      break
    case 'shut':
      panel.shuts()
      break
  }
}

/** The keys a person picks what to sit down to with. */
const choosing = (press: KeyboardEvent, vault: Owing) => {
  const asked = picks(press, vault.decks.length)
  if (!asked) return
  press.preventDefault()

  switch (asked.does) {
    case 'all':
      void start('')
      break
    case 'deck': {
      const deck = vault.decks[asked.at]
      if (deck) void start(deck.deck)
      break
    }
    case 'back':
      void vaultsAgain()
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
  // Whether a card can be asked about is the window's to know before a person
  // reaches for it, so it is asked once and the way in is drawn from it.
  void cards
    .asking({})
    .then((said) => {
      unreachable.value = said.unreachable
    })
    .catch(() => {
      unreachable.value = 'The agent could not be reached.'
    })
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
      :asking="panel.open.value"
      @show="sat.show"
      @answer="answered"
      @take-back="sat.takeBack"
      @leave="leave"
      @ask="panel.opens()"
      @shut="panel.shuts()"
    >
      <template #panel>
        <Asking :held="panel" />
      </template>
    </Session>
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
