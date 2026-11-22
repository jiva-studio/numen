/**
 * The questions a plugin puts to the person in the palette: a line of text, or one choice among
 * options. A question is answered with a value, or left with undefined when the person dismisses it.
 */
import { computed, shallowRef, type ComputedRef } from 'vue'
import type { PaletteGroup } from '@numen/workbench'
import type { AskOptions, ChooseOptions, PickerHost } from '@numen/plugin-loader'
import { PALETTE_WORDS } from '../../words'

const ANSWER = 'answer'

interface QuestionView {
  readonly typed: string
  readonly placeholder: string
  readonly groups: readonly PaletteGroup[]
  readonly type: (text: string) => void
  readonly answer: (id: string) => void
}

export interface QuestionSession {
  readonly ask: PickerHost['ask']
  readonly choose: PickerHost['choose']
  readonly view: ComputedRef<QuestionView | undefined>
  readonly dismiss: () => void
}

type Question =
  | { readonly kind: 'ask'; readonly options: AskOptions }
  | { readonly kind: 'choose'; readonly options: ChooseOptions }

export function createQuestionSession(): QuestionSession {
  const question = shallowRef<Question | undefined>(undefined)
  const typed = shallowRef('')
  let settle: ((value: string | undefined) => void) | undefined

  const finish = (value: string | undefined): void => {
    question.value = undefined
    typed.value = ''
    settle?.(value)
    settle = undefined
  }

  const open = (next: Question, text: string): Promise<string | undefined> => {
    finish(undefined)
    question.value = next
    typed.value = text
    return new Promise((resolve) => {
      settle = resolve
    })
  }

  const getAskGroup = (options: AskOptions): PaletteGroup => {
    const value = typed.value.trim()
    const title = options.title
    return {
      id: 'question',
      title,
      items: [
        {
          id: ANSWER,
          title: value ? `${title}: “${value}”` : `${PALETTE_WORDS.leavesOut} ${title}`,
          actions: [{ id: ANSWER, text: value ? PALETTE_WORDS.givesIt : PALETTE_WORDS.leavesOut }],
        },
      ],
    }
  }

  const getChooseGroup = (options: ChooseOptions): PaletteGroup => {
    const word = typed.value.trim().toLowerCase()
    return {
      id: 'question',
      title: options.title,
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

  const type = (text: string): void => {
    typed.value = text
  }

  const answer = (id: string): void => {
    const current = question.value
    if (!current) return
    finish(current.kind === 'ask' ? typed.value.trim() : id)
  }

  const view = computed<QuestionView | undefined>(() => {
    const current = question.value
    if (!current) return undefined
    return {
      typed: typed.value,
      placeholder: current.options.title,
      groups: [
        current.kind === 'ask' ? getAskGroup(current.options) : getChooseGroup(current.options),
      ],
      type,
      answer,
    }
  })

  return {
    ask: (options) => open({ kind: 'ask', options }, options.value ?? ''),
    choose: (options) => open({ kind: 'choose', options }, ''),
    view,
    dismiss: () => finish(undefined),
  }
}
