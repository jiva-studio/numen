/**
 * The query editor set as the notes editor sets a fenced block of code: the face, size, line height,
 * ground and padding of its code, the caret and selection of its theme, and the completion list drawn
 * as the menus of the interface are.
 */
import { EditorView } from '@codemirror/view'

export const queryTheme = EditorView.theme({
  '&': {
    color: 'var(--numen-ink)',
    backgroundColor: 'var(--numen-code-bg)',
    fontFamily: 'var(--numen-font-mono)',
    fontSize: 'var(--numen-reading-size)',
    lineHeight: '1.6',
  },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { fontFamily: 'inherit', lineHeight: 'inherit' },
  '.cm-content': {
    caretColor: 'var(--numen-ink)',
    fontSize: '0.9em',
    padding: '4px 0',
  },
  '.cm-line': { padding: '0 8px' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--numen-ink)' },
  '.cm-activeLine': { backgroundColor: 'transparent' },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--numen-selection)',
  },
  '&:not(.cm-focused) > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--numen-selection-away)',
  },
  '.cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'var(--numen-selection)',
  },
  '.cm-selectionMatch': { backgroundColor: 'var(--numen-selection-match)' },
  '&.cm-focused .cm-matchingBracket, .cm-matchingBracket': {
    backgroundColor: 'var(--numen-selection-match)',
  },
  '&.cm-focused .cm-nonmatchingBracket, .cm-nonmatchingBracket': {
    backgroundColor: 'var(--numen-alarm-bg)',
  },
  '.cm-tooltip': {
    backgroundColor: 'var(--numen-raised)',
    color: 'var(--numen-ink)',
    border: '1px solid var(--numen-panel-border)',
    borderRadius: 'var(--numen-radius)',
    boxShadow: 'var(--numen-panel-shadow)',
    fontFamily: 'var(--numen-font-sans)',
    fontSize: 'var(--numen-font-size)',
    overflow: 'hidden',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul': {
    fontFamily: 'inherit',
    padding: '0.375rem',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li': {
    color: 'var(--numen-ink)',
    padding: '0.125rem 0.5rem',
    borderRadius: 'var(--numen-radius)',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: 'var(--numen-bubble-bg)',
    color: 'var(--numen-ink)',
  },
  '.cm-tooltip-autocomplete .cm-completionMatchedText': {
    textDecoration: 'none',
    fontWeight: '600',
    backgroundColor: 'var(--numen-highlight)',
    borderRadius: '2px',
  },
  '.cm-tooltip-autocomplete .cm-completionDetail, .cm-tooltip-autocomplete .cm-completionIcon': {
    color: 'var(--numen-hushed)',
    fontStyle: 'normal',
  },
  '.cm-tooltip.cm-completionInfo': {
    color: 'var(--numen-ink)',
    backgroundColor: 'var(--numen-raised)',
  },
})
