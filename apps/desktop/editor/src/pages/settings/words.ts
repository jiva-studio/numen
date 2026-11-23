import { PLUGIN_WORDS } from '@numen/plugin-manager'

/** UI copy of the settings tab: group headings, setting titles and descriptions. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  ...PLUGIN_WORDS,
  /** Shown in place of the settings file path until the path is known. */
  file: 'numen.json in the configuration folder',
  /** Accessible name of the search field in a long model list. */
  searchValues: (title: string) => `Search ${title.toLowerCase()}`,
  searchConnections: 'Search connections',
  /** The button of the fix of a health item that names no title of its own. */
  healthAction: 'Fix',
  resetSettingOf: (name: string) => `Reset ${name} to its default`,
  /** The info button of a setting with details, and the popover it opens. */
  aboutSetting: (name: string) => `About ${name}`,
  /** The slots a plugin draws: what is placed in each, and how a person orders and hides it. */
  slotsShow: (name: string) => `Show ${name}`,
  slotsGrip: (name: string) => `Reorder ${name}`,
  slotsAddSeparator: 'Add separator',
  slotsRemoveSeparator: (position: number, total: number) =>
    total > 1 ? `Remove separator ${position} of ${total}` : 'Remove separator',
  slotsSeparatorGrip: (position: number, total: number) =>
    total > 1 ? `Reorder separator ${position} of ${total}` : 'Reorder separator',
  slotsReset: (title: string) => `Reset ${title} to its default`,
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
  connectionDetected: 'Discovered',
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
  connectionRowsDiscover: 'Discover automatically',
  connectionRowsChoose: 'Choose a connection',
  connectionRowsConnection: 'Connection',
  connectionRowsChooseValue: (title: string) => `Choose a ${title.toLowerCase()}`,
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
  /** A plugin's setting of type file or folder. */
  pathChoose: 'Choose…',
  pathAdd: 'Add…',
  pathClear: 'Clear',
  /** The identity section of the page of identity and licenses. */
  identitySection: 'Identity',
  identityNoneDetail:
    'An identity is your Numen account. Unlike a usual account, it lives on your own computer, not on our servers: there is no email and no password to remember. Your devices use it to recognize each other, and your subscription belongs to it. Numen never sees it, so nobody else can read your notes.',
  identityNoneWords:
    'When you create it, you get 12 recovery words — write them down. With them you can bring the same account to a new computer or get it back if this one is lost.',
  identityCreate: 'Create identity',
  identityRestore: 'Restore',
  identitySetupTitle: 'Recovery words',
  identityWriteDown: 'Write the 12 words down in order. We show them once.',
  identityNobodyRecovers:
    'Neither Numen nor support can recover these words. If you lose the words and all your devices, you lose access to your cloud vaults.',
  identityWordsWritten: 'I wrote the words down',
  identityRecoveryWords: 'Recovery words',
  identityCheckTitle: 'Check the words',
  identityConfirmHint: (positions: readonly number[]) =>
    `Enter words ${positions
      .slice(0, -1)
      .map((position) => `#${position}`)
      .join(', ')} and #${positions.at(-1)} from your record.`,
  identityConfirmWord: (position: number) => `Word ${position}`,
  identityConfirmPlaceholder: 'word',
  identityBack: 'Back',
  identityContinue: 'Continue',
  identityKeeping: 'Keeping…',
  identityCancel: 'Cancel',
  identityRestoreLabel: 'Your 12 recovery words',
  identityRestorePlaceholder: 'Type the 12 words, separated by spaces',
  identityRestoring: 'Restoring…',
  identityNotValid: 'Some words are not valid.',
  identityAccount: 'Account fingerprint',
  identityAccountDetail: 'Compare it when you link a device',
  identityThisDevice: 'This device',
  identityDevices: 'Devices',
  identityDeviceCount: (count: number) => (count === 1 ? '1 device' : `${count} devices`),
  identityDeviceLinked: (date: string) => (date === '' ? 'Linked' : `Linked ${date}`),
  identityDelete: 'Delete identity',
  identityDeleteTitle: 'Delete the identity from this device?',
  identityDeleteWarning: 'Without the 12 words this identity cannot be restored.',
  identityDeleteConfirm: 'Delete',
  identityDeleteCancel: 'Cancel',
  /** The page of licenses. */
  licensesEmpty: 'No licenses declared.',
  licensesEmptyDetail:
    'Add a plugin with a subscription or license key to manage its license here.',
  licenseKeyPlaceholder: 'Enter license key…',
  licenseKeyLabel: (title: string) => `License key for ${title}`,
  licenseActivate: 'Activate',
  licenseActivating: 'Activating…',
  licenseDeactivate: 'Deactivate',
  licenseDeactivating: 'Deactivating…',
  licenseBuy: 'Buy',
  licenseStatusActive: 'Active',
  licenseStatusExpired: 'Expired',
  licenseStatusUnverified: 'Unverified',
  licenseStatusInactive: 'Inactive',
  /** A plugin's setting of type list. */
  listAdd: 'Add',
  listNew: 'Add an item',
  listRemove: (item: string) => `Remove ${item}`,
  listItemTwice: (item: string) => `${item} is already in the list`,
}
