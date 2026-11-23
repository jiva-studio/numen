import type { DropRegistry, EditorExtensionRegistry, JsonValue, SlotHost } from '@numen/plugin-sdk'
import type { DeclaredSlotEntry } from '../../slots'
import type { SlotViewRegistry } from '../../slot-views'
import type { ViewRegistry } from '../../views'
import type { ContextSettingHost, ContextTabHost, ContextViewHost, PluginHostDeps } from '../types'
import { createLicenseServiceBridge } from './services'

export function createSettingBridge({
  plugins,
}: Pick<PluginHostDeps, 'plugins'>): ContextSettingHost {
  const licenses = createLicenseServiceBridge({ plugins })
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
    hasLicense: licenses.hasLicense,
    getLicenseKey: licenses.getLicenseKey,
    watchLicense: licenses.watchLicense,
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

export function createTabBridge({ tabs }: Pick<PluginHostDeps, 'tabs'>): ContextTabHost {
  return {
    openTab: tabs.openTab,
    navigateTab: tabs.navigateTab,
    closeTab: tabs.closeTab,
  }
}

/** The registries other than the host's own, which the context bridge delegates to. */
export interface SharedRegistries {
  readonly views: ViewRegistry
  readonly drops: DropRegistry
  readonly editorExtensions: EditorExtensionRegistry
  readonly slotViews: SlotViewRegistry
  readonly slots: SlotHost
  readonly readSlotEntries: () => readonly DeclaredSlotEntry[] | undefined
}
