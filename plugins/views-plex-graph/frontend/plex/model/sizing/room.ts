/**
 * The space available to the plex, and everything measured against it: the
 * options the whole drawing uses, the width a box opens to, and the headings
 * of a node.
 *
 * Text is measured where the plex is drawn, so what the drawing is handed is
 * already worked out.
 */
import { computed, onMounted, onScopeDispose, ref, type ShallowRef } from 'vue'
import { getWideBox } from './expansion'
import { useTitleWidths } from './measure'
import { getNodeHeadings, type PlexHeading } from '../../lib/inside'
import { resolveOptions, type PlexOptionsInput, type Size } from '../../lib/arrange'
import type { PlacedNode } from '../../lib/node'
import type { Viewport } from '@numen/ui'

/** The assumed size of the plex until it has been measured. */
const FALLBACK = { width: 1200, height: 800 }

/** What the measuring reads off the plex it belongs to. */
export interface PlexSizingProps {
  readonly options?: PlexOptionsInput | undefined
  readonly viewport: Viewport
  readonly headings?: ((id: string) => readonly PlexHeading[]) | undefined
}

/** `hasIcon` reports whether a caller draws an icon beside a title. */
export function useFreeSpace(
  element: Readonly<ShallowRef<HTMLElement | null>>,
  props: PlexSizingProps,
  hasIcon: () => boolean,
) {
  /** The available size of the plex, as last measured. */
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
   * Space reserved for the icon a caller draws beside a title; zero when the
   * slot is not filled.
   */
  const iconWidthReserve = computed(() => (hasIcon() ? options.value.iconWidth : 0))

  /**
   * How wide each title needs its box to be, and each label its line, measured
   * against the type the theme is written in. Taken before the first arrangement,
   * so a box is drawn at the size it keeps, and taken again when a theme changes
   * that type.
   */
  const measures = useTitleWidths(() => iconWidthReserve.value)

  /**
   * The width of a box drawn while it has attention: the width its whole title
   * needs, limited to the window.
   *
   * A title measured at no more than the current box width does not widen it,
   * and when text cannot be measured, nothing widens.
   */
  const expandWidth = computed(() => {
    const measure = measures.value?.node
    if (!measure) return undefined

    const { margin } = options.value
    const bounds = { viewport: room.value, margin }
    return (node: PlacedNode) => getWideBox(node, measure(node), bounds)
  })

  /**
   * The headings each node hangs under its box. The sizes are the ones the whole
   * picture is drawn to, so a plex set larger hangs them larger.
   */
  const suspended = computed(() => {
    const getHeadings = props.headings
    if (!getHeadings) return undefined

    const { margin } = options.value
    const deps = { measure: measures.value?.heading, viewport: room.value, margin }
    return (node: PlacedNode) =>
      getNodeHeadings(node, getHeadings(node.id), { options: options.value, deps })
  })

  return { room, options, measures, widen: expandWidth, hung: suspended }
}
