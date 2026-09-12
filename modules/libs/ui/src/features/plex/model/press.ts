/**
 * What a press on a node comes to: the node chosen, the node asked for on its
 * own, or a menu on it. The hand and the keyboard arrive here as the same three
 * answers, and a node not there yet answers none of them.
 */
import { computed } from 'vue'
import { isMenuKey, isPress, isShowKey } from '../lib/keys'
import { isReachable, isStop, type GestureRole, type PlacedNode, type Position } from '../lib/node'
import type { MenuOpening } from '@/shared/ui/menu'
import type { ReachStrategy } from './reaching'
import { getDestination, mergeListeners, type PlexDestination, type ShowStrategy } from './showing'

/** What the node says when a press comes to something. */
export interface NodePressSaid {
  readonly activate: () => void
  readonly show: (showing: PlexDestination) => void
  readonly reach: (pointer: PointerEvent) => void
  readonly menu: (at: Position, opening: MenuOpening) => void
}

/** What the press reads off the node it belongs to. */
export interface NodePressProps {
  readonly node: PlacedNode
  readonly gestureRole: GestureRole
  readonly reaching: ReachStrategy
  readonly showing: ShowStrategy
}

/**
 * `middleOf` is the middle of the node as it is drawn, for a press, which
 * carries no point of its own.
 */
export function useNodePress(
  props: NodePressProps,
  middleOf: () => Position | null,
  listeners: NodePressSaid,
) {
  /** Not a node yet, so nothing may be done to it and nothing is told about it. */
  const isGhost = computed(() => props.gestureRole === 'ghost')

  /** One predicate: the same rule decides the click and the name. */
  const canReach = computed(() => !isGhost.value && isReachable(props.node))

  /** Where the keyboard stops: the focus too, and nothing on its way in or out. */
  const canStop = computed(() => !isGhost.value && isStop(props.node))

  /** The focus is announced although it cannot be chosen: it is where you are. */
  const isAnnounced = computed(
    () => canReach.value || (!isGhost.value && props.node.seat === 'focus'),
  )

  const activateNode = (): void => {
    if (canReach.value) listeners.activate()
  }

  const showNode = (hasModifier: boolean): void => {
    if (canStop.value) listeners.show(getDestination(hasModifier))
  }

  /** What this node listens for beyond the handle. */
  const listening = mergeListeners(
    props.reaching.listeners({
      ready: () => !isGhost.value && props.gestureRole === 'open',
      reach: (event: PointerEvent) => listeners.reach(event),
    }),
    props.showing.listeners({
      ready: () => canStop.value,
      show: (hasModifier: boolean) => showNode(hasModifier),
    }),
  )

  return {
    isGhost,
    canReach,
    canStop,
    isAnnounced,
    listening,

    onClick: (): void => {
      activateNode()
    },

    onDoubleClick: (event: MouseEvent): void => {
      if (props.showing.doubleClick) showNode(event.altKey)
    },

    onContextMenu: (event: MouseEvent): void => {
      if (isGhost.value) return
      event.preventDefault()
      listeners.menu({ x: event.clientX, y: event.clientY }, 'pointer')
    },

    onKeyDown: (event: KeyboardEvent): void => {
      if (isMenuKey(event)) {
        if (isGhost.value) return
        event.preventDefault()
        const middle = middleOf()
        if (middle) listeners.menu(middle, 'keyboard')
        return
      }
      if (isShowKey(event)) {
        event.preventDefault()
        showNode(event.altKey)
        return
      }
      if (!isPress(event)) return
      event.preventDefault()
      activateNode()
    },
  }
}
