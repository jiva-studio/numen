/**
 * The flashcards window put together: what it holds is which screen is on and
 * which vault is open.
 *
 * The rules are beside it: what a session is, what a keystroke asks for, what
 * the vaults come to, and what the window has to say. Everything here is made
 * once, as the window opens, and what comes out stands under the screen it
 * belongs to.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { opensVault } from '@numen/ui'

import { cards } from '@/shared/clients'
import { screens } from '@/shared/screens'
import { deckName, useReviewCounter } from '@/entities/vault'
import { asks, picks, swallows } from '@/features/keyboard'
import { opens, useReviewedDays, useVaultPresets } from '@/pages/decks'
import { useReviewSession } from '@/pages/session'
import { useNotices } from './notices'
import { usePanels } from './usePanels'
import { useWindowStreams } from './useWindowStreams'
import type { Grade } from '@/entities/card'
import type { VaultCardsDue } from '@/entities/vault'
import type { NotesPanel, Report } from '@/pages/session'

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  /** The vault whose decks are open, and whose cards are being asked. */
  const vault = ref('')

  const { notices, says, failed, doing, putAway } = useNotices()
  const { vaults, counting: busy, day: today, count, stop } = useReviewCounter({ cards, failed })
  const state = useReviewSession({ cards, failed })
  const done = useReviewedDays({ cards, failed })
  const schedules = useVaultPresets({ presets: cards })

  /** Why nothing can be asked here, empty while something can. */
  const unreachable = ref('')

  const { showing, at, moved, agentPanel, notesPanel, reads, talks } = usePanels({
    card: () => state.card.value,
    unreachable: () => unreachable.value,
    vault: () => vault.value,
    says: (said) => says(said, 'caution'),
  })

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
    session: [state.forget, agentPanel.ends, notesPanel.ends],
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
   * What the session could not act on, said once as it opens: a deck whose cards
   * could not be given marks holds cards this session does not ask, and a line of
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
    const said = await state.start(vault.value, deck)
    if (!said) return
    goes('session')
    reported(said)
  }

  /** Sit down to every deck one preset schedules, held to the budget it keeps. */
  const startPreset = async (preset: string) => {
    const said = await state.start(vault.value, '', preset)
    if (!said) return
    goes('session')
    reported(said)
  }

  /**
   * Out of a session and back to the decks, with the counts as they now stand and
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
    await state.answer(how)
  }

  const keyed = (press: KeyboardEvent) => {
    if (on.value === 'vaults') return picking(press)
    if (on.value === 'decks') return chosen.value ? choosing(press, chosen.value) : undefined
    if (on.value !== 'session') return

    const asked = asks(press, {
      shown: state.shown.value,
      asking: showing.value === 'asking',
      reading: showing.value === 'reading',
    })
    if (!asked) return
    if (swallows(asked)) press.preventDefault()

    switch (asked.does) {
      case 'show':
        state.show()
        break
      case 'answer':
        void answered(asked.how)
        break
      case 'takeBack':
        void state.takeBack()
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

  /**
   * A deck written or a card changed underneath the window. A session is left
   * alone: its cards were laid out when it opened, and what a deck says now is
   * read at the next one.
   */
  const refresh = async () => {
    if (on.value === 'session') return
    await count()
    if (!vault.value) return
    await done.read(vault.value)
    await schedules.read(chosen.value, today.value)
  }

  useWindowStreams({ failed, doing, keyed, count, stop, refresh, unreachable })

  return {
    /** The window's own: which screen is on, and what it has to say. */
    on,
    notices,
    putAway,

    /** The list of vaults, and the way into one. */
    vaults: { list: vaults, counting: busy, choose },

    /** One vault's decks and presets, and the ways to sit down to them. */
    decks: { chosen, today, done, schedules, start, startPreset, vaultsAgain },

    /** The session, and the two panels standing beside the card. */
    session: { state, at, moved, answered, talks, reads, leave, agentPanel, notesPanel },
  }
}
