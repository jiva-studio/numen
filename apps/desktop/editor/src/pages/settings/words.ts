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
  /** The page of connections. */
  secretMasked: '••••••••',
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
  connectionRowsPrivacy: (limit: number) =>
    limit === 1
      ? 'The text of your notes is sent to the connection chosen here.'
      : 'The text of your notes is sent to each connection listed, in this order.',
  connectionRowsNone: 'No connections yet. Add one on the Connections page.',
  connectionRowsOpen: 'Open Connections',
  connectionRowsChoose: 'Choose a connection',
  connectionRowsConnection: 'Connection',
  connectionRowsModel: 'Model',
  connectionRowsModelPlaceholder: 'Choose or type a model',
  connectionRowsReasoning: 'Reasoning',
  connectionRowsDefault: 'Default',
  connectionRowsIndexQuestion: (count: number | undefined, name: string) =>
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
