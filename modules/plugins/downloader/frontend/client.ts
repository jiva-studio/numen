/**
 * Connect-RPC client for numen.v1.DownloadService.
 */
import { createClient } from '@connectrpc/connect'
import { DownloadService } from '@numen/protocol'
import { transport } from '@numen/wire'

export const downloaderClient = createClient(DownloadService, transport)
