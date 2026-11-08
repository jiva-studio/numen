/**
 * Where a tab plays from: a file of the vault from its own address, a note from the address its
 * `media` property names, or, while it names none, the page its `page` property names.
 */
import type { AddressSource, VaultNotes } from '@numen/plugin-sdk'
import type { RecordingSource } from '../types'

/** The extension of a note. */
const NOTE_EXTENSION = '.md'

const NOTHING: RecordingSource = { mediaUrl: '', mediaType: '', pageUrl: '' }

const readText = (properties: Readonly<Record<string, unknown>>, key: string): string => {
  const value = properties[key]
  return typeof value === 'string' ? value : ''
}

export function createSources(
  getNotes: () => Pick<VaultNotes, 'readProperties'> | undefined,
  resolve: (address: string) => Promise<AddressSource>,
) {
  const readNote = async (path: string): Promise<RecordingSource> => {
    const notes = getNotes()
    if (notes === undefined) return NOTHING
    const properties = await notes.readProperties(path)
    const media = readText(properties, 'media')
    if (media !== '') {
      const source = await resolve(media)
      return { mediaUrl: source.url, mediaType: source.mediaType, pageUrl: '' }
    }
    const page = readText(properties, 'page')
    if (page === '') return NOTHING
    return { ...NOTHING, pageUrl: (await resolve(page)).url }
  }

  return {
    getSource: async (path: string): Promise<RecordingSource> => {
      if (path.toLowerCase().endsWith(NOTE_EXTENSION)) return readNote(path)
      const source = await resolve(path)
      return { mediaUrl: source.url, mediaType: source.mediaType, pageUrl: '' }
    },
  }
}
