export const WORDS = {
  syncingLabel: 'Syncing…',
  syncingDetail: 'Synchronizing vault…',
  syncedLabel: 'Synced',
  syncedDetail: 'Vault is synchronized.',
  subscriptionTitle: 'Numen Sync',
  subscriptionLoading: 'Loading the subscription…',
  noSubscriptionText:
    'Sync your vaults between your devices. Pick the storage you need, change it any time.',
  renews: 'Renews',
  trialUntil: 'Trial until',
  planRow: 'Plan',
  storageRow: 'Storage',
  storageDetail: 'Space used by your synced vaults',
  syncedFiles: (size: string): string => `${size} of synced files`,
  perMonth: '/ month',
  pricePerMonth: (price: string): string => `${price} / month`,
  ended: 'Ended',
  graceText: (periodEnd: string, writesEnd: string): string =>
    `The subscription ended on ${periodEnd}. Changes are sent until ${writesEnd}, then the vaults in the cloud can only be read.`,
  readOnly: 'Read only',
  readOnlyText: (periodEnd: string, deleteAt: string): string =>
    `The subscription ended on ${periodEnd}. The vaults in the cloud can only be read and changes stay on this device. The cloud deletes them on ${deleteAt}.`,
  deleted: 'Deleted',
  deletedText: 'The cloud deleted your vaults. Buy a subscription to sync again.',
  needsIdentity: 'Needs an identity',
  noIdentityText:
    'Sync your vaults between your devices. Notes and files are encrypted on your devices before they leave them. Create an identity in the card above to start.',
  offline: 'Offline',
  chooseNamed: (name: string, detail: string): string =>
    detail === '' ? `Choose ${name}` : `Choose ${name}, ${detail}`,
  changePlan: 'Change plan',
  changePlanChoices: 'Plans to change to',
  usagePercentName: 'Space used',
  manageSubscription: 'Manage subscription',
  cloudFull:
    'Cloud storage is full: changes stay on this device. Move to a larger plan to keep syncing.',
  waitingForPayment: 'Finish the payment in your browser. This card updates by itself.',
  pageNotOpened: {
    notWebAddress: 'The page has an address the browser cannot open.',
    openFailed: 'The browser did not open the page.',
    permissionDenied: 'Numen Sync may not open the browser.',
  },
  stopWaiting: 'Cancel',
  unreachableText: (server: string): string =>
    server === ''
      ? 'Start the sync server, or correct its address in the settings.'
      : `Could not connect to ${server}. Start the server, or change the address in the settings.`,
  openSettings: 'Open the settings',
  paymentNotSeen: 'The payment is not seen yet. Open this page again after you pay.',
  vaultsTitle: 'Vaults',
  connectNew: 'Connect as a new synced vault',
  connectExisting: 'Connect to an existing synced vault',
  chooseRemote: 'Choose a synced vault',
  noRemoteVaults: 'No synced vaults in your account',
  closeRemote: 'Cancel',
  loading: 'Loading vaults…',
  stateNotConnected: 'Not connected',
  stateConnected: 'Connected',
  stateCopy: 'a copy of a connected vault',
  noIdentity: 'No identity on this device',
  syncedJustNow: 'Synced less than a minute ago',
  syncedMinutesAgo: (count: number): string =>
    `Synced ${count} ${count === 1 ? 'minute' : 'minutes'} ago`,
  conflicts: (count: number): string =>
    `${count} ${count === 1 ? 'note' : 'notes'} changed on two computers`,
  deletionsPausedSend: (count: number, total: number): string =>
    `Sync is paused: ${count} of ${total} files are removed`,
  deletionsPausedApply: (count: number, total: number): string =>
    `Sync is paused: ${count} of ${total} files were removed on another computer`,
  removeFiles: 'Remove the files',
  keepFiles: 'Keep the files',
} as const
