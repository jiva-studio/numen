/**
 * Reactive CodeMirror editor instance management.
 */
import { doc } from '@numen/ui'
import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import type { LanguageSupport } from '@codemirror/language'
import type { Extension } from '@codemirror/state'
import { EditorState, type StateEffect } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { EditorChange } from '../lib/change/change'
import {
  extensionsCompartment,
  code,
  previewCompartment,
  createEditableExtension,
  editableCompartment,
  getLineNumbersExtension,
  getTabSizeExtension,
  getFontExtension,
  getLineHeightExtension,
  getActiveLineExtension,
  fontCompartment,
  lineHeightCompartment,
  activeLineCompartment,
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
} from './setup'
import { loadLanguage } from '../lib/config/languages'
import { addressOpener, addressResolver, saveHandler } from '../lib/outside'
import { replace } from '../lib/change/replace'

export interface EditorProps {
  isLive?: boolean | undefined
  language?: string | undefined
  isReadonly?: boolean | undefined
  placeholder?: string | undefined
  name?: string | undefined
  keys?: string | undefined
  change?: EditorChange | null | undefined
  resolve?: ((address: string) => string) | undefined
  extensions?: Extension | undefined
  softWrap?: boolean | undefined
  lineNumbers?: boolean | undefined
  tabSize?: number | undefined
  fontFamily?: string | undefined
  lineHeight?: number | undefined
  highlightActiveLine?: boolean | undefined
}

interface EditorEmit {
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
  readonly fontFamily?: () => string | undefined
  readonly lineHeight?: () => number | undefined
  readonly highlightActiveLine?: () => boolean | undefined
}

export function useEditor(options: EditorStateOptions) {
  let view: EditorView | null = null
  let lastEmitted: string | null = null

  const readSoftWrap = () => (options.softWrap ? options.softWrap() : options.props.softWrap)
  const readLineNumbers = () =>
    options.lineNumbers ? options.lineNumbers() : options.props.lineNumbers
  const readTabSize = () => (options.tabSize ? options.tabSize() : options.props.tabSize)
  const readFontFamily = () =>
    options.fontFamily ? options.fontFamily() : options.props.fontFamily
  const readLineHeight = () =>
    options.lineHeight ? options.lineHeight() : options.props.lineHeight
  const readHighlightActiveLine = () =>
    options.highlightActiveLine ? options.highlightActiveLine() : options.props.highlightActiveLine

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
            fontFamily: readFontFamily(),
            lineHeight: readLineHeight(),
            highlightActiveLine: readHighlightActiveLine(),
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
    (off) =>
      view?.dispatch({ effects: editableCompartment.reconfigure(createEditableExtension(!off)) }),
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

  /** Reconfigures a setting that changes line heights; the view measures again, so the gutter follows the lines. */
  function reconfigureLayout(effect: StateEffect<unknown>): void {
    view?.dispatch({ effects: effect })
    view?.requestMeasure()
  }

  watch(readSoftWrap, (softWrap) =>
    reconfigureLayout(wrapCompartment.reconfigure(getWrapExtension(softWrap ?? true))),
  )

  watch(readLineNumbers, (show) =>
    view?.dispatch({
      effects: lineNumbersCompartment.reconfigure(getLineNumbersExtension(show ?? false)),
    }),
  )

  watch(readTabSize, (tabSize) =>
    reconfigureLayout(tabSizeCompartment.reconfigure(getTabSizeExtension(tabSize ?? 2))),
  )

  watch(readFontFamily, (family) =>
    reconfigureLayout(fontCompartment.reconfigure(getFontExtension(family ?? ''))),
  )

  watch(readLineHeight, (height) =>
    reconfigureLayout(lineHeightCompartment.reconfigure(getLineHeightExtension(height ?? 1.6))),
  )

  watch(readHighlightActiveLine, (isOn) =>
    view?.dispatch({
      effects: activeLineCompartment.reconfigure(getActiveLineExtension(isOn ?? false)),
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
