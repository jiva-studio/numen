/**
 * The steps of a command draw rows whose action carries no fixed answer: the step is
 * told by its own kind, and the palette hands the command what was typed or chosen.
 */
import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../words'
import { createAppCommands } from '@/testing/appCommands'
import * as stepModule from '../lib/step/step'
import { runSupport } from './palette/runSupport'
import { createCommandPalette } from './palette/palette'
import type { StepGroup } from '@/shared/model/command'
import type { CommandTarget } from '@/shared/model/command'

const words = { ...windowWords, ...PALETTE_WORDS }

const NAMED_ANSWERS = ['NAME', 'PICK', 'OPEN', 'CHOSEN', 'YES', 'NO', 'EXACT']
const NAMED_IDS = ['name', 'pick', 'open', 'chosen', 'yes', 'no', 'exactly']

const front: CommandTarget = {
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
}

const THEMES: readonly StepGroup[] = [
  { id: 'themes', title: 'Themes', items: [{ id: 'preset:numen', title: 'numen' }] as never },
]

const createPalette = () => {
  const showing = shallowRef('physics')
  const commands = createCommandPalette({
    core: {
      names: async () => [{ path: 'physics/Heat.md', title: 'Heat' }] as never,
      vaults: async () => ({
        vaults: [
          { id: 'physics', name: 'Physics', path: '/physics', isMissing: false },
          { id: 'heat', name: 'Heat', path: '/heat', isMissing: false },
        ],
        showing: showing.value,
      }),
    },
    words,
    at: () => front,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => THEMES, previewItem: vi.fn() },
    runs: runSupport(),
    getCommands: () => [
      ...createAppCommands(),
      {
        id: 'setting:app:appearance.theme',
        text: 'Theme',
        group: 'settings',
        input: 'items',
        isAvailable: () => true,
      },
    ],
  })
  commands.palette.setOpen(true)
  return commands
}

const settle = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('the answers of a step', () => {
  it('are no names the step module exports', () => {
    const exported = Object.keys(stepModule)

    for (const answer of NAMED_ANSWERS) expect(exported).not.toContain(answer)
  })

  it.each([
    ['a name to type', 'app:note.rename'],
    ['a confirmation', 'app:trash.empty'],
    ['a choice from a list', 'setting:app:appearance.theme'],
  ])('are not named in a row of %s', (_what, id) => {
    const commands = createPalette()
    commands.palette.startCommand(id, front)

    const rows = commands.view.groups.value.flatMap((group) => group.items)
    const ids = rows.flatMap((row) => [row.id, ...(row.actions ?? []).map((action) => action.id)])

    expect(rows.length).toBeGreaterThan(0)
    for (const named of NAMED_IDS) expect(ids).not.toContain(named)
  })

  it('are not named in a row of the notes found, or of the vaults', async () => {
    const found = createPalette()
    found.palette.startCommand('app:file.open#note', front)
    await found.palette.setTyped('he')
    const vaults = createPalette()
    vaults.palette.startCommand('app:vault.remove', front)
    await settle()

    for (const commands of [found, vaults]) {
      const rows = commands.view.groups.value.flatMap((group) => group.items)
      const ids = rows.flatMap((row) => [row.id, ...(row.actions ?? []).map((action) => action.id)])
      expect(rows.length).toBeGreaterThan(0)
      for (const named of NAMED_IDS) expect(ids).not.toContain(named)
    }
  })

  it('are not named in the two rows of a confirmation', async () => {
    const commands = createPalette()
    commands.palette.startCommand('app:vault.remove', front)
    await settle()
    const heat = commands.view.groups.value[0]?.items.find((row) => row.id === 'heat')
    commands.palette.chooseItem('heat', heat?.actions?.[0]?.id ?? '')

    const rows = commands.view.groups.value.flatMap((group) => group.items)
    const ids = rows.flatMap((row) => [row.id, ...(row.actions ?? []).map((action) => action.id)])

    expect(rows).toHaveLength(2)
    for (const named of NAMED_IDS) expect(ids).not.toContain(named)
  })

  it('run a confirmed command over the vault chosen, and go back where the other row is chosen', async () => {
    const commands = createPalette()
    commands.palette.startCommand('app:vault.remove', front)
    await settle()
    const heat = commands.view.groups.value[0]?.items.find((row) => row.id === 'heat')
    commands.palette.chooseItem('heat', heat?.actions?.[0]?.id ?? '')
    const [keep, confirm] = commands.view.groups.value[0]?.items ?? []

    expect(commands.palette.chooseItem(keep?.id ?? '', keep?.actions?.[0]?.id ?? '')).toBeNull()
    await settle()
    expect(commands.view.groups.value[0]?.items.map((row) => row.id)).toStrictEqual([
      'physics',
      'heat',
    ])
    commands.palette.chooseItem('heat', heat?.actions?.[0]?.id ?? '')
    expect(
      commands.palette.chooseItem(confirm?.id ?? '', confirm?.actions?.[0]?.id ?? ''),
    ).toMatchObject({
      id: 'app:vault.remove',
      target: { vault: { id: 'heat', name: 'Heat' } },
    })
  })
})
