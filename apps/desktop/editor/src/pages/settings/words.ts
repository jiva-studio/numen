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
  secretsAbout:
    'Keys and tokens for plugins. Enter a value once; it is never shown again. Allow each plugin declaration to read the secrets it needs.',
  secretsKeychain:
    'The values are kept in the system keychain or, where it is not available, in the file credentials/credentials.json, not in the settings file.',
  secretsFile: (path: string) =>
    `The values are not in the system keychain on this machine. They are kept in a protected file: ${path}`,
  secretStored: 'Enter a new value',
  secretNoValue: 'No value on this machine',
  secretDefault: 'Default',
  secretValue: (name: string) => `Value of ${name}`,
  secretMoveUp: 'Move up',
  secretMoveDown: 'Move down',
  secretRename: 'Rename secret',
  secretRenameName: 'Name of the secret',
  secretDelete: 'Delete secret',
  secretDiscard: 'Discard',
  secretAdd: 'Add secret',
  secretsEmpty: 'No secrets yet. Add one to share a key or token between plugins.',
  secretNewGroup: 'Group of the new secret',
  secretNewGroupPlaceholder: 'Group name',
  secretGroupTooLong: (limit: number) => `A group name has at most ${limit} characters.`,
  secretAllowedTo: 'Allowed to',
  secretGroupRename: 'Rename group',
  secretGroupRenameName: 'Name of the group',
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
