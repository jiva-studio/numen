/**
 * What the palette draws where it stands: the commands offered over what is in
 * front, the rows of a list, the two answers of a confirmation.
 *
 * Every one of these is the state read again, so a vault moving under an open
 * step is drawn as it is. Nothing here decides anything or asks the
 * application for anything: `palette.ts` holds the steps and does that.
 */
import type { Ref } from 'vue'
import type { PaletteGroup, PaletteItem } from '@numen/ui'
import { isWebAddress, isWebUrl } from './address'
import type { NoteType } from '../note'
import type { Vault } from '../vaults'
import { inGroup } from './commands'
import type { PaletteLists, StepRow } from './lists'
import type { RunSupport } from './runs'
import type { NameMatch } from './search'
import { CHOSEN, EXACT, NAME, NO, OPEN, PICK, YES, type PendingStep } from './step'
import type {
  Command,
  CommandGroup,
  CommandTarget,
  ConfirmWords,
  RetypeWords,
  Words,
} from './target'

/** Everything the view reads, which is the palette's own state. */
export interface ViewState {
  readonly words: Words
  /** Every command, and the one another command's row reaches by its id. */
  readonly commands: readonly Command[]
  readonly byId: ReadonlyMap<string, Command>
  readonly runs: RunSupport
  /** The lists the window holds, which a choosing step draws. */
  readonly holds: PaletteLists
  /** The notes a search turned up, and the vaults the installation holds. */
  readonly found: Readonly<Ref<readonly NameMatch[]>>
  readonly known: Readonly<Ref<readonly Vault[]>>
  /** The vault the window is showing, which is the one it will not open again. */
  readonly showing: Readonly<Ref<string>>
  /** Whether the vault is being asked, and what it said when it refused. */
  readonly working: Readonly<Ref<boolean>>
  readonly said: Readonly<Ref<string>>
  /** What a step is over, in the words a person reads it as. */
  readonly calling: (step: PendingStep) => string
  readonly named: (step: PendingStep) => string
}

/** The view one palette draws, over the state that palette holds. */
export function view(state: ViewState) {
  const { words, commands, byId, runs, holds, found, known, showing, working, said } = state
  const { calling, named } = state

  /** One command as it is drawn, and nothing where the words typed leave it out. */
  const drawn = (one: Command, over: CommandTarget, word: string): PaletteItem | null => {
    const found = word === '' ? -1 : one.text.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) return null
    const also = one.also ? byId.get(one.also) : undefined
    return {
      id: one.id,
      title: one.text,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(one.keys ? { keys: one.keys } : {}),
      actions: [
        { id: one.id, text: one.text },
        ...(also && also.where(over, runs) ? [{ id: also.id, text: also.text }] : []),
      ],
    }
  }

  /** Why nothing over the note in front is offered. */
  const why = (over: CommandTarget): string =>
    !over.ready ? words.noVault : over.path ? words.noneFound : words.noNote

  /** Every command offered over what is in front, in the groups it holds. */
  const listed = (over: CommandTarget, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    const items = (group: CommandGroup): readonly PaletteItem[] =>
      inGroup(commands, group)
        .filter((one) => one.where(over, runs))
        .map((one) => drawn(one, over, word))
        .filter((item) => item !== null)

    // The runs are offered over a book and over a recording, and their group
    // stands where one of them is in front.
    const overFile = items('file')

    return [
      { id: 'note', title: words.overNote, items: items('note'), silence: why(over) },
      ...(overFile.length === 0 ? [] : [{ id: 'file', title: words.overFile, items: overFile }]),
      { id: 'window', title: words.overWindow, items: items('window'), silence: words.noneFound },
      { id: 'vault', title: words.overVault, items: items('vault'), silence: words.noneFound },
    ]
  }

  /** A name to give, as the one thing the words typed can be. */
  const naming = (text: string): PaletteGroup => {
    const name = text.trim()
    return {
      id: 'naming',
      title: words.naming,
      items: name
        ? [
            {
              id: NAME,
              title: `${words.callIt} “${name}”`,
              actions: [{ id: NAME, text: words.callIt }],
            },
          ]
        : [],
      silence: words.typeName,
    }
  }

  /**
   * An address to point at, as the one thing the words typed can be. What is
   * offered is what a browser would go to; the vault reads it again and is what
   * refuses one it cannot fetch.
   */
  const address = (text: string): PaletteGroup => {
    const raw = text.trim()
    return {
      id: 'address',
      title: words.address,
      items: isWebUrl(raw)
        ? [
            {
              id: NAME,
              title: `${words.importIt} “${raw}”`,
              actions: [{ id: NAME, text: words.importIt }],
            },
          ]
        : [],
      silence: raw ? words.notAnAddress : words.typeAddress,
    }
  }

  /**
   * The notes the vault turned up. A note found by a heading is that note, and
   * a note found twice is one row.
   */
  /** Which of four the note a row of the picking step stands for is. */
  const typeOf = (id: string): NoteType | null =>
    found.value.find((one) => one.path === id)?.type ?? null

  const picking = (text: string): PaletteGroup => {
    const seen = new Set<string>()
    const items: PaletteItem[] = []
    for (const one of found.value) {
      if (seen.has(one.path)) continue
      seen.add(one.path)
      items.push({
        id: one.path,
        title: one.title || one.path,
        ...(one.heading ? {} : { at: one.at }),
        actions: [{ id: PICK, text: words.travel }],
      })
    }
    return {
      id: 'picking',
      title: words.names,
      items,
      working: working.value,
      silence: said.value || (text.trim() ? words.noneFound : words.typeNote),
    }
  }

  /**
   * The groups a list the window holds is drawn in, narrowed by the words typed.
   * They are read again on every keystroke, and a row drawn as not to be chosen
   * is not chosen.
   */
  const choosing = (step: PendingStep, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    return holds.offers(step.command.id, text).map((group) => ({
      id: group.id,
      title: group.title,
      items: group.items.map((one) => offered(one, word)).filter((item) => item !== null),
      silence: group.silence ?? words.noneFound,
    }))
  }

  /** One such row, and nothing where the words typed leave it out. */
  const offered = (one: StepRow, word: string): PaletteItem | null => {
    const found = word === '' ? -1 : one.title.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) return null
    return {
      id: one.id,
      title: one.title,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(one.detail ? { detail: one.detail } : {}),
      ...(one.disabled ? { disabled: true } : {}),
      actions: [{ id: CHOSEN, text: words.chooses }],
    }
  }

  /**
   * Why a vault the list holds is drawn and not chosen: its folder is not
   * there, or it is the one the window is showing. A vault that can be chosen
   * is marked with nothing.
   */
  const aside = (one: Vault): string =>
    one.missing ? words.gone : one.id === showing.value ? words.current : ''

  /**
   * The vaults the installation holds. The two it will not take are marked
   * ahead of their folder, and cannot be chosen. The folder is what tells one
   * vault from another, so it keeps the room.
   */
  const listing = (text: string, step: PendingStep): PaletteGroup => {
    const word = text.trim().toLowerCase()
    const items: PaletteItem[] = known.value
      .filter((one) => word === '' || one.name.toLowerCase().includes(word))
      .map((one) => {
        const why = aside(one)
        return {
          id: one.id,
          title: one.name,
          detail: why ? `${why} · ${one.path}` : one.path,
          ...(why ? { disabled: true } : {}),
          actions: [{ id: OPEN, text: step.command.text }],
        }
      })
    return {
      id: 'vaults',
      title: words.vaults,
      items,
      working: working.value,
      silence: said.value || words.noneFound,
    }
  }

  /**
   * The two answers put before a note goes to the trash. The one that changes
   * nothing is drawn first, and it is the one the keyboard opens on. Each is
   * reached by the name it is offered under, as an item of any other step is.
   */
  const asking = (step: PendingStep, text: string): PaletteGroup => {
    const word = text.trim().toLowerCase()
    const { keeps = '', kept = '', does = '', then = '' }: Partial<ConfirmWords> =
      step.command.answers ?? {}
    const items: PaletteItem[] = [
      {
        id: NO,
        title: keeps,
        detail: kept,
        actions: [{ id: NO, text: keeps }],
      },
      {
        id: YES,
        title: `${does} ${named(step)}`,
        detail: then,
        actions: [{ id: YES, text: does }],
      },
    ]
    return {
      id: 'asking',
      title: words.asking,
      items: word
        ? items.filter((one) => (one.actions?.[0]?.text ?? '').toLowerCase().includes(word))
        : items,
      silence: words.answer,
    }
  }

  /** The name typed back, which is the one thing that reaches destroying. */
  const exactly = (step: PendingStep, text: string): PaletteGroup => {
    const title = calling(step)
    const { does = '', then = '' }: Partial<RetypeWords> = step.command.warns ?? {}
    return {
      id: 'exactly',
      title: words.exactly,
      items: [
        {
          id: EXACT,
          title: `${does} “${title}”`,
          detail: then,
          disabled: text.trim() !== title,
          actions: [{ id: EXACT, text: does }],
        },
      ],
    }
  }

  /** What the palette draws where it stands, over the words typed into it. */
  const groupsOf = (step: PendingStep | null, over: CommandTarget, typed: string) => {
    if (!step) return listed(over, typed)
    if (step.step === 'naming') return [naming(typed)]
    if (step.step === 'address') return [address(typed)]
    if (step.step === 'picking') return [picking(typed)]
    if (step.step === 'choosing') return choosing(step, typed)
    if (step.step === 'vaults') return [listing(typed, step)]
    if (step.step === 'asking') return [asking(step, typed)]
    return [exactly(step, typed)]
  }

  return { groupsOf, typeOf, aside }
}
