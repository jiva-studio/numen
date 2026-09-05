/**
 * The flashcards window put together: what it holds is which screen is on and
 * which vault is open.
 *
 * The rules are beside it: what a sitting is, what a keystroke asks for, what
 * the vaults come to, and what the window has to say. Everything here is made
 * once, as the window opens, and what comes out stands under the screen it
 * belongs to.
 */
import { computed, onMounted, onUnmounted, ref, useTemplateRef } from 'vue'
import { following, opensVault } from '@numen/ui'

import type NotesPanel from './NotesPanel.vue'
import type { PanelPlace } from './PanelCarousel.vue'
import { WINDOW, cards, deckName, itself } from './core'
import { counting } from './counting'
import { asks, picks, swallows } from './keying'
import { raising } from './notices'
import { reviewed } from './reviewed'
import { opens, scheduling } from './scheduling'
import { session } from './session'
import { asking } from './asking'
import { reading } from './reading'
import { screens } from './screens'
import { around } from './reading/core'
import { core as agent } from './agent/core'
import type { Grade, VaultCardsDue } from './core'
import type { Report } from './session'

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
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
  const at = computed<PanelPlace>(() =>
    showing.value === 'reading' ? 'before' : showing.value === 'asking' ? 'after' : 'here',
  )

  /**
   * The strip taken somewhere by a hand. A panel reached this way is opened, not
   * merely shown: what is in it is fetched and started when it is asked for.
   */
  const moved = (where: PanelPlace) => {
    if (where === 'before') void notesPanel.opens()
    else if (where === 'after') agentPanel.opens()
    else showing.value = 'here'
  }

  const agentPanel = asking({
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

  const notesPanel = reading({
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
    if (named === '' && showing.value === 'reading') notesPanel.shuts()
    else void notesPanel.opens(named)
  }

  const talks = () => {
    if (showing.value === 'asking') agentPanel.shuts()
    else agentPanel.opens()
  }

  /**
   * The three screens, and what each of them holds. Everything a screen took up
   * stands here beside it, which is the whole of what going back lets go of.
   */
  const { on, goes } = screens(['vaults', 'decks', 'session'] as const, {
    decks: [
      done.forget,
      schedules.forget,
      () => {
        vault.value = ''
      },
    ],
    session: [sat.forget, agentPanel.ends, notesPanel.ends],
  })

  /** What is read, so the keys can scroll it: the caret is nowhere in it. */
  const page = useTemplateRef<InstanceType<typeof NotesPanel>>('page')

  const chosen = computed(() => vaults.value.find((one) => one.vault === vault.value) ?? null)

  /**
   * Into a vault. A vault is opened once it has been counted, so what the rest of
   * them come to is nobody's question any more and the counting is let go of.
   */
  const choose = (id: string) => {
    stop()
    vault.value = id
    goes('decks')
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
    goes('session')
    reported(said)
  }

  /** Sit down to every deck one preset schedules, held to the budget it keeps. */
  const startPreset = async (preset: string) => {
    const said = await sat.start(vault.value, '', preset)
    if (!said) return
    goes('session')
    reported(said)
  }

  /**
   * Out of a sitting and back to the decks, with the counts as they now stand and
   * the days too: what a person just answered is part of what they have done.
   */
  const leave = async () => {
    goes('decks')
    void done.read(vault.value)
    await count()
    void schedules.read(chosen.value, today.value)
  }

  /** Back to the vaults, which is where a person picks another collection. */
  const vaultsAgain = async () => {
    goes('vaults')
    await count()
  }

  /**
   * The card answered. The conversation the panel was holding is over with the
   * card it was about.
   */
  const answered = async (how: Grade) => {
    agentPanel.ends()
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
        if (showing.value === 'asking') agentPanel.shuts()
        else notesPanel.shuts()
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
  const choosing = (press: KeyboardEvent, vault: VaultCardsDue) => {
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
      .getAgentState({})
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
      () => cards.watchReloads({}),
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
      () => itself.watchTasks({ window: WINDOW }),
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

  return {
    /** The window's own: which screen is on, and what it has to say. */
    on,
    notices,
    putAway,

    /** The list of vaults, and the way into one. */
    vaults: { list: vaults, counting: busy, choose },

    /** One vault's decks and presets, and the ways to sit down to them. */
    decks: { chosen, today, done, schedules, start, startPreset, vaultsAgain },

    /** The sitting, and the two panels standing beside the card. */
    session: { sat, at, moved, answered, talks, reads, leave, agentPanel, notesPanel },
  }
}
