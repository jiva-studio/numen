/**
 * Frontend entrypoint for the Running plugins plugin: a title bar item that counts the plugins that
 * run, and a popover that lists them with their health, their times and their stop for idleness.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { createRunningPlugins, type RunningPlugins } from './model/plugins'
import PluginsIndicator from './ui/PluginsIndicator.vue'
import './styles.css'

/** The id the manifest declares the title bar view under. */
export const PLUGINS_VIEW = 'plugins'

/** The component of the title bar view: the indicator over the list of the model. */
export function createStatusView(model: RunningPlugins) {
  return defineComponent({
    name: 'RunningPluginsStatusView',
    setup: () => () =>
      h(PluginsIndicator, {
        processes: model.processes.value,
        phase: model.phase.value,
        locale: undefined,
      }),
  })
}

export const activate = (context: PluginContext): void => {
  const model = createRunningPlugins(context.work.processes)
  context.register.slotView({ id: PLUGINS_VIEW, component: createStatusView(model) })
  context.plugin.subscriptions.push({ dispose: () => model.stop() })
  model.start()
}

export default definePlugin({
  activate,
})
