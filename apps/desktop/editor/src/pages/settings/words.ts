import { PLUGIN_WORDS } from '@numen/plugin-manager'

/** UI copy of the settings tab: group headings, setting titles and descriptions. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  ...PLUGIN_WORDS,
  /** Shown in place of the settings file path until the path is known. */
  file: 'numen.json in the configuration folder',
  /** Accessible name of the search field in a long model list. */
  searchModels: 'Search models',
  /** Accessible name of the info button next to a setting title. */
  aboutSetting: (name: string) => `About ${name}`,
  /** The secrets declared by plugins, on their page and on the field of a plugin. */
  keyPlaceholder: 'Paste a value',
  secretsTitle: 'Secrets',
  secretsAbout: 'Secrets for plugins are stored here.',
  secretsFile: (path: string) => `Values are kept in ${path}`,
  secretMasked: '••••••••',
  secretReaders: (count: number) =>
    count === 0 ? 'No plugins' : count === 1 ? '1 plugin' : `${count} plugins`,
  secretNewGroupName: 'New group',
  secretDefault: 'Default',
  secretValue: (name: string) => `Value of ${name}`,
  secretMoveUp: 'Move up',
  secretMoveDown: 'Move down',
  secretRename: 'Rename secret',
  secretRenameName: 'Name of the secret',
  secretDelete: 'Delete secret',
  secretDiscard: 'Discard',
  secretAdd: 'Add secret',
  secretAddGroup: 'Add group',
  secretAllowedTo: 'Allowed to',
  secretGroupRename: 'Rename group',
  secretGroupRenameName: 'Name of the group',
  secretGroupDelete: 'Delete group',
  secretGroupTooLong: (limit: number) => `Use at most ${limit} characters.`,
  secretGroupDeleteAsking: (group: string, count: number, readers: readonly string[]) =>
    `Delete the group ${group} with its ${count === 1 ? '1 secret' : `${count} secrets`}? Their values are removed from this machine${
      readers.length === 0 ? '.' : `, and ${readers.join(', ')} can no longer read them.`
    }`,
  secretDeleteAsking: (name: string, readers: readonly string[]) =>
    readers.length === 0
      ? `Delete ${name}? Its value is removed from this machine.`
      : `Delete ${name}? Its value is removed from this machine, and ${readers.join(', ')} can no longer read it.`,
  secretDeleteConfirm: 'Delete',
  secretDeleteCancel: 'Cancel',
  secretChoose: 'Choose a secret',
  secretSearch: 'Search secrets',
  secretNothing: 'No secret to choose',
  secretOpenPage: 'Open the page of secrets',
  secretFieldNone: 'No secret allowed',
  /** The label of the action that opens a plugin's settings. */
  configure: 'Configure',
  /** A plugin's setting of type file or folder. */
  pathNone: 'None',
  pathChoose: 'Choose…',
  pathAdd: 'Add…',
  pathClear: 'Clear',
}
