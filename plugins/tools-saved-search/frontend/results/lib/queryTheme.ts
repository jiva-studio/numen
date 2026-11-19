/**
 * The query editor set as the notes editor sets a fenced block of code: the face, size, line height,
 * ground and padding of its code, the caret and selection of its theme, the problems underlined in the
 * alarm colour, and the completion list and the problem tooltips drawn as the menus of the interface are.
 */
import { EditorView } from '@codemirror/view'

export const queryTheme = EditorView.theme({
  '&': {
    color: 'var(--numen-ink)',
    backgroundColor: 'var(--numen-code-bg)',
    fontFamily: 'var(--numen-editor-font-family, var(--numen-font-mono))',
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
  '.cm-activeLine': { backgroundColor: 'var(--numen-editor-active-line, transparent)' },
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
  '.cm-lintRange-error': {
    backgroundImage: 'none',
    textDecorationLine: 'underline',
    textDecorationStyle: 'wavy',
    textDecorationColor: 'var(--numen-alarm)',
    textDecorationSkipInk: 'none',
    textUnderlineOffset: '0.2em',
  },
  '.cm-tooltip.cm-tooltip-lint': {
    padding: '0.25rem 0',
    maxWidth: '28rem',
  },
  '.cm-tooltip-lint .cm-diagnostic': {
    padding: '0.125rem 0.625rem',
    borderLeft: 'none',
    color: 'var(--numen-ink)',
    whiteSpace: 'pre-wrap',
  },
  '.cm-tooltip-lint .cm-diagnostic-error': { borderLeft: 'none' },
  '.cm-tooltip.cm-tooltip-autocomplete > ul': {
    fontFamily: 'inherit',
    fontSize: 'var(--numen-text-2)',
    padding: '0.25rem',
    maxHeight: '16rem',
    maxWidth: '28rem',
    minWidth: '16rem',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li': {
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'baseline',
    columnGap: '1rem',
    minHeight: '22px',
    boxSizing: 'border-box',
    lineHeight: '22px',
    color: 'var(--numen-ink)',
    padding: '0 0.5rem',
    borderRadius: 'var(--numen-radius)',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: 'var(--numen-bubble-bg)',
    color: 'var(--numen-ink)',
  },
  '.cm-tooltip-autocomplete .cm-completionLabel': {
    flex: 'none',
    fontFamily: 'var(--numen-font-mono)',
    color: 'var(--numen-ink)',
  },
  '.cm-tooltip-autocomplete .cm-completionMatchedText': {
    textDecoration: 'none',
    fontWeight: '700',
  },
  '.cm-tooltip-autocomplete .cm-completionDetail': {
    flex: '0 1 auto',
    minWidth: '0',
    marginLeft: 'auto',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    color: 'var(--numen-hushed)',
    fontStyle: 'normal',
  },
  '.cm-tooltip-autocomplete .cm-completionIcon': { display: 'none' },
  '.cm-tooltip-autocomplete .numen-query-option': {
    display: 'flex',
    flexBasis: '100%',
    flexDirection: 'column',
    gap: '0.125rem',
    padding: '0 0 0.375rem',
    color: 'var(--numen-hushed)',
    fontSize: 'var(--numen-text-2)',
    lineHeight: '1.4',
    whiteSpace: 'normal',
  },
  '.cm-tooltip-autocomplete .numen-query-option[hidden]': { display: 'none' },
  '.cm-tooltip-autocomplete .numen-query-option code': {
    fontFamily: 'var(--numen-font-mono)',
    overflowWrap: 'anywhere',
  },
  '.cm-tooltip.cm-completionInfo': { display: 'none' },
})
