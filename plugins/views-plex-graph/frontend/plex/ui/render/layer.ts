/** What the whole drawing provides to its notes: the layer over them that an open note is drawn in, and the sizes they settle to. */
import type { InjectionKey, Ref } from 'vue'
import type { SettledNodes } from '../../model/useSettledNodes'


export const LIFT_LAYER: InjectionKey<Readonly<Ref<SVGGElement | null>>> = Symbol('plex-lift-layer')

export const SETTLED_NODES: InjectionKey<SettledNodes> = Symbol('plex-settled-nodes')
