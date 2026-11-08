/**
 * Frontend entrypoint for the Update Notifier plugin: a title bar view that announces a newer
 * build of the application and offers its download.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { updater, type UpdateApi } from './api/wire'
import { createUpdates, type Timer, type Updates } from './model/updates'
import UpdateIndicator from './UpdateIndicator.vue'
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
  set: (run, delayMs) => setTimeout(run, delayMs),
  clear: (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>),
}

const browserParts: UpdateParts = { api: updater, timer: browserTimer }

/** The component of the title bar view: the indicator over the announcement of the model. */
export function createStatusView(context: PluginContext, updates: Updates) {
  return defineComponent({
    name: 'UpdateStatusView',
    setup: () => () =>
      h(UpdateIndicator, {
        update: updates.announced.value,
        platform: context.app.platform,
        onDownload: () => showFailure(context, updates.openDownload()),
        onReleaseNotes: () => showFailure(context, updates.openNotes()),
        onDismiss: () => updates.dismiss(),
      }),
  })
}

function showFailure(context: PluginContext, opening: ReturnType<Updates['openDownload']>): void {
  opening.then(
    (result) => {
      if (!result.ok) context.notices?.show(result.error, 'error')
    },
    (error: unknown) => context.notices?.show(String(error), 'error'),
  )
}

function showMissingAddress(context: PluginContext): void {
  const address = context.getSetting<string>('endpoint')
  if (context.getSetting<string>('channel') !== 'custom' || (address ?? '').trim() !== '') return
  context.notices?.show(words.missingAddress, 'error')
}

export const activate = (context: PluginContext, parts: UpdateParts = browserParts): void => {
  const updates = createUpdates({
    api: parts.api,
    app: context.app,
    openExternal: (url) => context.openExternal(url),
    getIntervalHours: () => context.getSetting('checkIntervalHours'),
    timer: parts.timer,
  })
  context.registerSlotView({ id: UPDATE_VIEW, component: createStatusView(context, updates) })
  context.onSettingChange('checkIntervalHours', () => updates.reschedule())
  const warnOfMissingAddress = () => showMissingAddress(context)
  context.onSettingChange('endpoint', () => {
    updates.checkSoon()
    warnOfMissingAddress()
  })
  context.onSettingChange('channel', () => {
    updates.checkSoon()
    warnOfMissingAddress()
  })
  context.subscriptions.push({ dispose: () => updates.stop() })
  updates.start()
  warnOfMissingAddress()
}

export default definePlugin({
  activate,
})
