/**
 * The questions a plugin puts to the person in the palette: a line of text, or one choice among
 * options. A question is answered with a value, or left with undefined when the person dismisses it.
 */
import { computed, shallowRef, type ComputedRef } from 'vue'
import type { PaletteGroup } from '@numen/workbench'
import type { PromptTextOptions, SelectOptionOptions, PickerHost } from '@numen/plugin-loader'
import { PALETTE_WORDS } from '../../words'

interface PromptView {
  readonly step: string
  readonly typed: string
  readonly placeholder: string
  readonly heading: string
  readonly groups: readonly PaletteGroup[]
  readonly enterText: (text: string) => void
  readonly answer: (id: string) => void
  readonly submit: () => void
}

export interface PromptSession {
  readonly promptText: PickerHost['promptText']
  readonly selectOption: PickerHost['selectOption']
  readonly view: ComputedRef<PromptView | undefined>
  readonly dismiss: () => void
}

type Question =
  | { readonly kind: 'promptText'; readonly options: PromptTextOptions }
  | { readonly kind: 'selectOption'; readonly options: SelectOptionOptions }

export function createPromptSession(): PromptSession {
  const question = shallowRef<Question | undefined>(undefined)
  const typed = shallowRef('')
  const step = shallowRef('')
  let settle: ((value: string | undefined) => void) | undefined
  let isAnswered = false
  let count = 0

  const finish = (value: string | undefined): void => {
    question.value = undefined
    typed.value = ''
    isAnswered = false
    settle?.(value)
    settle = undefined
  }

  const open = (next: Question, text: string): Promise<string | undefined> => {
    finish(undefined)
    count += 1
    step.value = `question-${count}`
    question.value = next
    typed.value = text
    return new Promise((resolve) => {
      settle = resolve
    })
  }

  const getOptionGroup = (options: SelectOptionOptions): PaletteGroup => {
    const word = typed.value.trim().toLowerCase()
    return {
      id: 'question',
      title: '',
      items: options.options
        .filter((one) => one.title.toLowerCase().includes(word))
        .map((one) => ({
          id: one.value,
          title: one.title,
          actions: [{ id: one.value, text: PALETTE_WORDS.chooses }],
        })),
      silence: PALETTE_WORDS.noneFound,
    }
  }

  const enterText = (text: string): void => {
    typed.value = text
  }

  /**
   * The answer settles at once; the palette stays up for a turn of the clock so that a question
   * asked right after it replaces this one without the palette closing and opening again.
   */
  const reply = (value: string): void => {
    if (!question.value || isAnswered) return
    isAnswered = true
    settle?.(value)
    settle = undefined
    setTimeout(() => {
      if (isAnswered) finish(undefined)
    }, 0)
  }

  const view = computed<PromptView | undefined>(() => {
    const current = question.value
    if (!current) return undefined
    return {
      step: step.value,
      typed: typed.value,
      placeholder:
        current.kind === 'selectOption' ? PALETTE_WORDS.typeItems : current.options.title,
      heading: current.options.title,
      groups: current.kind === 'promptText' ? [] : [getOptionGroup(current.options)],
      enterText,
      answer: (id) => {
        if (current.kind === 'selectOption') reply(id)
      },
      submit: () => {
        if (current.kind === 'promptText') reply(typed.value.trim())
      },
    }
  })

  return {
    promptText: (options) => open({ kind: 'promptText', options }, options.value ?? ''),
    selectOption: (options) => open({ kind: 'selectOption', options }, ''),
    view,
    dismiss: () => finish(undefined),
  }
}
