/** What the whole drawing provides to its notes: the layer over them that an open note is drawn in, and the sizes of the notes. */
import type { InjectionKey, Ref } from 'vue'
import type { NodeSizes } from '../../model/useNodeSizes'

export const LIFT_LAYER: InjectionKey<Readonly<Ref<SVGGElement | null>>> = Symbol('plex-lift-layer')

export const NODE_SIZES: InjectionKey<NodeSizes> = Symbol('plex-node-sizes')
