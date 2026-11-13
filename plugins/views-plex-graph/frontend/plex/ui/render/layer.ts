/** The layer over the notes that a note open to its full width is drawn in, which the whole drawing provides. */
import type { InjectionKey, Ref } from 'vue'

export const LIFT_LAYER: InjectionKey<Readonly<Ref<SVGGElement | null>>> = Symbol('plex-lift-layer')
