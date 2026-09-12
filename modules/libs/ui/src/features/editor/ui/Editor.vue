<script setup lang="ts">
/**
 * Markdown, written and read in the same place.
 *
 * The text in the editor is the text of the file, mark for mark; nothing here
 * rewrites what was typed. What changes is how a construct is drawn: away
 * from the caret it is drawn as it reads, and where the caret stands the
 * marks come back.
 */
import { onBeforeUnmount, onMounted, useId, useTemplateRef, watch } from 'vue'
import { EditorState, type Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { EditorChange } from '../lib/change'
import {
  adding,
  code,
  drawing,
  editable,
  editing,
  preview,
  prose,
  setup,
  showing,
  shown,
  written,
} from '../lib/setup'
import { wholly } from '../config/languages'
import { opening, resolving, saving } from '../lib/outside'
import { replace } from '../lib/replace'

const props = withDefaults(
  defineProps<{
    /** Marks are drawn as what they mean. Off, the text is shown as written. */
    live?: boolean
    /**
     * What the whole document is written in, by the name a fence would use. A
     * document naming none is markdown, and one naming a language is set in the
     * face code is set in.
     */
    language?: string
    readonly?: boolean
    placeholder?: string
    /** What it is announced as. What has been typed here is no name for it. */
    name?: string
    /**
     * How to leave, read out on arrival. Tab is the editor's while a person is
     * writing, so the way back out has to be said or nobody finds it.
     */
    keys?: string
    /** A change being made to this text by something other than the reader. */
    change?: EditorChange | null
    /** What an address in the text becomes before the window loads it. */
    resolve?: (address: string) => string
    /** More the caller draws into this editor. */
    extensions?: Extension
  }>(),
  {
    live: true,
    language: '',
    readonly: false,
    placeholder: 'Write',
    name: 'Editor',
    keys: 'Tab indents. Press Escape, then Tab, to leave the editor.',
    change: null,
    extensions: () => [],
  },
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

/** What the words about the keyboard are addressed by, this editor's alone. */
const keysId = `${useId()}-keys`

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
          name: props.name,
          describedBy: keysId,
          change: props.change,
          extensions: props.extensions,
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
  if (props.language) void writes(props.language)
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
  view.dispatch(replace(view.state, fresh))
})

watch(
  () => props.live,
  (on) => view?.dispatch({ effects: drawing.reconfigure(preview(on)) }),
)

/**
 * The language is loaded when it is first wanted, so the editor is drawn before
 * it arrives and is reconfigured once it is here. A document whose language
 * changed while one was loading keeps the one it asked for last.
 */
const writes = async (name: string) => {
  const support = name ? await wholly(name) : null
  if (!view || name !== props.language) return
  view.dispatch({ effects: written.reconfigure(support ? code(support) : prose()) })
}

watch(() => props.language, writes)

watch(
  () => props.readonly,
  (off) => view?.dispatch({ effects: editing.reconfigure(editable(!off)) }),
)

watch(
  () => props.change,
  (change) => view?.dispatch({ effects: showing.reconfigure(shown(change)) }),
)

watch(
  () => props.extensions,
  (more) => view?.dispatch({ effects: adding.reconfigure(more) }),
)

defineExpose({
  /** Take the keyboard. False while there is no editor yet to take it. */
  focus: () => {
    if (!view) return false
    view.focus()
    return true
  },
  /**
   * Take the editor's measurements again. An editor drawn while it is hidden
   * has none to take. The caller says when it is on screen.
   */
  measure: () => view?.requestMeasure(),
  /**
   * Put the caret on one line of the prose and bring it into sight. Lines are
   * counted from the first line of the prose, and one past the end lands on the
   * last line there is.
   *
   * An editor holding no text holds no lines, and says so. Text can arrive
   * after the editor is drawn, and a caret asked for a line then stands on
   * that line.
   */
  reveal: (line: number) => {
    if (!view || view.state.doc.length === 0) return false
    const at = Math.min(Math.max(Math.trunc(line), 0) + 1, view.state.doc.lines)
    const { from } = view.state.doc.line(at)
    view.dispatch({
      selection: { anchor: from },
      effects: EditorView.scrollIntoView(from, { y: 'start' }),
    })
    view.focus()
    return true
  },
})
</script>

<template>
  <div ref="host" class="editor numen h-full min-h-0 overflow-auto font-sans text-base text-ink">
    <!-- Read out as the keyboard arrives, which is the one moment a person
         needs to know how to get away again. -->
    <span :id="keysId" class="sr-only">{{ keys }}</span>
  </div>
</template>

<style scoped>
/* The prose a person is writing takes a selection, and the caret and the
   clipboard work over it. */
.editor {
  user-select: text;
  -webkit-user-select: text;
}

/* A splitter above is being dragged, and the text under the pointer is left
   alone for as long as it is. */
[data-resizing] .editor :deep(.cm-content) {
  user-select: none;
  -webkit-user-select: none;
}
</style>
