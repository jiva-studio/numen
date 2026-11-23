export const WORDS = {
  syncingLabel: 'Syncing…',
  syncingDetail: 'Synchronizing vault…',
  syncedLabel: 'Synced',
  syncedDetail: 'Vault is synchronized.',
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
} as const
