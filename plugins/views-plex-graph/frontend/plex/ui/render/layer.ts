/** What the whole drawing provides to its notes: the layer over them that an open note is drawn in, and the sizes they settle to. */
import type { InjectionKey, Ref } from 'vue'
import type { NodeHeadings } from '../../lib/inside'
import type { PlacedNode } from '../../lib/node'
import type { WideBox } from '../../model/sizing/expansion'

export const LIFT_LAYER: InjectionKey<Readonly<Ref<SVGGElement | null>>> = Symbol('plex-lift-layer')

/** What each note of the frame is settled to, worked out for a note when it asks. */
export interface SettledSource {
  readonly getWide: (node: PlacedNode) => WideBox | null
  readonly getHung: (node: PlacedNode) => NodeHeadings | null
}

export const SETTLED_NODES: InjectionKey<SettledSource> = Symbol('plex-settled-nodes')
