/**
 * Reactive CodeMirror editor instance management.
 */
import { doc } from '@numen/ui'
import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import type { LanguageSupport } from '@codemirror/language'
import type { Extension } from '@codemirror/state'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { EditorChange } from '../lib/change/change'
import {
  extensionsCompartment,
  code,
  previewCompartment,
  createEditable,
  editableCompartment,
  getLineNumbersExtension,
  getTabSizeExtension,
  getWrapExtension,
  lineNumbersCompartment,
  preview,
  prose,
  setup,
  showChange,
  changeCompartment,
  languageCompartment,
  tabSizeCompartment,
  wrapCompartment,
} from '../lib/setup'
import { loadLanguage } from '../config/languages'
import { addressOpener, addressResolver, saveHandler } from '../lib/outside'
import { replace } from '../lib/change/replace'

export interface EditorProps {
  /** Marks are drawn as what they mean. Off, the text is shown as written. */
  isLive?: boolean | undefined
  /**
   * What the whole document is written in, by the name a fence would use. A
   * document naming none is markdown, and one naming a language is set in the
   * face code is set in.
   */
  language?: string | undefined
  isReadonly?: boolean | undefined
  placeholder?: string | undefined
  /** What it is announced as. What has been typed here is no name for it. */
  name?: string | undefined
  /**
   * How to leave the editor, announced on arrival. The editor captures Tab while
   * the user types, so the way out must be announced.
   */
  keys?: string | undefined
  /** A change being made to this text by something other than the reader. */
  change?: EditorChange | null | undefined
  /** What an address in the text becomes before the window loads it. */
  resolve?: ((address: string) => string) | undefined
  /** More the caller draws into this editor. */
  extensions?: Extension | undefined
  softWrap?: boolean | undefined
  lineNumbers?: boolean | undefined
  tabSize?: number | undefined
}

export interface EditorEmit {
  (event: 'open', address: string): void
  (event: 'save'): void
}

export interface EditorStateOptions {
  readonly host: Ref<HTMLElement | null>
  readonly props: EditorProps
  readonly text: Ref<string>
  readonly keysId: string
  readonly emit: EditorEmit
  readonly softWrap?: () => boolean | undefined
  readonly lineNumbers?: () => boolean | undefined
  readonly tabSize?: () => number | undefined
}

export function useEditor(options: EditorStateOptions) {
  let view: EditorView | null = null
  let lastEmitted: string | null = null

  const readSoftWrap = () => (options.softWrap ? options.softWrap() : options.props.softWrap)
  const readLineNumbers = () =>
    options.lineNumbers ? options.lineNumbers() : options.props.lineNumbers
  const readTabSize = () => (options.tabSize ? options.tabSize() : options.props.tabSize)

  function setLanguage(name: string, support: LanguageSupport | null): void {
    if (!view || name !== options.props.language) return
    view.dispatch({ effects: languageCompartment.reconfigure(support ? code(support) : prose()) })
  }

  /** Loads the language pack of the name; a pack that fails to load leaves the document plain. */
  function applyLanguage(name: string): void {
    const loading = name ? loadLanguage(name) : Promise.resolve(null)
    loading.then(
      (support) => setLanguage(name, support),
      () => setLanguage(name, null),
    )
  }

  onMounted(() => {
    if (!options.host.value) return
    const rootNode = options.host.value.getRootNode()
    const root =
      rootNode instanceof Document ||
      (typeof ShadowRoot !== 'undefined' && rootNode instanceof ShadowRoot)
        ? rootNode
        : doc
    view = new EditorView({
      parent: options.host.value,
      root,
      state: EditorState.create({
        doc: options.text.value,
        extensions: [
          setup({
            isLivePreview: options.props.isLive,
            readonly: options.props.isReadonly,
            placeholder: options.props.placeholder,
            name: options.props.name,
            describedBy: options.keysId,
            change: options.props.change,
            extensions: options.props.extensions,
            softWrap: readSoftWrap(),
            lineNumbers: readLineNumbers(),
            tabSize: readTabSize(),
          }),
          addressResolver.of((address) => options.props.resolve?.(address) ?? address),
          addressOpener.of((address) => options.emit('open', address)),
          saveHandler.of(() => options.emit('save')),
          EditorView.updateListener.of((update) => {
            if (!update.docChanged) return
            lastEmitted = update.state.doc.toString()
            options.text.value = lastEmitted
          }),
        ],
      }),
    })
    if (options.props.language) applyLanguage(options.props.language)
  })

  onBeforeUnmount(() => {
    view?.destroy()
    view = null
  })

  watch(options.text, (newText) => {
    if (!view || newText === lastEmitted || view.state.doc.toString() === newText) return
    view.dispatch(replace(view.state, newText))
  })

  watch(
    () => options.props.isLive,
    (on) => view?.dispatch({ effects: previewCompartment.reconfigure(preview(on ?? true)) }),
  )

  watch(
    () => options.props.language,
    (name) => {
      applyLanguage(name ?? '')
    },
  )

  watch(
    () => options.props.isReadonly,
    (off) => view?.dispatch({ effects: editableCompartment.reconfigure(createEditable(!off)) }),
  )

  watch(
    () => options.props.change,
    (change) =>
      view?.dispatch({ effects: changeCompartment.reconfigure(showChange(change ?? null)) }),
  )

  watch(
    () => options.props.extensions,
    (more) => view?.dispatch({ effects: extensionsCompartment.reconfigure(more ?? []) }),
  )

  watch(readSoftWrap, (softWrap) =>
    view?.dispatch({ effects: wrapCompartment.reconfigure(getWrapExtension(softWrap ?? true)) }),
  )

  watch(readLineNumbers, (show) =>
    view?.dispatch({
      effects: lineNumbersCompartment.reconfigure(getLineNumbersExtension(show ?? false)),
    }),
  )

  watch(readTabSize, (tabSize) =>
    view?.dispatch({
      effects: tabSizeCompartment.reconfigure(getTabSizeExtension(tabSize ?? 2)),
    }),
  )

  const focus = (): boolean => {
    if (!view) return false
    view.focus()
    return true
  }

  const measure = (): void => {
    view?.requestMeasure()
  }

  const reveal = (line: number): boolean => {
    if (!view || view.state.doc.length === 0) return false
    const lineNumber = Math.min(Math.max(Math.trunc(line), 0) + 1, view.state.doc.lines)
    const { from } = view.state.doc.line(lineNumber)
    view.dispatch({
      selection: { anchor: from },
      effects: EditorView.scrollIntoView(from, { y: 'start' }),
    })
    view.focus()
    return true
  }

  return {
    focus,
    measure,
    reveal,
  }
}
