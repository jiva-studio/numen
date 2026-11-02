/**
 * How the review mounts a view another plugin declares, such as the conversation of its agent
 * panel. The review kind provides the context's `mountView`; where nothing provides one, nothing
 * is mounted.
 */
import type { InjectionKey } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'

export type MountView = PluginContext['mountView']

export const MOUNT_VIEW: InjectionKey<MountView> = Symbol('mountView')

/** A mount that draws nothing. */
export const mountNothing: MountView = () => ({ focus: () => undefined, dispose: () => undefined })
