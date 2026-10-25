/**
 * The document of the spine being drawn, and the markup it is drawn from.
 */
import { computed, ref, shallowRef, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '@numen/plugin-sdk'
import { resolveImageUrls } from '../lib/markup'
import type { Books, SpineDocument } from '../types'

export type MessageWriter = (text: string, tone?: 'info' | 'error') => void

export interface BookDocumentOptions {
  fingerprint: Ref<string>
  writeMessage: MessageWriter
  isOpen: () => boolean
}

export function useBookDocument(books: Books, path: string, options: BookDocumentOptions) {
  const { fingerprint, writeMessage, isOpen } = options
  const markup = ref('')
  const drawnDocument = shallowRef<SpineDocument | undefined>(undefined)

  const documentSpan = computed<Span>(() => drawnDocument.value?.span ?? { from: 0, to: 0 })
  const picture = computed(() => drawnDocument.value?.path ?? '')

  /** Which ask for markup is the current one. An older answer is dropped. */
  let wanted = 0

  const draw = async (document: SpineDocument | undefined) => {
    if (!document || document.path === drawnDocument.value?.path) return
    // Stryker disable next-line UpdateOperator: only the inequality of the tickets matters; the direction of the count does not
    const request = ++wanted
    try {
      const markupContent = await books.readMarkup(path, document.path, fingerprint.value)
      if (!isOpen() || request !== wanted) return
      drawnDocument.value = document
      markup.value = resolveImageUrls(markupContent, (name) =>
        books.getEntryUrl(path, name, fingerprint.value),
      )
    } catch (error) {
      if (!isOpen() || request !== wanted) return
      writeMessage(formatErrorMessage(error), 'error')
    }
  }

  const close = () => {
    markup.value = ''
  }

  return { markup, renderedChapter: picture, documentSpan, draw, close }
}
