/**
 * The list of items with check marks that a plugin opens in the palette: the options it gave, the
 * rows as the person sees them, and what a toggle, a created item and a close do to them.
 */
import { computed, shallowRef, type ComputedRef } from 'vue'
import type { PaletteMultiProps } from '@numen/workbench'
import type { PickerHost, PickerItem, PickerOptions } from '@numen/plugin-loader'
import { createQuestionSession, type QuestionSession } from './createQuestionSession'

export interface PickerSession {
  readonly open: PickerHost['open']
  readonly ask: PickerHost['ask']
  readonly choose: PickerHost['choose']
  readonly question: QuestionSession['view']
  readonly multi: ComputedRef<PaletteMultiProps | undefined>
  readonly toggle: (id: string) => Promise<void>
  readonly create: (text: string) => Promise<void>
  readonly dismiss: () => void
}

export function createPickerSession(): PickerSession {
  const options = shallowRef<PickerOptions | undefined>(undefined)
  const items = shallowRef<readonly PickerItem[]>([])
  let settle: (() => void) | undefined
  const questions = createQuestionSession()

  const dismiss = (): void => {
    questions.dismiss()
    options.value = undefined
    items.value = []
    settle?.()
    settle = undefined
  }

  const open: PickerHost['open'] = (given) => {
    dismiss()
    options.value = given
    items.value = given.items
    return new Promise<void>((resolve) => {
      settle = resolve
    })
  }

  const multi = computed<PaletteMultiProps | undefined>(() => {
    const current = options.value
    if (!current) return undefined
    return {
      title: current.title,
      placeholder: current.placeholder,
      items: items.value,
      recent: current.recent,
      getCreateTitle: current.getCreateTitle,
      ...(current.getCreateColor ? { getCreateColor: current.getCreateColor } : {}),
    }
  })

  const replace = (item: PickerItem): void => {
    items.value = items.value.some((one) => one.id === item.id)
      ? items.value.map((one) => (one.id === item.id ? item : one))
      : [...items.value, item]
  }

  const toggle = async (id: string): Promise<void> => {
    const current = options.value
    if (current) replace(await current.onToggle(id))
  }

  const create = async (text: string): Promise<void> => {
    const current = options.value
    if (current) replace(await current.onCreate(text))
  }

  return {
    open,
    ask: questions.ask,
    choose: questions.choose,
    question: questions.view,
    multi,
    toggle,
    create,
    dismiss,
  }
}
