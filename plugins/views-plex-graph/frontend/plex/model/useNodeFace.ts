import { computed, type Ref } from 'vue'
import type { GestureRole, PlacedNode, Position } from '../lib/node'
import type { ExpandedBox } from './sizing/expansion'

/** The props of the face a node draws, from its props and the state it works out. */
export function useNodeFace(
  props: {
    readonly node: PlacedNode
    readonly gestureRole: GestureRole
    readonly showPills: boolean
    readonly restHeight: number
  },
  state: {
    readonly box: Readonly<Ref<ExpandedBox>>
    readonly isGhost: Readonly<Ref<boolean>>
    readonly isHandleVisible: Readonly<Ref<boolean>>
    readonly handle: Readonly<Ref<Position>>
  },
) {
  return computed(() => ({
    node: props.node,
    box: state.box.value,
    isGhost: state.isGhost.value,
    isTarget: props.gestureRole === 'target',
    showPills: props.showPills,
    restHeight: props.restHeight,
    isHandleVisible: state.isHandleVisible.value,
    handle: state.handle.value,
  }))
}

/** The classes of the group of a node: its relation, its role in a gesture, and whether the pointer is on it. */
export function useNodeClasses(
  props: { readonly node: PlacedNode; readonly gestureRole: GestureRole },
  isPointerOver: Readonly<Ref<boolean>>,
) {
  return computed(() => [
    `plex__node--${props.node.relation}`,
    `plex__node--${props.gestureRole}`,
    { 'plex__node--pointer': isPointerOver.value },
  ])
}
