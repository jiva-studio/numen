<script setup lang="ts">
/**
 * Markdown, written and read in the same place.
 *
 * The text in the editor is the text of the file, mark for mark; nothing here
 * rewrites what was typed. What changes is how a construct is drawn: away
 * from the caret it is drawn as it reads, and where the caret stands the
 * marks come back.
 */
import { onBeforeUnmount, onMounted, useTemplateRef, watch } from 'vue'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { EditorChange } from './change'
import { drawing, editable, editing, preview, setup, showing, shown } from './setup'
import { opening, resolving, saving } from './outside'
import { replacing } from './replacing'

const props = withDefaults(
  defineProps<{
    /** Marks are drawn as what they mean. Off, the text is shown as written. */
    live?: boolean
    readonly?: boolean
    placeholder?: string
    /** A change being made to this text by something other than the reader. */
    change?: EditorChange | null
    /** What an address in the text becomes before the window loads it. */
    resolve?: (address: string) => string
  }>(),
  { live: true, readonly: false, placeholder: 'Write', change: null },
)

const emit = defineEmits<{
  /** A drawn link was followed. */
  (event: 'open', address: string): void
  /** The person asked, with `Ctrl+S`, for the text to be kept now. */
  (event: 'save'): void
}>()

const text = defineModel<string>({ default: '' })

const host = useTemplateRef<HTMLElement>('host')
let view: EditorView | null = null

onMounted(() => {
  if (!host.value) return
  view = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: text.value,
      extensions: [
        setup({
          live: props.live,
          readonly: props.readonly,
          placeholder: props.placeholder,
          change: props.change,
        }),
        resolving.of((address) => props.resolve?.(address) ?? address),
        opening.of((address) => emit('open', address)),
        saving.of(() => emit('save')),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) text.value = update.state.doc.toString()
        }),
      ],
    }),
  })
})

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})

// Text put in from outside replaces what is there; text that came from here
// is already in. The caret, the selection and the scroll offset stay where
// they were, and the replacement is no step to undo.
watch(text, (fresh) => {
  if (!view || view.state.doc.toString() === fresh) return
  view.dispatch(replacing(view.state, fresh))
})

watch(
  () => props.live,
  (on) => view?.dispatch({ effects: drawing.reconfigure(preview(on)) }),
)

watch(
  () => props.readonly,
  (off) => view?.dispatch({ effects: editing.reconfigure(editable(!off)) }),
)

watch(
  () => props.change,
  (change) => view?.dispatch({ effects: showing.reconfigure(shown(change)) }),
)

defineExpose({
  focus: () => view?.focus(),
  /**
   * Take the editor's measurements again. An editor drawn while it is hidden
   * has none to take. The caller says when it is on screen.
   */
  measure: () => view?.requestMeasure(),
})
</script>

<template>
  <div ref="host" class="editor numen h-full min-h-0 overflow-auto font-sans text-base text-ink" />
</template>
