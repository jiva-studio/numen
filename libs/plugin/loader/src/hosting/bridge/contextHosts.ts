import type { DropRegistry, EditorExtensionRegistry, JsonValue, SlotHost } from '@numen/plugin-sdk'
import type { DeclaredSlotEntry } from '../../slots'
import type { SlotViewRegistry } from '../../slot-views'
import type { ViewRegistry } from '../../views'
import type { PluginStyles } from '../../styles'
import type { ContextSettingHost, ContextViewHost, PluginHostDeps } from '../types'

export function createSettingBridge({
  plugins,
}: Pick<PluginHostDeps, 'plugins'>): ContextSettingHost {
  return {
    getSetting: <T extends JsonValue = JsonValue>(pluginId: string, key: string) =>
      plugins.getSetting(pluginId, key) as T | undefined,
    watchSetting: <T extends JsonValue = JsonValue>(
      pluginId: string,
      key: string,
      handler: (value: T) => void | Promise<void>,
    ) => {
      return plugins.watchSetting(
        pluginId,
        key,
        handler as (v: JsonValue | undefined) => void | Promise<void>,
      )
    },
    getEditorSettings: plugins.getEditorSettings,
    watchEditorSettings: plugins.watchEditorSettings,
    hasLicense: plugins.hasLicense,
    getLicenseKey: plugins.getLicenseKey,
    watchLicense: plugins.watchLicense,
  }
}

export function createViewBridge(shared: SharedRegistries): ContextViewHost {
  const { views, drops, slotViews, editorExtensions } = shared
  return {
    registerSlotView: slotViews.registerSlotView,
    registerView: views.registerView,
    mountView: views.mountView,
    registerDropHandler: drops.registerDropHandler,
    drops: { canDrop: drops.canDrop, drop: drops.drop },
    registerEditorExtension: editorExtensions.registerEditorExtension,
    editorExtensions: {
      get: editorExtensions.get,
      onChange: editorExtensions.onChange,
      fail: editorExtensions.fail,
    },
  }
}

/** The registries other than the host's own, which the context bridge delegates to. */
export interface SharedRegistries {
  readonly views: ViewRegistry
  readonly drops: DropRegistry
  readonly editorExtensions: EditorExtensionRegistry
  readonly slotViews: SlotViewRegistry
  readonly slots: SlotHost
  readonly styles: PluginStyles
  readonly readSlotEntries: () => readonly DeclaredSlotEntry[] | undefined
}
