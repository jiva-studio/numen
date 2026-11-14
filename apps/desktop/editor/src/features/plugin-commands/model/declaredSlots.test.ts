/** The slot entries the window passes on, read from the answers of the core. */
import { describe, expect, it } from 'vitest'
import type { DeclaredSlot, DeclaredCommandsApi } from '@/entities/plugin'
import { useDeclaredCommands } from './declaredCommands'
import { UNANSWERED_RUNS } from '@/testing/declaredCommands'

type DeclaredList = Awaited<ReturnType<DeclaredCommandsApi['listDeclaredCommands']>>

const menu = (command: string, overrides: Partial<DeclaredSlot> = {}): DeclaredSlot => ({
  pluginId: 'org.example.atlas',
  slot: 'palette.searchResult',
  command,
  ...overrides,
})

const list = (...slots: DeclaredSlot[]): DeclaredList => ({ commands: [], slots })

describe('the slot entries of the plugins', () => {
  it('are unknown until the first list has answered', async () => {
    const declared = useDeclaredCommands({ ...UNANSWERED_RUNS, list: async () => list(menu('atlas.open')) })

    expect(declared.slots.value).toBeUndefined()
    await declared.load()

    expect(declared.slots.value).toEqual([menu('atlas.open')])
  })

  it('keep the group and the order of each entry', async () => {
    const declared = useDeclaredCommands({ ...UNANSWERED_RUNS,
      list: async () =>
        list(
          menu('atlas.open', { group: 'navigate', order: 20 }),
          menu('atlas.zero', { order: 0 }),
        ),
    })

    await declared.load()

    expect(declared.slots.value).toEqual([
      menu('atlas.open', { group: 'navigate', order: 20 }),
      menu('atlas.zero', { order: 0 }),
    ])
  })

  it('hold the newest list when an earlier read answers last', async () => {
    let answerFirst: (answer: DeclaredList) => void = () => undefined
    const first = new Promise<DeclaredList>((resolve) => {
      answerFirst = resolve
    })
    const answers = [first, Promise.resolve(list(menu('atlas.newest')))]
    const declared = useDeclaredCommands({ ...UNANSWERED_RUNS, list: () => answers.shift() as Promise<DeclaredList> })

    const slow = declared.load()
    await declared.load()
    answerFirst(list(menu('atlas.stale')))
    await slow

    expect(declared.slots.value?.map((one) => one.command)).toEqual(['atlas.newest'])
  })

  it('are none once the plugins no longer place a command', async () => {
    let answer = list(menu('atlas.open'))
    const declared = useDeclaredCommands({ ...UNANSWERED_RUNS, list: async () => answer })
    await declared.load()

    answer = list()
    await declared.load()

    expect(declared.slots.value).toEqual([])
  })
})
