/**
 * Frontend entrypoint for the Title bar plugin: the bar at the top of each window, drawn in the
 * slot `titleBar`.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import TitleBar from './ui/TitleBar.vue'

export { TitleBar }

/** The id the manifest declares the view of the title bar under. */
export const TITLE_BAR_VIEW = 'bar'

export const activate = (context: PluginContext): void => {
  context.registerSlotView({ id: TITLE_BAR_VIEW, component: TitleBar })
}

export default definePlugin({
  activate,
})
