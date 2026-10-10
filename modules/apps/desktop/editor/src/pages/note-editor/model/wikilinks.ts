/**
 * Wikilink completion adapter for note tabs.
 *
 * Connects CodeMirror wikilink autocompletion with vault note search,
 * section headings lookup, and on-the-fly note creation.
 */
import {
  createWikilinkCompletion,
  type Extension,
  type WikilinkCompletionOptions,
  type WikilinkOption,
} from '@numen/ui'
import type { NoteTabDeps } from '../types'

export function createWikilinkOptions(
  deps: NoteTabDeps,
  getCurrentPath: () => string,
): WikilinkCompletionOptions {
  return {
    search: async (query: string): Promise<readonly WikilinkOption[]> => {
      const matches = await deps.names(query, 20)
      return matches.map((one) => {
        if (one.heading) {
          return {
            title: one.title,
            heading: one.heading,
            detail: one.title,
            type: 'heading',
          }
        }
        return {
          title: one.title,
          type: 'note',
        }
      })
    },

    headings: async (noteTitle: string): Promise<readonly string[]> => {
      let path = getCurrentPath()
      if (noteTitle) {
        const matches = await deps.names(noteTitle, 5)
        const matched =
          matches.find(
            (one) => one.title.toLowerCase() === noteTitle.toLowerCase() && !one.heading,
          ) ?? matches[0]
        if (matched) path = matched.path
      }

      if (!path) return []
      const found = await deps.headings([path])
      const headingsList = found.get(path) ?? []
      return headingsList.map((one) => one.text)
    },

    createNote: async (title: string): Promise<string | null> => {
      try {
        const result = await deps.create({ title, folder: '', links: [] })
        return result.ok ? result.value.path : null
      } catch {
        return null
      }
    },
  }
}

export function createNoteWikilinkExtension(
  deps: NoteTabDeps,
  getCurrentPath: () => string,
): Extension {
  return createWikilinkCompletion(createWikilinkOptions(deps, getCurrentPath))
}
