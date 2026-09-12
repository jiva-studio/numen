/**
 * The document of the spine being drawn, and the markup it is drawn from.
 */
import { computed, ref, shallowRef, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '@/shared/span'
import type { MessageWriter } from '@/shared/notices/messages'
import { resolveImageUrls } from '../lib/markup'
import type { Books, SpineDocument } from '../types'

export function useBookDocument(
  books: Books,
  path: string,
  fingerprint: Ref<string>,
  writeMessage: MessageWriter,
  isOpen: () => boolean,
) {
  const markup = ref('')
  const drawnDocument = shallowRef<SpineDocument | undefined>(undefined)

  const reading = computed<Span>(() => drawnDocument.value?.span ?? { from: 0, to: 0 })
  const drawn = computed(() => drawnDocument.value?.path ?? '')

  /** Which ask for markup is the current one. An older answer is dropped. */
  let wanted = 0

  const draw = async (document: SpineDocument | undefined) => {
    if (!document || document.path === drawnDocument.value?.path) return
    const asked = ++wanted
    try {
      const markupContent = await books.readMarkup(path, document.path, fingerprint.value)
      if (!isOpen() || asked !== wanted) return
      drawnDocument.value = document
      markup.value = resolveImageUrls(markupContent, (name) =>
        books.getEntryUrl(path, name, fingerprint.value),
      )
    } catch (error) {
      if (!isOpen() || asked !== wanted) return
      writeMessage(formatErrorMessage(error), 'error')
    }
  }

  const close = () => {
    markup.value = ''
  }

  return { markup, drawn, reading, draw, close }
}
