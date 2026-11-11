import { createRefRegistry } from '@numen/ui'

/** The drawn views of the nodes, by node id, held to move the keyboard focus to one of them. */
export function useNodeViews() {
  const views = new Map<string, { focus: () => void }>()
  const { getRef: holdNode } = createRefRegistry<string>((id, view) => {
    const focus = view ? Reflect.get(view, 'focus') : undefined
    if (view && typeof focus === 'function') views.set(id, { focus: () => focus.call(view) })
    else views.delete(id)
  })

  const focusNode = (id: string): void => views.get(id)?.focus()

  return { holdNode, focusNode }
}
