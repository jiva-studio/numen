import { PLUGIN_WORDS } from '@numen/plugin-manager'

/** UI copy of the settings tab: group headings, setting titles and descriptions. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  ...PLUGIN_WORDS,
  /** Shown in place of the settings file path until the path is known. */
  file: 'numen.json in the configuration folder',
  /** Accessible name of the search field in a long model list. */
  searchModels: 'Search models',
  searchValues: (title: string) => `Search ${title.toLowerCase()}`,
  searchConnections: 'Search connections',
  /** The button of the fix of a health item that names no title of its own. */
  healthAction: 'Fix',
  /** Accessible name of the info button next to a setting title. */
  aboutSetting: (name: string) => `About ${name}`,
  /** The page of connections. */
  secretMasked: '••••••••',
  connectionAdd: 'Add connection',
  connectionEdit: 'Edit connection',
  connectionMenuDelete: 'Delete',
  connectionsEmpty: 'No connections yet.',
  connectionsEmptyDetail:
    'Add a connection to let a plugin use a service or a program. You enter the address and the key once.',
  connectionsNoTypes: 'No installed plugin provides a kind of connection.',
  connectionUnavailable: 'Type unavailable',
  connectionUnavailableDetail:
    '"Type unavailable" means no installed plugin provides this type. Plugins skip such a connection.',
  connectionIncomplete: 'Incomplete',
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
  connectionAddress: 'Address',
  connectionAddressFixed: 'The address is part of this type.',
  connectionAddressChanged: 'Changing the address removes the stored key. Enter it again.',
  connectionAgentNote: (pluginName: string) =>
    `${pluginName} runs the program on this computer with your own login. No other field is needed.`,
  connectionNew: 'New connection…',
  searchConnectionTypes: 'Search types',
  connectionSave: 'Save',
  connectionCancel: 'Cancel',
  connectionFieldSelect: 'Choose',
  connectionRequired: 'Required.',
  /** The rows of connections on the page of a plugin. */
  connectionRowsNone: 'No connections yet. Add a connection first.',
  connectionRowsOpen: 'Open Connections',
  connectionRowsChoose: 'Choose a connection',
  connectionRowsConnection: 'Connection',
  connectionRowsValuePlaceholder: (title: string) => `Choose or type a ${title.toLowerCase()}`,
  connectionRowsDefault: 'Default',
  connectionRowsIndexQuestion: (count: number | undefined, name: string) =>
    count === undefined
      ? `The vault will be indexed again through ${name}. This can take a while and may cost money.`
      : `The vault will be indexed again: ${count} ${count === 1 ? 'note' : 'notes'} through ${name}. This can take a while and may cost money.`,
  connectionRowsIndexLabel: 'Index the notes again?',
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
  connectionRowsOptionsFailed: (title: string, detail: string) =>
    `The values of ${title} could not be listed: ${detail}`,
  /** title: what a plugin titles the operation, or its name. */
  connectionCannot: (title: string) => `This type cannot ${title}`,
  /** The label of the action that opens a plugin's settings. */
  configure: 'Configure',
  /** A plugin's setting of type file or folder. */
  pathNone: 'None',
  pathChoose: 'Choose…',
  pathAdd: 'Add…',
  pathClear: 'Clear',
}
