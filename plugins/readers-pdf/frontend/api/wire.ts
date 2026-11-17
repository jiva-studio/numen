/**
 * Wire adapter for the plugin's JSON API.
 */
import { createApi } from '@numen/plugin-sdk'
import type { Documents } from '../types'
import { createDocuments } from './documents'

/** The service name the manifest declares. */
const API_SERVICE = 'md.numen.readers.pdf'

export const documents: Documents = createDocuments(createApi(API_SERVICE))
