/**
 * Frontend entrypoint for the Update Notifier plugin: a title bar view that announces a newer
 * build of the application and offers its download.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { updater, type UpdateApi } from './api/wire'
import { createUpdates, type Timer, type Updates } from './model/updates'
import UpdateIndicator from './ui/UpdateIndicator.vue'
import { WORDS as words } from './words'
import './styles.css'

export { UpdateIndicator }

/** The id the manifest declares the title bar view under. */
export const UPDATE_VIEW = 'update'

/** What the plugin is made of outside the window, replaced by a test. */
export interface UpdateParts {
  readonly api: UpdateApi
  readonly timer: Timer
}

const browserTimer: Timer = {
  set: (run, delayMs) => window.setTimeout(run, delayMs),
  clear: (handle) => window.clearTimeout(handle),
}

const browserParts: UpdateParts = { api: updater, timer: browserTimer }

/** The component of the title bar view: the indicator over the announcement of the model. */
export function createStatusView(
  plugin: Pick<PluginContext, 'identity'>,
  host: Pick<HostServices, 'ui'>,
  updateModel: Updates,
) {
  return defineComponent({
    name: 'UpdateStatusView',
    setup: () => () =>
      h(UpdateIndicator, {
        version: plugin.identity.app.version,
        update: updateModel.announced.value,
        platform: plugin.identity.app.platform,
        onDownload: () => showFailure(host, updateModel.announcement.openDownload()),
        onReleaseNotes: () => showFailure(host, updateModel.announcement.openNotes()),
        onDismiss: () => updateModel.announcement.dismiss(),
      }),
  })
}

function showFailure(
  host: Pick<HostServices, 'ui'>,
  opening: ReturnType<Updates['announcement']['openDownload']>,
): void {
  opening.then(
    (result) => {
      if (!result.ok) host.ui.notices.show(result.error.message, 'error')
    },
    (error: unknown) => host.ui.notices.show(String(error), 'error'),
  )
}

function showMissingAddress(
  plugin: Pick<PluginContext, 'settings'>,
  host: Pick<HostServices, 'ui'>,
): void {
  const address = plugin.settings.get<string>('endpoint')
  if (plugin.settings.get<string>('channel') !== 'custom' || (address ?? '').trim() !== '') return
  host.ui.notices.show(words.missingAddress, 'error')
}

export const activate = (
  plugin: PluginContext,
  host: HostServices,
  parts: UpdateParts = browserParts,
): void => {
  const updateModel = createUpdates({
    api: parts.api,
    app: plugin.identity.app,
    openExternal: (url) => plugin.identity.openExternal(url),
    getIntervalHours: () => plugin.settings.get('checkIntervalHours'),
    timer: parts.timer,
  })
  plugin.register.slotView({
    id: UPDATE_VIEW,
    component: createStatusView(plugin, host, updateModel),
  })
  plugin.settings.onChange('checkIntervalHours', () => updateModel.schedule.reschedule())
  const warnOfMissingAddress = () => showMissingAddress(plugin, host)
  plugin.settings.onChange('endpoint', () => {
    updateModel.schedule.checkSoon()
    warnOfMissingAddress()
  })
  plugin.settings.onChange('channel', () => {
    updateModel.schedule.checkSoon()
    warnOfMissingAddress()
  })
  plugin.identity.subscriptions.push({ dispose: () => updateModel.schedule.stop() })
  updateModel.schedule.start()
  warnOfMissingAddress()
}

export default definePlugin({
  activate,
})
