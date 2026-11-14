/** The expanded box and the headings of a note, worked out only while it is expanded: on, or open. */
import { computed } from 'vue'
import type { NodeHeadings } from '../lib/inside'
import type { GestureRole, PlacedNode } from '../lib/node'
import type { PlexBehavior } from './behavior'
import type { ExpandedBox } from './sizing/expansion'
import type { NodeSizes } from './useNodeSizes'
import type { NodeStateProps } from './useNodeState'

export interface ExpandedSizesProps {
  readonly node: PlacedNode
  readonly gestureRole: GestureRole
  readonly expanded: ExpandedBox | null
  readonly placedHeadings: NodeHeadings | null
  readonly behavior: Pick<PlexBehavior, 'reaching'>
}

/** A size given as a prop is used as it is; otherwise it is requested from `sizes`. `state` reads the same sizes the note draws. */
export function useExpandedSizes(
  props: ExpandedSizesProps,
  sizes: NodeSizes | null,
  isExpanded: () => boolean,
) {
  const expanded = computed(
    () => props.expanded ?? (isExpanded() ? (sizes?.getExpandedBoxOf(props.node) ?? null) : null),
  )
  const placedHeadings = computed(
    () =>
      props.placedHeadings ??
      (isExpanded() ? (sizes?.getPlacedHeadingsOf(props.node) ?? null) : null),
  )

  const state: NodeStateProps = {
    get node() {
      return props.node
    },
    get gestureRole() {
      return props.gestureRole
    },
    get expanded() {
      return expanded.value
    },
    get placedHeadings() {
      return placedHeadings.value
    },
    get behavior() {
      return props.behavior
    },
  }

  return { expanded, placedHeadings, state }
}
