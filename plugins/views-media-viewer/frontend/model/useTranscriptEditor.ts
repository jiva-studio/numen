import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import type { Extension } from '@codemirror/state'
import { createTranscriptView, type TranscriptView } from './transcriptView'

export interface TranscriptEditorSource {
  readonly modelValue: string
  readonly isEditable: boolean
  readonly label: string
  readonly extensions: Extension
}

export interface TranscriptEditorEvents {
  readonly change: (text: string) => void
  readonly save: () => void
}

/** The editor view of the transcript's words, mounted in the host element and kept equal to the source. */
export function useTranscriptEditor(
  host: Readonly<Ref<HTMLElement | null>>,
  source: TranscriptEditorSource,
  events: TranscriptEditorEvents,
): void {
  let view: TranscriptView | null = null

  onMounted(() => {
    if (!host.value) return
    view = createTranscriptView({
      parent: host.value,
      text: source.modelValue,
      isEditable: source.isEditable,
      label: source.label,
      extensions: source.extensions,
      onChange: events.change,
      onSave: events.save,
    })
  })

  onBeforeUnmount(() => {
    view?.destroy()
    view = null
  })

  watch(
    () => source.modelValue,
    (text) => view?.setText(text),
  )
  watch(
    () => source.isEditable,
    (isEditable) => view?.setEditable(isEditable),
  )
  watch(
    () => source.extensions,
    (extensions) => view?.setExtensions(extensions),
  )
}
