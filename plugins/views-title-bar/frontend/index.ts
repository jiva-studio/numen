/**
 * Frontend entrypoint for the Title bar plugin: the bar at the top of each window, drawn in the
 * slot `titleBar`.
 */
import { defineComponent, h, shallowRef } from 'vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { BUTTONS_SETTING, readButtonsSetting } from './lib/frame'
import TitleBar from './ui/TitleBar.vue'

export { TitleBar }

/** The id the manifest declares the view of the title bar under. */
export const TITLE_BAR_VIEW = 'bar'

/** The view of the slot: the bar, with the window buttons placed where the setting `buttons` says. */
export function createTitleBarView(context: PluginContext) {
  const buttons = shallowRef(readButtonsSetting(context.getSetting(BUTTONS_SETTING)))
  context.onSettingChange(BUTTONS_SETTING, (value) => {
    buttons.value = readButtonsSetting(value)
  })
  return defineComponent({
    name: 'TitleBarView',
    inheritAttrs: false,
    setup:
      (_, { attrs }) =>
      () =>
        h(TitleBar, { ...attrs, buttons: buttons.value }),
  })
}

export const activate = (context: PluginContext): void => {
  context.registerSlotView({ id: TITLE_BAR_VIEW, component: createTitleBarView(context) })
}

export default definePlugin({
  activate,
})
