import { createRefRegistry } from '@numen/ui'

const isFocusable = <T>(view: T): view is T & { focus: () => void } =>
  typeof view === 'object' && view !== null && 'focus' in view && typeof view.focus === 'function'

/** The drawn views of the nodes, by node id, held to move the keyboard focus to one of them. */
export function useNodeViews() {
  const views = new Map<string, { focus: () => void }>()
  const { getRef: holdNode } = createRefRegistry<string>((id, view) => {
    if (isFocusable(view)) views.set(id, view)
    else views.delete(id)
  })

  const focusNode = (id: string): void => views.get(id)?.focus()

  return { holdNode, focusNode }
}
