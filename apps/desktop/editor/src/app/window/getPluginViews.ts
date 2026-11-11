import { SLOT, type PluginHost } from '@numen/plugin-loader'

/** The views the plugins put in the title bar and the status bar of the window. */
export const getPluginViews = (pluginHost: Pick<PluginHost, 'slotViews'>) => ({
  titleBarView: pluginHost.slotViews.getSlotView(SLOT.titleBar),
  statusBarView: pluginHost.slotViews.getSlotView(SLOT.statusBar),
})
