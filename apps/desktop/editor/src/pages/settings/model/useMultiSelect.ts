import { ref } from 'vue'
import { measureRect, type Position } from '@numen/ui'

export interface MultiSelectDeps {
  readonly getSelected: () => readonly string[]
  readonly isDisabled: () => boolean
  readonly getElement: () => HTMLElement | null
  readonly updateSelected: (selected: readonly string[]) => void
  readonly toggle: (id: string, isOn: boolean) => void
}

/** The list of a multiple choice: where it opens under the line, the search typed in it, and a choice switched on or off. */
export function useMultiSelect(deps: MultiSelectDeps) {
  const query = ref('')
  const menuAnchor = ref<{ at: Position; wide: number } | null>(null)

  const openChoices = (): void => {
    const line = deps.getElement()
    if (!line || deps.isDisabled()) return
    const box = measureRect(line)
    menuAnchor.value = { at: { x: box.left, y: box.bottom }, wide: box.width }
  }

  const closeChoices = (): void => {
    menuAnchor.value = null
    query.value = ''
  }

  const toggleChoice = (id: string): void => {
    const selected = deps.getSelected()
    const isOn = !selected.includes(id)
    deps.updateSelected(isOn ? [...selected, id] : selected.filter((one) => one !== id))
    deps.toggle(id, isOn)
  }

  return { query, menuAnchor, openChoices, closeChoices, toggleChoice }
}
