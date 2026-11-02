/**
 * Frontend entrypoint for the Index stats plugin: a status bar item that counts the vectors of the
 * search index, and a popover with the details.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { statsApi, type StatsApi } from './api/wire'
import { createIndexStats, type Activity, type IndexStats, type Timer } from './model/stats'
import StatsIndicator from './StatsIndicator.vue'
import './styles.css'

export { StatsIndicator }

/** The id the manifest declares the status bar view under. */
export const STATS_VIEW = 'stats'

/** What the plugin is made of outside the window, replaced by a test. */
export interface StatsParts {
  readonly api: StatsApi
  readonly timer: Timer
  readonly activity: Activity
}

const browserTimer: Timer = {
  set: (run, delayMs) => setTimeout(run, delayMs),
  clear: (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>),
}

/** The window is in use while it is visible and has the focus. */
const browserActivity: Activity = {
  isActive: () => document.visibilityState === 'visible' && document.hasFocus(),
  onActive: (run) => {
    const onChange = () => {
      if (browserActivity.isActive()) run()
    }
    window.addEventListener('focus', onChange)
    document.addEventListener('visibilitychange', onChange)
    return () => {
      window.removeEventListener('focus', onChange)
      document.removeEventListener('visibilitychange', onChange)
    }
  },
}

/** The component of the status bar view: the indicator over the stats of the model. */
export function createStatusView(model: IndexStats) {
  return defineComponent({
    name: 'IndexStatsStatusView',
    setup: () => () =>
      h(StatsIndicator, { stats: model.stats.value, phase: model.phase.value, locale: undefined }),
  })
}

export const activate = (
  context: PluginContext,
  parts: StatsParts = { api: statsApi, timer: browserTimer, activity: browserActivity },
): void => {
  const model = createIndexStats(parts)
  context.registerStatusView({ id: STATS_VIEW, component: createStatusView(model) })
  context.subscriptions.push({ dispose: () => model.stop() })
  model.start()
}

export default definePlugin({
  activate,
})
