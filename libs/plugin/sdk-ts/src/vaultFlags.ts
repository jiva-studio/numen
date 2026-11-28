// VaultFlagsHost is where a plugin sets flags on the vaults of the window, which the `when` conditions of
// every plugin read as `vault.flags.<plugin id>.<key>` for the vault the window shows.
export interface VaultFlagsHost {
  // Sets the flag of the key on the vault. A flag is true or false; one a plugin never sets reads as missing.
  setFlag(vaultId: string, key: string, value: boolean): void
}
