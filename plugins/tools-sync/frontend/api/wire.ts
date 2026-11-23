/**
 * The clients of the services of the plugin's backend bound to the plugin's own API.
 */
import { createApi } from '@numen/plugin-sdk'
import { createFolderSharesClient, createReceivedSharesClient } from './shares'
import { createSubscriptionClient } from './subscription'
import { createVaultsClient, SERVICE } from './vaults'

export const vaultsClient = createVaultsClient(createApi(SERVICE))
export const subscriptionClient = createSubscriptionClient(createApi(SERVICE))
export const folderSharesClient = createFolderSharesClient(createApi(SERVICE))
export const readerSharesClient = createReceivedSharesClient(createApi(SERVICE))
