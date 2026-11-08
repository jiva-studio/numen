/** The vault settings operations the window uses. */
import type { Configuration, SettingEdit } from '@/entities/settings'

export interface SettingsPort {
  /** Every setting currently, and the models the settings offer. */
  getSettings(): Promise<Configuration>
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
  /**
   * The settings that changed, until the signal aborts: a write by this window, another window, a
   * plugin or the palette, and an edit of the file by hand. Each batch holds the settings that
   * changed together, named by their declared path, with the value each now holds as JSON. The
   * stream opens with an empty batch, which the window ignores.
   */
  watchSettings(signal: AbortSignal): AsyncIterable<readonly SettingEdit[]>
  /**
   * The secrets shared between plugins. A value is stored in the system keychain, or in a
   * protected file where there is none, and is never read back or written to the settings file.
   * A machine that can store neither rejects the call.
   */
  createSecret(group: string, name: string, value: string): Promise<string>
  renameSecret(id: string, name: string): Promise<void>
  writeSecret(id: string, value: string): Promise<void>
  /** A secret removed with its value and its permissions. */
  deleteSecret(id: string): Promise<void>
  /** Every secret of a group removed with its values and its permissions. */
  deleteSecretGroup(group: string): Promise<void>
  /**
   * For each secret id, whether its value is on this machine. The core reads the key store for
   * each secret, so only the page of secrets asks.
   */
  listSecretStates(): Promise<ReadonlyMap<string, boolean>>
  /** A secret put before another one of its group; an empty `beforeId` puts it at the end. */
  moveSecret(id: string, beforeId: string): Promise<void>
  renameSecretGroup(group: string, name: string): Promise<void>
  /** One declaration of a plugin let to read a secret. The group is empty for a declaration with none. */
  allowPluginSecret(secretId: string, pluginId: string, name: string, group: string): Promise<void>
  denyPluginSecret(secretId: string, pluginId: string, name: string, group: string): Promise<void>
  /**
   * A plugin whose last start failed, started once more. The answer is why it failed
   * again, and empty where it started.
   */
  retryPlugin(pluginId: string): Promise<string>
  /**
   * Opens the system picker for a plugin setting of type file or folder, and
   * writes the selected path. The returned problem states why the setting does not
   * accept the path, and is empty when the path was written.
   */
  choosePluginPath(
    pluginId: string,
    key: string,
  ): Promise<{ readonly path: string; readonly isChosen: boolean; readonly problem: string }>
  /** How many files the folder of a plugin holds in the shown vault, `.numen/<plugin id>/`. */
  getPluginData(pluginId: string): Promise<number>
  /** The folder of a plugin in the shown vault deleted, with the text versions the plugin made there. */
  deletePluginData(pluginId: string): Promise<void>
}
