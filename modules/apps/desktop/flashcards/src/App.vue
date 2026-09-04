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
import { computed, onMounted, onUnmounted, ref, useTemplateRef } from 'vue'
import { Notices, following, opensVault } from '@numen/ui'
import '@numen/ui/styles.css'

import Vaults from './Vaults.vue'
import Decks from './Decks.vue'
import Session from './Session.vue'
import Finished from './Finished.vue'
import Asking from './Asking.vue'
import Reading from './Reading.vue'
import type { Where } from './Beside.vue'
import { VERSION } from './version'
import { cards, deckName } from './core'
import { counting } from './counting'
import { asks, picks, swallows } from './keying'
import { raising } from './notices'
import { reviewed } from './reviewed'
import { opens, scheduling } from './scheduling'
import { session } from './session'
import { asking } from './asking'
import { reading } from './reading'
import { around } from './reading/core'
import { core as agent } from './agent/core'
import type { Owing, Said } from './core'
import type { Report } from './session'

/** Which of the three screens the window is on. */
const on = ref<'vaults' | 'decks' | 'session'>('vaults')

/** The vault whose decks are open, and whose cards are being asked. */
const vault = ref('')

const { notices, says, failed, doing, putAway } = raising()
const { vaults, counting: busy, day: today, count, stop } = counting({ cards, failed })
const sat = session({ cards, failed })
const done = reviewed({ cards, failed })
const schedules = scheduling({ presets: cards })

/** Why nothing can be asked here, empty while something can. */
const unreachable = ref('')

/**
 * Which of the card and the two panels beside it the window is showing. One
 * thing is in the window at a time, so one thing says which, and a panel coming
 * in is the other one going out.
 */
const showing = ref<'reading' | 'here' | 'asking'>('here')

/** The same thing in the words the strip stands the three in. */
const at = computed<Where>(() =>
  showing.value === 'reading' ? 'before' : showing.value === 'asking' ? 'after' : 'here',
)

/**
 * The strip taken somewhere by a hand. A panel reached this way is opened, not
 * merely shown: what is in it is fetched and started when it is asked for.
 */
const moved = (where: Where) => {
  if (where === 'before') void read.opens()
  else if (where === 'after') panel.opens()
  else showing.value = 'here'
}

const panel = asking({
  agent,
  card: () => sat.card.value,
  unreachable: () => unreachable.value,
  open: () => showing.value === 'asking',
  // A panel put away takes the window back to the card only when the window is
  // on it: a card answered with the reading up ends the conversation, and the
  // reading stays where it is.
  shows: (open) => {
    if (open) showing.value = 'asking'
    else if (showing.value === 'asking') showing.value = 'here'
  },
  says: (said) => says(said, 'caution'),
})

const read = reading({
  open: () => showing.value === 'reading',
  shows: (open) => {
    if (open) showing.value = 'reading'
    else if (showing.value === 'reading') showing.value = 'here'
  },
  vault: () => vault.value,
  deck: () => sat.card.value?.deck ?? '',
  around,
  says: (said) => says(said, 'caution'),
})

/**
 * The two ways in, each of them also the way out: a person who brought a panel
 * in with a key or a control takes it away with the same one.
 *
 * A link pressed in the card names the note to open on, and asks for the
 * reading rather than toggling it: the press was about that note.
 */
const reads = (named = '') => {
  if (named === '' && showing.value === 'reading') read.shuts()
  else void read.opens(named)
}

const talks = () => {
  if (showing.value === 'asking') panel.shuts()
  else panel.opens()
}

/** What is read, so the keys can scroll it: the caret is nowhere in it. */
const page = useTemplateRef<InstanceType<typeof Reading>>('page')

const chosen = computed(() => vaults.value.find((one) => one.vault === vault.value) ?? null)

/**
 * Into a vault. A vault is opened once it has been counted, so what the rest of
 * them come to is nobody's question any more and the counting is let go of.
 */
const choose = (id: string) => {
  stop()
  vault.value = id
  on.value = 'decks'
  void done.read(id)
  void schedules.read(chosen.value, today.value)
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

/** Sit down to every deck one preset schedules, held to the budget it keeps. */
const startPreset = async (preset: string) => {
  const said = await sat.start(vault.value, '', preset)
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
  read.ends()
  sat.forget()
  on.value = 'decks'
  void done.read(vault.value)
  await count()
  void schedules.read(chosen.value, today.value)
}

/** Back to the vaults, which is where a person picks another collection. */
const vaultsAgain = async () => {
  panel.ends()
  read.ends()
  sat.forget()
  done.forget()
  schedules.forget()
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
  if (on.value === 'vaults') return picking(press)
  if (on.value === 'decks') return chosen.value ? choosing(press, chosen.value) : undefined
  if (on.value !== 'session') return

  const asked = asks(press, {
    shown: sat.shown.value,
    asking: showing.value === 'asking',
    reading: showing.value === 'reading',
  })
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
      talks()
      break
    case 'read':
      reads()
      break
    case 'scroll':
      page.value?.scrolls(asked.back)
      break
    case 'shut':
      if (showing.value === 'asking') panel.shuts()
      else read.shuts()
      break
  }
}

/**
 * The keys a person picks a vault with: a letter opens the vault standing at
 * it, which is the letter drawn on that row. A vault whose count has not
 * arrived carries no letter, and the letter standing at it opens nothing.
 */
const picking = (press: KeyboardEvent) => {
  const at = opensVault(press, vaults.value.length)
  const one = at === null ? undefined : vaults.value[at]
  if (!one || !one.counted) return
  press.preventDefault()
  choose(one.vault)
}

/** The keys a person picks what to sit down to with. */
const choosing = (press: KeyboardEvent, vault: Owing) => {
  const asked = picks(press, vault.decks.length)
  if (!asked) return
  press.preventDefault()

  switch (asked.does) {
    case 'all':
      if (vault.due + vault.new > 0) void start('')
      break
    case 'deck': {
      const deck = vault.decks[asked.at]
      if (deck && opens(deck, schedules.byDeck.value)) void start(deck.deck)
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
    async (said) => {
      // The stream says nothing on its own account so that a page that has gone
      // fails the write. Only a move is a move.
      if (!said.reload) return
      if (on.value === 'session') return
      await count()
      if (!vault.value) return
      await done.read(vault.value)
      await schedules.read(chosen.value, today.value)
    },
  )
  // What is being done behind the window, which is a vault read into the index.
  // It is a stream because a reading begins without the page asking for one.
  void follows(
    () => cards.tasks({}),
    (said) => {
      doing(
        said.tasks.map((at) => ({
          id: at.id,
          doing: at.doing,
          about: at.about,
          failed: at.failed,
          asked: at.asked,
        })),
      )
    },
  )
})
onUnmounted(() => {
  open = false
  stop()
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
      :presets="schedules.presets.value"
      :by-deck="schedules.byDeck.value"
      :scheduled="schedules.known.value"
      :today="today"
      @start="start"
      @start-preset="startPreset"
      @back="vaultsAgain"
    />

    <Finished v-else-if="sat.over.value" :done="sat.done.value" @leave="leave" />

    <Session
      v-else-if="sat.card.value"
      :card="sat.card.value"
      :shown="sat.shown.value"
      :left="sat.left.value"
      :taken-back="sat.answers.value.length > 0"
      :at="at"
      @update:at="moved"
      @show="sat.show"
      @answer="answered"
      @take-back="sat.takeBack"
      @leave="leave"
      @ask="talks"
      @read="reads"
    >
      <template #reading>
        <Reading ref="page" :held="read" />
      </template>
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
