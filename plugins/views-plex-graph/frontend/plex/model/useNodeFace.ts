import { computed, type ComputedRef, type Ref } from 'vue'
import type { PlexNodeFaceProps, PlexNodeProps } from '../ui/render/props'
import type { ExpandedBox } from './sizing/expansion'
import type { Position } from '../lib/node'

/** What a node's face is drawn from: the props of the node and the state it works out. */
export function useNodeFace(
  props: Pick<PlexNodeProps, 'node' | 'gestureRole' | 'showPills' | 'restHeight'>,
  state: {
    readonly box: Readonly<Ref<ExpandedBox>>
    readonly isGhost: Readonly<Ref<boolean>>
    readonly isHandleVisible: Readonly<Ref<boolean>>
    readonly handle: Readonly<Ref<Position>>
  },
): ComputedRef<PlexNodeFaceProps> {
  return computed(() => ({
    node: props.node,
    box: state.box.value,
    isGhost: state.isGhost.value,
    isTarget: props.gestureRole === 'target',
    showPills: props.showPills ?? true,
    restHeight: props.restHeight ?? 0,
    isHandleVisible: state.isHandleVisible.value,
    handle: state.handle.value,
  }))
}
