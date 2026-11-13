/** What a node is drawn as, from the node, its role in a gesture and whether it is attended to. */
import { computed, type Ref } from 'vue'
import type { NodeHeadings } from '../lib/inside'
import type { GestureRole, PlacedNode } from '../lib/node'
import type { PlexBehavior } from './behavior'
import type { WideBox } from './sizing/expansion'

export interface NodeStateProps {
  readonly node: PlacedNode
  readonly gestureRole: GestureRole
  readonly wide: WideBox | null
  readonly hung: NodeHeadings | null
  readonly behavior: Pick<PlexBehavior, 'reaching'>
}

/** `isOn` is whether the pointer is over the node, or keyboard focus is visibly on it. */
export function useNodeState(
  props: NodeStateProps,
  state: { readonly isGhost: Readonly<Ref<boolean>>; readonly isOn: Readonly<Ref<boolean>> },
) {
  const { isGhost, isOn } = state

  /** Whether the handle is shown: while a gesture begins at the node, or while it is attended to. */
  const isHandleShown = computed(
    () =>
      props.behavior.reaching.hasHandle &&
      props.node.opacity >= 1 &&
      (props.gestureRole === 'source' || (props.gestureRole === 'open' && isOn.value)),
  )

  /** A node that is not there yet is announced as nothing; the focus is a picture. */
  const role = computed(() => {
    if (isGhost.value) return undefined
    return props.node.relation === 'focus' ? 'img' : 'button'
  })

  /** The position of the node while it is attended to and can open; null in every other state. */
  const openPosition = computed(() =>
    (props.wide || props.hung) &&
    props.node.opacity >= 1 &&
    props.gestureRole === 'open' &&
    isOn.value
      ? `${props.node.x} ${props.node.y}`
      : null,
  )

  /** One hue per relation, from a token named after it. */
  const hue = computed(() => ({
    '--relation-hue': `var(--numen-relation-${props.node.relation})`,
  }))

  return { isHandleShown, role, openPosition, hue }
}
