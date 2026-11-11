import { computed, nextTick, ref, useId, watch, type ComponentPublicInstance, type Ref } from 'vue'
import { reportFailure, runInBackground, type Unvalidated } from '@numen/ui'
import { findKeptAction, placeActions } from '../lib/actions'
import { getSteppedIndex, type PaletteAction } from '../lib/item'
import { isActionsChord } from '../lib/keys'

/** A row standing now, which carries the element it is drawn as. */
interface RowHandle {
  readonly element: HTMLElement | null
}

const isRowHandle = (value: Unvalidated): value is RowHandle =>
  typeof value === 'object' && value !== null && 'element' in value

/** The state and handlers of the action panel: its search, the action it is on, and its keyboard. */
export function useActionPanel(options: {
  getActions: () => readonly PaletteAction[]
  open: Ref<boolean>
  searchInput: Readonly<Ref<HTMLInputElement | null>>
  choose: (action: string) => void
}) {
  const { getActions, open, searchInput, choose } = options

  const uid = useId()

  /** What is typed here, which narrows the actions and nothing else. */
  const searchQuery = ref('')

  /** The action the panel is on, by its identity. */
  const activeActionId = ref('')

  const actions = computed(() => placeActions(getActions(), searchQuery.value))

  /** Which row that action relies on, counted over the actions the words left. */
  const here = computed(() =>
    actions.value.findIndex((one) => one.action.id === activeActionId.value),
  )

  /**
   * The last pointer position. Only a pointer that has moved highlights the row
   * under it: a list scrolling under a stationary pointer also reports a move.
   */
  let pointerPosition = { x: -1, y: -1 }

  /** The rows as rendered, keyed by the action each represents. */
  const actionRows = new Map<string, RowHandle>()

  /** A fresh list keeps the action the panel was on, wherever the words put it. */
  watch(actions, (now) => {
    if (open.value) goTo(findKeptAction(now, activeActionId.value))
  })

  // It opens on the first action there is, with an empty field and the keyboard
  // in it, and leaves nothing standing behind it.
  watch(
    open,
    async (now) => {
      if (!now) {
        activeActionId.value = ''
        return
      }
      searchQuery.value = ''
      goTo(findKeptAction(actions.value, ''))
      await nextTick()
      searchInput.value?.focus()
    },
    { immediate: true },
  )

  function onRowPointerMove(to: number, event: PointerEvent): void {
    if (event.clientX === pointerPosition.x && event.clientY === pointerPosition.y) return
    pointerPosition = { x: event.clientX, y: event.clientY }
    goTo(to)
  }

  function onChooseAction(to: number): void {
    const selectedAction = actions.value[to]
    if (!selectedAction) return
    choose(selectedAction.action.id)
    open.value = false
  }

  function onKey(event: KeyboardEvent): void {
    const onStep = (delta: number, from = here.value) => {
      event.preventDefault()
      goTo(getSteppedIndex(actions.value.length, from, delta))
      runInBackground(reveal(), reportFailure)
    }
    if (isActionsChord(event) || event.key === 'Escape') {
      event.preventDefault()
      open.value = false
    } else if (event.key === 'ArrowDown') onStep(1)
    else if (event.key === 'ArrowUp') onStep(-1)
    else if (event.key === 'Home') onStep(1, -1)
    else if (event.key === 'End') onStep(-1, 0)
    else if (event.key === 'Enter') {
      event.preventDefault()
      onChooseAction(here.value)
    }
    // Tab is blocked so keyboard focus stays in this field while the panel is open.
    else if (event.key === 'Tab') event.preventDefault()
  }

  function getActionName(pos: number): string {
    return `${uid}-action-${pos}`
  }

  function goTo(to: number): void {
    activeActionId.value = actions.value[to]?.action.id ?? ''
  }

  async function reveal(): Promise<void> {
    await nextTick()
    actionRows.get(activeActionId.value)?.element?.scrollIntoView({ block: 'nearest' })
  }

  function holdRow(action: string, row: Element | ComponentPublicInstance | null): void {
    if (isRowHandle(row)) actionRows.set(action, row)
    else actionRows.delete(action)
  }

  return {
    uid,
    searchQuery,
    actions,
    here,
    onRowPointerMove,
    onChooseAction,
    onKey,
    rows: { getActionName, holdRow },
  }
}
