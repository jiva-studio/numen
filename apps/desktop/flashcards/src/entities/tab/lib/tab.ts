import type { EffectScope } from 'vue'
import type { TabKindRegistration } from '@numen/plugin-loader'

/** A tab kind, with the plugin that registered it. */
export interface PluginView {
  readonly pluginId: string
  readonly registration: TabKindRegistration
}

/** The one tab the window draws: a kind opened on a vault, and what the kind made of it. */
export interface WindowTab {
  readonly id: string
  readonly vault: string
  readonly view: PluginView
  readonly state: object | undefined
  readonly scope: EffectScope
}

/** The kind the window draws: the main-pane kind of the lowest order, or null where there is none. */
export function findFrontView(views: readonly PluginView[]): PluginView | null {
  const mainViews = views.filter((one) => one.registration.placement?.pane === 'main')
  mainViews.sort(
    // Stryker disable next-line OptionalChaining: mainViews only holds registrations with defined placement
    (a, b) => (a.registration.placement?.order ?? 0) - (b.registration.placement?.order ?? 0),
  )
  return mainViews[0] ?? null
}
