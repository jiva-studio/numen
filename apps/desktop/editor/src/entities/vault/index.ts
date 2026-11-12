/** The vaults an installation holds: what one is, what changes the list, and what it reports. */
export type { Vault, VaultChange, VaultList, Vaults } from './lib/vault'
export { VAULT_ERRORS } from './words'
export { vaults } from './api/vaults'
export { getAgentUnreachable, getVaultState, watchVaultChanges } from './api/state'
export { useVaults } from './model/useVaults'
