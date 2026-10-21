/**
 * What a press on a node evaluates to: the node chosen, the node asked for on its
 * own, or a menu on it. The hand and the keyboard arrive here as the same three
 * answers, and a node not there yet answers none of them.
 */
import { computed } from 'vue'
import { isMenuKey, isPress, isShowKey } from '../lib/keys'
import { isReachable, isStop, type GestureRole, type PlacedNode, type Position } from '../lib/node'
import type { MenuTrigger } from '@/shared/ui/menu'
import type { PlexBehaviour } from './behaviour'
import { getDestination, mergeListeners, type PlexDestination } from './showing'

/** What the node says when a press evaluates to something. */
export interface NodePressSaid {
  readonly activate: () => void
  readonly show: (showing: PlexDestination) => void
  readonly reach: (pointer: PointerEvent) => void
  readonly menu: (at: Position, trigger: MenuTrigger) => void
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
    () => canReach.value || (!isGhost.value && props.node.relation === 'focus'),
  )

  const activateNode = (): void => {
    if (canReach.value) listeners.activate()
  }

  const showNode = (hasModifier: boolean): void => {
    if (canStop.value) listeners.show(getDestination(hasModifier))
  }

  /** What this node listens for beyond the handle. */
  const listening = mergeListeners(
    props.behaviour.reaching.listeners({
      ready: () => !isGhost.value && props.gestureRole === 'open',
      reach: (event: PointerEvent) => listeners.reach(event),
    }),
    props.behaviour.showing.listeners({
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
}

interface KeyPressContext {
  readonly isGhost: boolean
  readonly middleOf: () => Position | null
  readonly onMenu: (at: Position, trigger: MenuTrigger) => void
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
