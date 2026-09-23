/**
 * How much room the plex has, and everything measured against it: the options
 * the whole picture is drawn to, how wide a box opens, and the parts a node
 * hangs under it.
 *
 * Text is measured where the plex is drawn, so what the drawing is handed is
 * already worked out.
 */
import { computed, onMounted, onScopeDispose, ref, type ShallowRef } from 'vue'
import { getWideBox } from './dwell'
import { useTitleWidths } from './measure'
import { getNodeParts, type PlexPart } from '../lib/inside'
import { resolveOptions, type PlexOptionsInput, type Size } from '../lib/arrange'
import type { PlacedNode } from '../lib/node'
import type { Viewport } from '@/shared/lib/viewport'

/** What the room is taken to be until it has been measured. */
const FALLBACK = { width: 1200, height: 800 }

/** What the measuring reads off the plex it belongs to. */
export interface PlexRoomProps {
  readonly options?: PlexOptionsInput | undefined
  readonly viewport: Viewport
  readonly parts?: ((id: string) => readonly PlexPart[]) | undefined
}

/** `hasIcon` says whether a caller draws anything beside a title. */
export function useRoom(
  element: Readonly<ShallowRef<HTMLElement | null>>,
  props: PlexRoomProps,
  hasIcon: () => boolean,
) {
  /** How much room the plex has, as it was last measured. */
  const room = ref<Size>(FALLBACK)

  onMounted(() => {
    const found = element.value
    if (!found) return

    onScopeDispose(
      props.viewport.watch(found, (size) => {
        room.value = size
      }),
    )
  })

  /** Settled once and read by the measuring, the gesture and the drawing. */
  const options = computed(() => resolveOptions({ ...props.options, viewport: room.value }))

  /**
   * Room for the icon a caller draws beside a title, and none where the slot is
   * not filled.
   */
  const iconRoom = computed(() => (hasIcon() ? options.value.iconWidth : 0))

  /**
   * How wide each title needs its box to be, and each label its line, measured
   * against the type the theme is written in. Taken before the first arrangement,
   * so a box is drawn at the size it keeps, and taken again when a theme changes
   * that type.
   */
  const measures = useTitleWidths(() => iconRoom.value)

  /**
   * How wide a box is drawn while the attention rests on it: the room its whole
   * title asks for, held inside the window.
   *
   * A title measured at no more than the box it is already in widens nothing,
   * and where there was nothing to measure the text with, nothing widens at all.
   */
  const widen = computed(() => {
    const measure = measures.value?.node
    if (!measure) return undefined

    const { margin } = options.value
    const within = room.value
    return (node: PlacedNode) => getWideBox(node, measure(node), within, margin)
  })

  /**
   * The parts each node hangs under its box. The sizes are the ones the whole
   * picture is drawn to, so a plex set larger hangs them larger.
   */
  const hung = computed(() => {
    const held = props.parts
    if (!held) return undefined

    const { margin } = options.value
    const deps = { measure: measures.value?.part, viewport: room.value, margin }
    return (node: PlacedNode) => getNodeParts(node, held(node.id), options.value, deps)
  })

  return { room, options, measures, widen, hung }
}
