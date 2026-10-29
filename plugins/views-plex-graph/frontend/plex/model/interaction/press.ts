/**
 * The result of a press on a node: the node is selected, opened on its own, or a
 * menu is shown. Pointer and keyboard input produce the same three results, and
 * a node that does not exist yet produces none.
 */
import { computed } from 'vue'
import { isMenuKey, isPress, isShowKey } from '../../lib/keys'
import {
  isReachable,
  isStop,
  type GestureRole,
  type PlacedNode,
  type Position,
} from '../../lib/node'
import type { MenuTrigger } from '@numen/ui'
import type { PlexBehaviour } from '../behaviour'
import { getDestination, mergeListeners, type PlexDestination } from './destination'

/** The callbacks a node invokes when a press resolves to an action. */
export interface NodePressReport {
  readonly activate: () => void
  readonly show: (showing: PlexDestination) => void
  readonly reach: (pointer: PointerEvent) => void
  readonly menu: (point: Position, trigger: MenuTrigger) => void
}

/** What the press reads off the node it belongs to. */
export interface NodePressProps {
  readonly node: PlacedNode
  readonly gestureRole: GestureRole
  readonly behaviour: Pick<PlexBehaviour, 'reaching' | 'showing'>
}

/**
 * `middleOf` is the middle of the node as it is drawn, for a press, which
 * carries no point of its own.
 */
export function useNodePress(
  props: NodePressProps,
  middleOf: () => Position | null,
  listeners: NodePressReport,
) {
  /** A placeholder is not a node yet, so no action applies to it and nothing is reported. */
  const isGhost = computed(() => props.gestureRole === 'ghost')

  /** One predicate: the same rule decides the click and the name. */
  const canReach = computed(() => !isGhost.value && isReachable(props.node))

  /** Where the keyboard stops: the focus too, and nothing on its way in or out. */
  const canStop = computed(() => !isGhost.value && isStop(props.node))

  /** The focus is announced although it cannot be chosen: it is where you are. */
  const isAnnounced = computed(
    () => canReach.value || (!isGhost.value && props.node.relation === 'focus'),
  )

  const activateNode = (): void => {
    if (canReach.value) listeners.activate()
  }

  const showNode = (hasModifier: boolean): void => {
    if (canStop.value) listeners.show(getDestination(hasModifier))
  }

  /** The listeners this node adds beyond the handle. */
  const combinedListeners = mergeListeners(
    props.behaviour.reaching.listeners({
      ready: () => !isGhost.value && props.gestureRole === 'open',
      reach: (event: PointerEvent) => listeners.reach(event),
    }),
    props.behaviour.showing.listeners({
      ready: () => canStop.value,
      show: (hasModifier: boolean) => showNode(hasModifier),
    }),
  )

  const state = {
    isGhost,
    canReach,
    canStop,
    isAnnounced,
    listening: combinedListeners,

    onClick: (): void => {
      activateNode()
    },

    onDoubleClick: (event: MouseEvent): void => {
      if (props.behaviour.showing.isDoubleClick) showNode(event.altKey)
    },

    onContextMenu: (event: MouseEvent): void => {
      if (isGhost.value) return
      event.preventDefault()
      listeners.menu({ x: event.clientX, y: event.clientY }, 'pointer')
    },

    onKeyDown: (event: KeyboardEvent): void => {
      handleNodeKey(event, {
        isGhost: isGhost.value,
        middleOf,
        onMenu: listeners.menu,
        onShow: showNode,
        onActivate: activateNode,
      })
    },
  }
  return state
}

interface KeyPressContext {
  readonly isGhost: boolean
  readonly middleOf: () => Position | null
  readonly onMenu: (point: Position, trigger: MenuTrigger) => void
  readonly onShow: (hasModifier: boolean) => void
  readonly onActivate: () => void
}

function handleNodeKey(event: KeyboardEvent, context: KeyPressContext): void {
  if (isMenuKey(event)) {
    if (context.isGhost) return
    event.preventDefault()
    const middle = context.middleOf()
    if (middle) context.onMenu(middle, 'keyboard')
    return
  }
  if (isShowKey(event)) {
    event.preventDefault()
    context.onShow(event.altKey)
    return
  }
  if (!isPress(event)) return
  event.preventDefault()
  context.onActivate()
}
