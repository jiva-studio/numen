/**
 * What a plugin says about whether it works: a health item under a key of its own, which the window
 * shows on the page of the plugin and, for an error, in the list of tasks, with the fix it offers.
 */

// HealthSeverity is how bad an item is. A warning shows on the page of the plugin; an error shows in
// the list of tasks too.
type HealthSeverity = 'warning' | 'error'

// HealthAction is the command a person runs to fix what an item reports: an `app:` command, or a
// command of the plugin itself by its full id. `values` are the values of the arguments the command
// declares; an empty title reads the title of the command.
interface HealthAction {
  readonly command: string
  readonly values?: Readonly<Record<string, string>>
  readonly title?: string
}

// HealthItem is one concern of the plugin that does not work. `key` is the plugin's name for the
// concern: lowercase letters, digits and hyphens, at most 32 characters. `text` is one sentence of
// at most 120 characters, `detail` at most 500. A plugin holds at most 8 keys.
export interface HealthItem {
  readonly key: string
  readonly severity: HealthSeverity
  readonly text: string
  readonly detail?: string
  readonly action?: HealthAction
}

// HealthHost is where a plugin reports its health items.
export interface HealthHost {
  // Puts the item in the list, or replaces the one under its key. Rejects an item the core refuses:
  // past the limits, a command the plugin may not run, or values the command does not accept.
  report(item: HealthItem): Promise<void>
  // Takes the item under the key out of the list.
  clear(key: string): Promise<void>
}

export const SETTINGS_OPEN_COMMAND = 'app:settings.open'

// Returns the action that opens the settings of the plugin with the setting of the key marked. An empty
// key opens the settings of the plugin. It is health.OpenSettings of the Go SDK.
export const openSettings = (setting = ''): NonNullable<HealthItem['action']> => ({
  command: SETTINGS_OPEN_COMMAND,
  values: setting === '' ? {} : { setting },
})
