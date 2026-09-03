/**
 * What the editor is painted in. Every colour is a token, and the `--editor-`
 * names are this file's own.
 *
 * Three of them are left to whoever puts the editor on the screen, set on the
 * element the editor is drawn in: `--editor-measure`, how wide the column of
 * words may be, `--editor-lead`, the room above the first line, and
 * `--editor-margin`, the room either side of the column and its gutter.
 */
import { EditorView } from '@codemirror/view'

/**
 * A whole document of code, set in the face the marks inside prose are set in.
 * The family is the one this file names once.
 */
export const monospaced = EditorView.theme({
  '&': { fontFamily: 'var(--editor-mono)' },
})

export const theme = EditorView.theme({
  '&': {
    '--editor-mark': 'var(--numen-syntax-mark)',
    '--editor-link': 'var(--numen-ring)',
    '--editor-marker': 'var(--numen-edge-label)',
    '--editor-code-bg': 'var(--numen-code-bg)',
    '--editor-head-bg': 'var(--numen-table-head-bg)',
    '--editor-mono':
      "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace",
    '--editor-cell-padding': '4px 8px',
    '--editor-grow': '16px',
    '--editor-block-gap': '0.6em',
    /* The room above the first line. */
    '--editor-room': '0.5rem',
    /* How thick a quotation's rule is, and how much of the gutter it takes. */
    '--editor-quote-rule': '3px',
    '--editor-quote-room': '0.5rem',

    '--editor-keyword': 'var(--numen-syntax-keyword)',
    '--editor-name': 'var(--numen-syntax-name)',
    '--editor-type': 'var(--numen-syntax-type)',
    '--editor-string': 'var(--numen-syntax-string)',
    '--editor-number': 'var(--numen-syntax-number)',
    '--editor-comment': 'var(--numen-syntax-comment)',
    '--editor-punctuation': 'var(--numen-syntax-punctuation)',

    color: 'var(--numen-node-fg)',
    backgroundColor: 'transparent',
    fontFamily: 'var(--numen-font-sans)',
    fontSize: 'var(--numen-reading-size)',
    lineHeight: '1.6',
    height: '100%',
  },
  '&.cm-focused': { outline: 'none' },
  /* The words stand in the middle of whatever room there is, which is where a
     measure narrower than the room puts them. */
  '.cm-scroller': {
    fontFamily: 'inherit',
    lineHeight: 'inherit',
    justifyContent: 'center',
    paddingInline: 'var(--editor-margin, 0)',
  },
  /* The prose opens close to the top of its box and keeps the gutter's room
     below it, so the last line can be brought clear of the edge. */
  '.cm-content': {
    caretColor: 'var(--numen-node-fg)',
    paddingTop: 'var(--editor-lead, var(--editor-room))',
    paddingBottom: 'var(--numen-gutter)',
    maxWidth: 'var(--editor-measure, none)',
  },
  '.cm-gutters': { maxWidth: 'var(--editor-measure, none)' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--numen-node-fg)' },
  '.cm-line': { padding: '0 var(--numen-gutter)' },
  '.cm-placeholder': { color: 'var(--numen-edge-label)' },
  /* The editor draws a buffer either side of replaced content and measures it
     for cursor coordinates. Its margin box is a point on the baseline, so the
     line stands as tall as its text in any font, and its own box is that text's
     height. */
  '.cm-widgetBuffer': { verticalAlign: 'baseline', height: '1em', marginTop: '-1em' },
  /* Selection, the one colour selected text is painted with anywhere. The base
     theme is told neither light nor dark, so it paints a light ground under
     text this one keeps on either. */
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--numen-selection)',
  },
  '&:not(.cm-focused) > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--numen-selection-away)',
  },
  '.cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'var(--numen-selection)',
  },

  /* The ground the base theme lays under a line and a special character is the
     one it picks for a light page. */
  '.cm-activeLine': { backgroundColor: 'transparent' },
  '.cm-specialChar': { color: 'var(--editor-mark)' },
  '.cm-selectionMatch': { backgroundColor: 'var(--numen-selection-match)' },

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

  /* Nothing stands above the first line, so it takes no gap. */
  '.cm-content > .cm-line:first-child': { paddingTop: '0' },

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
     being harder to see. The rule stands in the gutter, so the words of a
     quotation begin where every other line's words begin. */
  '.cm-quote': {
    borderLeft: 'var(--editor-quote-rule) solid var(--numen-ring)',
    marginLeft: 'calc(var(--numen-gutter) - var(--editor-quote-room))',
    paddingLeft: 'calc(var(--editor-quote-room) - var(--editor-quote-rule))',
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
  /* A stretch something other than the reader is about to change. The words
     that arrive in it fade up on the product's own keyframes. */
  '.cm-changing': {
    backgroundColor: 'var(--numen-highlight)',
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
