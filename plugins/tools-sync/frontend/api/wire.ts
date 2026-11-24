/**
 * The clients of the services of the plugin's backend bound to the plugin's own API.
 */
import { createApi } from '@numen/plugin-sdk'
import { createAcceptInvitationClient, createShareInvitationClient } from './invitations'
import { createFolderSharesClient, createReceivedSharesClient } from './shares'
import { createAccountClient } from './account'
import { createSubscriptionClient } from './subscription'
import { createVaultsClient, SERVICE } from './vaults'

export const vaultsClient = createVaultsClient(createApi(SERVICE))
export const subscriptionClient = createSubscriptionClient(createApi(SERVICE))
export const folderSharesClient = createFolderSharesClient(createApi(SERVICE))
export const receivedSharesClient = createReceivedSharesClient(createApi(SERVICE))
export const shareInvitationClient = createShareInvitationClient(createApi(SERVICE))
export const acceptInvitationClient = createAcceptInvitationClient(createApi(SERVICE))
export const accountClient = createAccountClient(createApi(SERVICE))
