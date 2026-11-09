import { PLUGIN_WORDS } from '@numen/plugin-manager'

function countOf(count: number, noun: string): string {
  return count === 1 ? `1 ${noun}` : `${count} ${noun}s`
}

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
  secretsFile: (path: string) => `Values are kept in ${path}`,
  secretMasked: '••••••••',
  secretNoReaders: 'No plugins',
  secretValue: (name: string) => `Value of ${name}`,
  secretMoveUp: 'Move up',
  secretMoveDown: 'Move down',
  secretActions: 'Secret actions',
  secretDeleteItem: 'Delete',
  secretRenameName: 'Name of the secret',
  secretDelete: 'Delete secret',
  secretDiscard: 'Discard',
  secretSaveFirst: 'Save a value first',
  secretAdd: 'Add secret',
  columnName: 'Name',
  columnValue: 'Value',
  columnUsedBy: 'Used by',
  secretAddGroup: 'Add group',
  secretSearchPlugins: 'Search plugins',
  secretMore: (count: number) => `+${count}`,
  secretValueSet: 'Value is set',
  secretValueEmpty: 'No value yet',
  secretNotSaved: 'Not saved. Press Enter to try again.',
  secretGroupTaken: (name: string) => `A group named ${name} already exists.`,
  secretGroupActions: 'Group actions',
  secretGroupRename: 'Rename group',
  secretGroupRenameName: 'Name of the group',
  secretGroupDelete: 'Delete group',
  secretGroupTooLong: (limit: number) => `Use at most ${limit} characters.`,
  secretGroupDeleteAsking: (group: string, count: number, readers: readonly string[]) =>
    `Delete the group ${group} with its ${countOf(count, 'secret')}? Their values are removed from this machine${
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
  secretOpenPage: 'Open the page of secrets',
  secretFieldNone: 'No secret allowed',
  /** The page of connections. */
  connectionsIntro:
    'A connection is a service you pay for, a server you run, or an agent on this computer. Keys are kept in the system keychain.',
  connectionAdd: 'Add connection',
  connectionEdit: 'Edit connection',
  connectionActions: 'Connection actions',
  connectionMenuEdit: 'Edit',
  connectionMenuDelete: 'Delete',
  connectionsEmpty: 'No connections yet.',
  connectionsEmptyDetail:
    'Add a connection to let a plugin use a service or a program. You enter the address and the key once.',
  connectionsNoTypes: 'No installed plugin provides a kind of connection.',
  connectionUnavailable: 'Type unavailable',
  connectionUnavailableDetail:
    '"Type unavailable" means no installed plugin provides this type. Plugins skip such a connection.',
  connectionIncomplete: 'Incomplete',
  connectionKeyMissing: 'No key',
  connectionDelete: 'Delete connection',
  connectionDeleteAsking: (name: string, hasKey: boolean) =>
    hasKey ? `Delete "${name}"? Its key is removed from this machine.` : `Delete "${name}"?`,
  connectionUsedBy: (names: string) => `Used by ${names}.`,
  connectionUsedByDeleted: (names: string) => `Used by ${names}. Their rows for it are removed.`,
  connectionDeleteConfirm: 'Delete',
  connectionDeleteCancel: 'Cancel',
  connectionDeleteFailed: 'The connection was not deleted:',
  connectionType: 'Type',
  connectionName: 'Name',
  connectionNameHint: 'Shown in lists. Leave empty to use the name of the type.',
  connectionAddress: 'Address',
  connectionAddressFixed: 'The address is part of this type.',
  connectionAddressHint: 'The key is sent only to this address.',
  connectionKeyStored: 'A key is stored. Type to replace it.',
  connectionKeyHint: 'Stored in the system keychain. Not shown again after saving.',
  connectionKeyClear: 'Remove key',
  connectionKeyClearing: 'The key will be removed when you save.',
  connectionAddressChanged: 'Changing the address removes the stored key. Enter it again.',
  connectionPrivateNetwork: 'Allow a private network address',
  connectionPrivateNetworkHint:
    'Needed for a server on your local network. Addresses on this computer are always allowed.',
  connectionModelNote: 'Choose the model on the page of the plugin, not here.',
  connectionAgentNote: (pluginName: string) =>
    `${pluginName} runs the program on this computer with your own login. No other field is needed.`,
  connectionGuide: 'How to get a key and what leaves this computer',
  connectionGuideAddress: 'https://docs.numen.md/connections/',
  connectionSave: 'Save',
  connectionCancel: 'Cancel',
  connectionFieldSelect: 'Choose',
  connectionRequired: 'Required.',
  connectionOptionalKey: 'May stay empty for a server on this computer.',
  /** The rows of connections on the page of a plugin. */
  connectionRowsPrivacy: 'The text of your notes is sent to each connection listed, in this order.',
  connectionRowsNone: 'No connections yet. Add one on the Connections page.',
  connectionRowsChoose: 'Choose a connection',
  connectionRowsConnection: 'Connection',
  connectionRowsModel: 'Model',
  connectionRowsModelPlaceholder: 'Choose or type a model',
  connectionRowsReasoning: 'Reasoning',
  connectionRowsDefault: 'Default',
  connectionRowsIndexAsking: (count: number | undefined, name: string) =>
    count === undefined
      ? `This indexes your notes through ${name}. It may cost money.`
      : `This indexes ${count} ${count === 1 ? 'note' : 'notes'} through ${name}. It may cost money.`,
  connectionRowsIndexLabel: 'Index the notes',
  connectionRowsIndexConfirm: 'Index',
  connectionRowsRemove: (position: number) => `Remove row ${position}`,
  connectionRowsGrip: (position: number, count: number) => `Move row ${position} of ${count}`,
  connectionRowsPickedUp: (position: number, count: number) =>
    `Picked up row ${position} of ${count}. Use the arrow keys to move it and Space to drop it.`,
  connectionRowsMoved: (position: number, count: number) =>
    `Moved to position ${position} of ${count}.`,
  connectionRowsDropped: (position: number, count: number) =>
    `Dropped at position ${position} of ${count}.`,
  connectionRowsCancelled: 'Move cancelled.',
  connectionRowsModelsFailed: 'The models could not be listed:',
  connectionCannot: (operation: string) => `This type cannot ${operation}`,
  /** The label of the action that opens a plugin's settings. */
  configure: 'Configure',
  /** A plugin's setting of type file or folder. */
  pathNone: 'None',
  pathChoose: 'Choose…',
  pathAdd: 'Add…',
  pathClear: 'Clear',
}
