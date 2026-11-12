/**
 * Frontend entrypoint for the Update Notifier plugin: a title bar view that announces a newer
 * build of the application and offers its download.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext, type Caught } from '@numen/plugin-sdk'
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
export function createStatusView(context: Pick<PluginContext, 'plugin' | 'ui'>, updates: Updates) {
  return defineComponent({
    name: 'UpdateStatusView',
    setup: () => () =>
      h(UpdateIndicator, {
        version: context.plugin.app.version,
        update: updates.announced.value,
        platform: context.plugin.app.platform,
        onDownload: () => showFailure(context, updates.announcement.openDownload()),
        onReleaseNotes: () => showFailure(context, updates.announcement.openNotes()),
        onDismiss: () => updates.announcement.dismiss(),
      }),
  })
}

function showFailure(
  context: Pick<PluginContext, 'ui'>,
  opening: ReturnType<Updates['announcement']['openDownload']>,
): void {
  opening.then(
    (result) => {
      if (!result.ok) context.ui.notices.show(result.error, 'error')
    },
    (error: Caught) => context.ui.notices.show(String(error), 'error'),
  )
}

function showMissingAddress(context: PluginContext): void {
  const address = context.settings.get<string>('endpoint')
  if (context.settings.get<string>('channel') !== 'custom' || (address ?? '').trim() !== '') return
  context.ui.notices.show(words.missingAddress, 'error')
}

export const activate = (context: PluginContext, parts: UpdateParts = browserParts): void => {
  const updates = createUpdates({
    api: parts.api,
    app: context.plugin.app,
    openExternal: (url) => context.plugin.openExternal(url),
    getIntervalHours: () => context.settings.get('checkIntervalHours'),
    timer: parts.timer,
  })
  context.register.slotView({ id: UPDATE_VIEW, component: createStatusView(context, updates) })
  context.settings.onChange('checkIntervalHours', () => updates.schedule.reschedule())
  const warnOfMissingAddress = () => showMissingAddress(context)
  context.settings.onChange('endpoint', () => {
    updates.schedule.checkSoon()
    warnOfMissingAddress()
  })
  context.settings.onChange('channel', () => {
    updates.schedule.checkSoon()
    warnOfMissingAddress()
  })
  context.plugin.subscriptions.push({ dispose: () => updates.schedule.stop() })
  updates.schedule.start()
  warnOfMissingAddress()
}

export default definePlugin({
  activate,
})
