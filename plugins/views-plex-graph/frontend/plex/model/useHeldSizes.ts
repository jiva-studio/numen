/** The widened box and the headings of a note, worked out only while it is held: on, or open. */
import { computed } from 'vue'
import type { NodeHeadings } from '../lib/inside'
import type { GestureRole, PlacedNode } from '../lib/node'
import type { PlexBehavior } from './behavior'
import type { WideBox } from './sizing/expansion'
import type { NodeStateProps } from './useNodeState'
import type { SettledNodes } from './useSettledNodes'

export interface HeldSizesProps {
  readonly node: PlacedNode
  readonly gestureRole: GestureRole
  readonly wide: WideBox | null
  readonly hung: NodeHeadings | null
  readonly behavior: Pick<PlexBehavior, 'reaching'>
}

/** A size given as a prop is used as it is; otherwise it is asked of `settled`. `state` reads the same sizes the note draws. */
export function useHeldSizes(
  props: HeldSizesProps,
  settled: SettledNodes | null,
  isHeld: () => boolean,
) {
  const wide = computed(
    () => props.wide ?? (isHeld() ? (settled?.getWide(props.node) ?? null) : null),
  )
  const hung = computed(
    () => props.hung ?? (isHeld() ? (settled?.getHung(props.node) ?? null) : null),
  )

  const state: NodeStateProps = {
    get node() {
      return props.node
    },
    get gestureRole() {
      return props.gestureRole
    },
    get wide() {
      return wide.value
    },
    get hung() {
      return hung.value
    },
    get behavior() {
      return props.behavior
    },
  }

  return { wide, hung, state }
}
