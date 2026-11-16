/**
 * Wire adapter for the plugin's md.numen.readers.epub JSON API and its entries route.
 */
import { createApi } from '@numen/plugin-sdk'
import type { Books } from '../types'
import { createBooks } from './books'

/** The service name the manifest declares. */
const SERVICE = 'md.numen.readers.epub'

export const books: Books = createBooks(createApi(SERVICE))
