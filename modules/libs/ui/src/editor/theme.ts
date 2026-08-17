/**
 * What the editor is painted in.
 *
 * The colours it shares with the rest of the product are tokens; the ones only
 * text with marks in it has a use for are named here and nowhere else.
 */
import { EditorView } from '@codemirror/view'

export const theme = EditorView.theme({
  '&': {
    '--editor-mark': 'light-dark(#b3b6bc, #5f656d)',
    '--editor-link': 'var(--numen-ring)',
    '--editor-marker': 'var(--numen-edge-label)',
    '--editor-code-bg': 'light-dark(#f2f2f0, #1d2023)',
    '--editor-head-bg': 'light-dark(#f2f2f0, #22252a)',
    '--editor-mono':
      "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace",
    '--editor-cell-padding': '4px 8px',
    '--editor-grow': '16px',
    '--editor-block-gap': '0.6em',

    '--editor-keyword': 'light-dark(#8250bf, #c39cf0)',
    '--editor-name': 'light-dark(#1f5aa6, #7fb0ea)',
    '--editor-type': 'light-dark(#0a6b74, #63bcc6)',
    '--editor-string': 'light-dark(#2f7a4a, #7fc79b)',
    '--editor-number': 'light-dark(#a35a12, #e0a668)',
    '--editor-comment': 'light-dark(#8a8e96, #7b8089)',
    '--editor-punctuation': 'light-dark(#6c7079, #979da6)',

    color: 'var(--numen-node-fg)',
    backgroundColor: 'transparent',
    fontFamily: 'var(--numen-font-sans)',
    fontSize: 'var(--numen-font-size)',
    lineHeight: '1.6',
    height: '100%',
  },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { fontFamily: 'inherit', lineHeight: 'inherit' },
  '.cm-content': { caretColor: 'var(--numen-node-fg)', padding: '4px 0' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--numen-node-fg)' },
  '.cm-line': { padding: '0 4px' },
  '.cm-placeholder': { color: 'var(--numen-edge-label)' },
  /* Selection, at the reach the editor's own base theme uses for it. That theme
     is told neither light nor dark, so it paints a light ground under text this
     one keeps on either. */
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'light-dark(#2f6fd038, #78a9ef42)',
  },
  '&:not(.cm-focused) > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'light-dark(#2f6fd01f, #78a9ef24)',
  },
  '.cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'light-dark(#2f6fd038, #78a9ef42)',
  },

  /* The ground the base theme lays under a line and a special character is the
     one it picks for a light page. */
  '.cm-activeLine': { backgroundColor: 'transparent' },
  '.cm-specialChar': { color: 'var(--editor-mark)' },
  '.cm-selectionMatch': { backgroundColor: 'light-dark(#2f6fd01a, #78a9ef1f)' },

  /* A heading is set by its level, and nothing else about it changes. */
  '.cm-heading': {
    fontWeight: '600',
    lineHeight: '1.3',
    paddingTop: 'var(--editor-block-gap)',
  },
  '.cm-heading-1': { fontSize: '1.75em' },
  '.cm-heading-2': { fontSize: '1.5em' },
  '.cm-heading-3': { fontSize: '1.3em' },
  '.cm-heading-4': { fontSize: '1.15em' },
  '.cm-heading-5': { fontSize: '1.05em' },
  '.cm-heading-6': { fontSize: '1em' },

  '.cm-strong': { fontWeight: '600' },
  '.cm-em': { fontStyle: 'italic' },
  '.cm-strike': { textDecoration: 'line-through', opacity: '0.7' },
  '.cm-link': { color: 'var(--editor-link)', textDecoration: 'underline' },

  '.cm-code-inline': {
    fontFamily: 'var(--editor-mono)',
    fontSize: '0.9em',
    backgroundColor: 'var(--editor-code-bg)',
    borderRadius: 'var(--numen-radius)',
    padding: '1px 4px',
  },

  /* A fenced block is one slab: every line takes the ground, and the first
     and the last round it off. */
  '.cm-code': {
    fontFamily: 'var(--editor-mono)',
    fontSize: '0.9em',
    backgroundColor: 'var(--editor-code-bg)',
    padding: '0 8px',
  },
  '.cm-code-first': {
    marginTop: 'var(--editor-block-gap)',
    paddingTop: '4px',
    borderTopLeftRadius: 'var(--numen-radius)',
    borderTopRightRadius: 'var(--numen-radius)',
  },
  '.cm-code-last': {
    paddingBottom: '4px',
    borderBottomLeftRadius: 'var(--numen-radius)',
    borderBottomRightRadius: 'var(--numen-radius)',
  },

  /* A quotation is prose a person reads, set apart by its rule and not by
     being harder to see. */
  '.cm-quote': {
    borderLeft: '3px solid var(--numen-ring)',
    paddingLeft: '10px',
    color: 'var(--numen-node-fg)',
    fontStyle: 'italic',
  },

  '.cm-bullet': { color: 'var(--editor-marker)' },
  '.cm-box': {
    verticalAlign: 'middle',
    margin: '0 2px 2px 0',
    accentColor: 'var(--numen-focus-bg)',
    cursor: 'pointer',
  },

  '.cm-rule': { padding: '6px 0' },
  '.cm-rule hr': {
    border: 'none',
    borderTop: '1px solid var(--numen-node-border)',
    margin: '0',
  },

  '.cm-picture img': {
    display: 'block',
    maxWidth: '100%',
    borderRadius: 'var(--numen-radius)',
  },
  '.cm-picture-lost': {
    height: '2em',
    border: '1px dashed var(--numen-node-border)',
    borderRadius: 'var(--numen-radius)',
  },

  /* A table is drawn where its pipes were, and typed into cell by cell. The
     two edges it can grow along carry the button that grows it. */
  '.cm-table': {
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1fr) var(--editor-grow)',
    gridTemplateRows: 'auto var(--editor-grow)',
    gap: '2px',
    padding: 'var(--editor-block-gap) 0',
  },
  '.cm-table table': {
    gridColumn: '1',
    gridRow: '1',
    borderCollapse: 'collapse',
    width: '100%',
    fontSize: '1em',
  },
  '.cm-add-column, .cm-add-row': {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    border: '1px solid transparent',
    borderRadius: 'var(--numen-radius)',
    background: 'transparent',
    color: 'var(--editor-marker)',
    cursor: 'pointer',
    font: 'inherit',
    lineHeight: '1',
    opacity: '0',
    padding: '0',
    transition: 'opacity var(--numen-motion-hover) var(--numen-easing)',
  },
  '.cm-add-column': { gridColumn: '2', gridRow: '1' },
  '.cm-add-row': { gridColumn: '1', gridRow: '2' },
  '.cm-table:hover .cm-add-column, .cm-table:hover .cm-add-row': { opacity: '1' },
  '.cm-add-column:hover, .cm-add-row:hover': {
    borderColor: 'var(--numen-node-border)',
    color: 'var(--numen-node-fg)',
  },
  '.cm-table th, .cm-table td': {
    border: '1px solid var(--numen-node-border)',
    padding: 'var(--editor-cell-padding)',
    textAlign: 'start',
    verticalAlign: 'top',
  },
  '.cm-table th': { backgroundColor: 'var(--editor-head-bg)', fontWeight: '600' },
  '.cm-table .cm-cell:focus': {
    outline: 'var(--numen-ring-width) solid var(--numen-ring)',
    outlineOffset: '-1px',
  },
})
