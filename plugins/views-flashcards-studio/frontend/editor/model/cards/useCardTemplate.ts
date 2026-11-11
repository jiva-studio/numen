import { computed, nextTick, shallowRef, type ComponentPublicInstance } from 'vue'
import { getPanes, insert } from '../../../lib/cards'
import type { CardModelWords, Half, Pane, TemplateRow } from '../../../lib/cards'

/** The part of a template pane the caret handling reads. */
interface PaneBox {
  readonly box: HTMLTextAreaElement | null
}

export interface CardTemplateOptions {
  readonly getTemplate: () => TemplateRow
  readonly getWords: () => CardModelWords
  readonly write: (half: Half, text: string) => void
}

/** The panes of one card template and the insertion of a field name at the caret of the aimed half. */
export function useCardTemplate(options: CardTemplateOptions) {
  const aim = shallowRef<Half | null>(null)
  const focusedHalf = computed<Half>(() => aim.value ?? 'front')
  const front = shallowRef<PaneBox | null>(null)
  const back = shallowRef<PaneBox | null>(null)
  const divided = computed<readonly Pane[]>(() =>
    getPanes(options.getTemplate(), options.getWords()),
  )

  const getBox = (half: Half): HTMLTextAreaElement | null =>
    (half === 'front' ? front.value : back.value)?.box ?? null

  const setBox = (pane: Pane, element: Element | ComponentPublicInstance | null): void => {
    if (pane.mode !== 'written') return
    const box = element as PaneBox | null
    if (pane.half === 'front') front.value = box
    else back.value = box
  }

  const putField = async (field: string): Promise<void> => {
    const half = focusedHalf.value
    const target = getBox(half)
    if (!target) return
    const caret = (aim.value === null ? null : target.selectionStart) ?? target.value.length
    const inserted = insert(target.value, caret, field)
    options.write(half, inserted.text)
    await nextTick()
    const again = getBox(half)
    again?.focus()
    again?.setSelectionRange(inserted.caret, inserted.caret)
  }

  const isFocused = (pane: Pane): boolean =>
    pane.mode === 'written' && focusedHalf.value === pane.half

  return { aim, divided, setBox, putField, isFocused }
}
