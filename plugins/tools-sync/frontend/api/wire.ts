/**
 * The client of the vaults service bound to the plugin's own API.
 */
import { createApi } from '@numen/plugin-sdk'
import { createVaultsClient, SERVICE } from './vaults'

export const vaultsClient = createVaultsClient(createApi(SERVICE))
