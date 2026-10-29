/**
 * The steps a command asks for stand one above another: each step opens with
 * its own field and prompt, going back takes the last one off, and what is
 * chosen on a step is carried out only where it answers that step.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { WORDS as words } from '@/shared/words'
import { EXACT, YES, type PendingStep } from '../../lib/step/step'
import type { Command, CommandTarget, PromptStep, StepGroup } from '../../types'
import type { NameMatch } from '../search/search'
import { createPaletteSteps } from './navigation'

const on: CommandTarget = {
  tab: 'tab-1',
  kind: 'note',
  path: 'Llama.md',
  title: 'Llama',
  file: 'Llama.md',
  vault: { id: 'v1', name: 'Zoo' },
  isReady: true,
}

const command = (overrides: Partial<Command> = {}): Command => ({
  id: 'rename',
  text: 'Rename',
  group: 'note',
  isAvailable: () => true,
  ...overrides,
})

const pending = (step: PromptStep, overrides: Partial<PendingStep> = {}): PendingStep => ({
  step,
  command: command(),
  on,
  ...overrides,
})

const createSteps = (groups: readonly StepGroup[] = []) => {
  const text = ref('')
  const happened: string[] = []
  const invoked: [string, CommandTarget, string | undefined][] = []
  const steps = createPaletteSteps({
    words,
    lists: { getStepGroups: () => groups, previewItem: () => {} },
    knows: { getTitle: () => '', getTabAt: () => null },
    text,
    onDrop: () => void happened.push('drop'),
    onLights: (item) => void happened.push(`lights ${item}`),
    onLists: () => void happened.push('lists'),
    onClose: () => void happened.push('close'),
    invocation: (id, target, name) => {
      invoked.push([id, target, name])
      return { id } as never
    },
  })
  return { steps, text, happened, invoked }
}

const match = (path: string, title: string) => ({ path, title }) as NameMatch

describe('the palette at its list of commands', () => {
  it('relies on no step, and asks for a command', () => {
    const { steps } = createSteps()

    expect(steps.steps.value).toStrictEqual([])
    expect(steps.here.value).toBeNull()
    expect(steps.placeholder.value).toBe(words.typeCommand)
    expect(steps.step.value).toBe('0::commands')
  })
})

describe('a step taken', () => {
  it('lets go of what was found, and is named by its depth, command and step', () => {
    const { steps, happened } = createSteps()

    steps.pushStep(pending('naming'))

    expect(happened).toStrictEqual(['drop'])
    expect(steps.step.value).toBe('1:rename:naming')
  })

  it('asks for the list of vaults only on the step that picks one', () => {
    const { steps, happened } = createSteps()

    steps.pushStep(pending('vaults'))

    expect(happened).toStrictEqual(['drop', 'lists'])
  })

  it('asks for the name back in plain words where the command gives none', () => {
    const { steps } = createSteps()

    steps.pushStep(pending('exactly'))
    expect(steps.placeholder.value).toBe(words.typeBack)
  })

  it('is taken back with what was found and lit release', () => {
    const { steps, happened, text } = createSteps()
    steps.pushStep(pending('naming'))
    text.value = 'Alpaca'

    steps.popStep()

    expect(happened).toStrictEqual(['drop', 'lights ', 'drop'])
    expect(text.value).toBe('')
    expect(steps.here.value).toBeNull()
  })
})

describe('what a step is over', () => {
  it('is returned as a count where it is over several files', () => {
    const { steps } = createSteps()

    expect(steps.getStepLabel(pending('asking', { on: { ...on, others: ['a.md', 'b.md'] } }))).toBe(
      words.several(3),
    )
  })

  it('is returned by its title where it is over one', () => {
    const { steps } = createSteps()

    expect(steps.getStepLabel(pending('asking'))).toBe('“Llama”')
  })
})

describe('the steps when files move', () => {
  it('are left as they are where nothing moved', () => {
    const { steps } = createSteps()
    steps.pushStep(pending('naming'))
    const before = steps.steps.value

    steps.applyRenames([])
    steps.applyRenames()

    expect(steps.steps.value).toBe(before)
  })

  it('are all release on a reset', () => {
    const { steps } = createSteps()
    steps.pushStep(pending('naming'))

    steps.reset()

    expect(steps.steps.value).toStrictEqual([])
  })
})

describe('choosing on a step', () => {
  const choose = (step: PendingStep, item: string, action: string, found: NameMatch[] = []) => {
    const createdItem = createSteps()
    return {
      ...createdItem,
      chosen: () => createdItem.steps.chooseInStep(step, item, action, found, [], () => false),
    }
  }

  it('picks the note chosen among those found', () => {
    const { chosen, invoked } = choose(pending('picking'), 'b.md', '', [
      match('a.md', 'A'),
      match('b.md', 'B'),
    ])

    chosen()

    expect(invoked[0]?.[1]).toMatchObject({ path: 'b.md', title: 'B' })
  })

  it('picks nothing for a note no longer found', () => {
    const { chosen, invoked } = choose(pending('picking'), 'gone.md', '', [match('a.md', 'A')])

    expect(chosen()).toBeNull()
    expect(invoked).toStrictEqual([])
  })

  it('names with what was typed, the spaces around it taken off', () => {
    const createdItem = choose(pending('naming'), '', '')
    createdItem.text.value = '  Alpaca  '

    createdItem.chosen()

    expect(createdItem.invoked[0]?.[2]).toBe('Alpaca')
  })

  it('is carried out on yes, and on no other answer but no', () => {
    expect(choose(pending('asking'), '', 'maybe').chosen()).toBeNull()
    const { chosen, invoked } = choose(pending('asking'), '', YES)

    chosen()

    expect(invoked).toHaveLength(1)
  })

  it('takes a name typed back only as the answer that retypes it', () => {
    const inputValue = choose(pending('exactly'), '', 'maybe')
    inputValue.text.value = 'Llama'
    const exact = choose(pending('exactly'), '', EXACT)
    exact.text.value = 'Llama'

    expect(inputValue.chosen()).toBeNull()
    exact.chosen()
    expect(exact.invoked).toHaveLength(1)
  })
})

describe('the row a list opens on', () => {
  const row = (id: string, isCurrent = false) => ({ id, text: id, isCurrent }) as never
  const groups = [{ id: 'g', title: 'Themes', items: [row('light'), row('dark', true)] }]

  it('is the one standing now, on the step that chooses from a list', () => {
    const { steps } = createSteps(groups)

    steps.pushStep(pending('choosing'))

    expect(steps.opensOn.value).toBe('dark')
  })

  it('is none where no row remains now', () => {
    const { steps } = createSteps([{ id: 'g', title: 'Themes', items: [row('light')] }])

    steps.pushStep(pending('choosing'))

    expect(steps.opensOn.value).toBe('')
  })

  it('is none on any other step', () => {
    const { steps } = createSteps(groups)

    steps.pushStep(pending('naming'))

    expect(steps.opensOn.value).toBe('')
  })
})
