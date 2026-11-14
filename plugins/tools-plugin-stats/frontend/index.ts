/**
 * Frontend entrypoint for the Running plugins plugin: a title bar item that counts the plugins that
 * run, and a popover that lists them with their health, their times and their stop for idleness.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { getMonotonicNow } from '@numen/ui'
import {
  createRunningPlugins,
  type MonotonicClock,
  type RunningPlugins,
  type Ticker,
} from './model/plugins'
import PluginsIndicator from './ui/PluginsIndicator.vue'
import './styles.css'

/** The id the manifest declares the title bar view under. */
export const PLUGINS_VIEW = 'plugins'

/** What the plugin is made of outside the window, replaced by a test. */
export interface PluginsParts {
  readonly clock: MonotonicClock
  readonly ticker: Ticker
}

const browserParts: PluginsParts = {
  clock: { now: getMonotonicNow },
  ticker: {
    start: (run, intervalMs) => window.setInterval(run, intervalMs),
    stop: (handle) => window.clearInterval(handle),
  },
}

/** The component of the title bar view: the indicator over the list and the time of the model. */
export function createStatusView(model: RunningPlugins) {
  return defineComponent({
    name: 'RunningPluginsStatusView',
    setup: () => () =>
      h(PluginsIndicator, {
        processes: model.processes.value,
        phase: model.phase.value,
        now: model.now.value,
        locale: undefined,
      }),
  })
}

export const activate = (
  plugin: PluginContext,
  host: HostServices,
  parts: PluginsParts = browserParts,
): void => {
  const model = createRunningPlugins({ host: host.work.processes, ...parts })
  plugin.register.slotView({ id: PLUGINS_VIEW, component: createStatusView(model) })
  plugin.identity.subscriptions.push({ dispose: () => model.stop() })
  model.start()
}

export default definePlugin({
  activate,
})
