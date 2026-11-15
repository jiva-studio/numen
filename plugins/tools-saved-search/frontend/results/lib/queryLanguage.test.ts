import { describe, expect, it } from 'vitest'
import { CompletionContext } from '@codemirror/autocomplete'
import { ensureSyntaxTree } from '@codemirror/language'
import { EditorState, Text } from '@codemirror/state'
import { highlightTree } from '@lezer/highlight'
import {
  createQueryCompletion,
  getQueryDiagnostics,
  queryHighlighting,
  queryLanguage,
  type QueryFunction,
} from './queryLanguage'

const CATALOG: readonly QueryFunction[] = [
  {
    name: 'words',
    kind: 'function',
    signatures: ['words(text)'],
    description: 'Passages that hold every word.',
    example: 'words("seed tray")',
  },
  {
    name: 'tags',
    kind: 'function',
    signatures: ['tags(name)'],
    description: 'Notes with the tag.',
    example: 'tags("garden")',
  },
  { name: 'now', kind: 'variable', signatures: ['now'], description: 'The time now.', example: '' },
  {
    name: 'top',
    kind: 'stage',
    signatures: ['.top(count)'],
    description: 'Keeps the first results.',
    example: 'words("spade").top(5)',
  },
]

const complete = (doc: string, explicit = false) => {
  const state = EditorState.create({ doc })
  const source = createQueryCompletion(() => CATALOG)
  return source(new CompletionContext(state, doc.length, explicit))
}

const getStyles = (doc: string): Array<[string, string]> => {
  const state = EditorState.create({ doc, extensions: queryLanguage })
  const tree = ensureSyntaxTree(state, doc.length, 1000)
  if (!tree) throw new Error('the query was not parsed')
  const styles: Array<[string, string]> = []
  highlightTree(tree, queryHighlighting, (from, to, classes) => {
    styles.push([state.sliceDoc(from, to), classes])
  })
  return styles
}

describe('createQueryCompletion', () => {
  it('lists the functions and variables of the catalog with their signature, description and example', () => {
    const result = complete('wo')

    expect(result?.from).toBe(0)
    expect(result?.options.map((option) => option.label)).toStrictEqual(['words', 'tags', 'now'])
    expect(result?.options[0]).toMatchObject({
      label: 'words',
      type: 'function',
      detail: 'words(text)',
      info: 'Passages that hold every word.\n\nwords("seed tray")',
    })
  })

  it('lists only the stages after a dot', () => {
    const result = complete('words("spade").t')

    expect(result?.from).toBe(15)
    expect(result?.options.map((option) => option.label)).toStrictEqual(['top'])
  })

  it('offers nothing in the middle of nothing unless asked', () => {
    expect(complete('words("spade") && ')).toBeNull()
    expect(complete('words("spade") && ', true)?.options).toHaveLength(3)
  })
})

describe('getQueryDiagnostics', () => {
  it('underlines the name at the line and column of the problem', () => {
    const doc = Text.of(['words("seed")', '  && meanin("a shed")'])

    const [diagnostic] = getQueryDiagnostics(doc, [
      { line: 2, column: 6, message: 'unknown function meanin' },
    ])

    expect(doc.sliceString(diagnostic!.from, diagnostic!.to)).toBe('meanin')
    expect(diagnostic).toMatchObject({ severity: 'error', message: 'unknown function meanin' })
  })

  it('underlines a whole string', () => {
    const doc = Text.of(['words("seed tray)'])

    const [diagnostic] = getQueryDiagnostics(doc, [{ line: 1, column: 7, message: 'unclosed' }])

    expect(doc.sliceString(diagnostic!.from, diagnostic!.to)).toBe('"seed tray)')
  })

  it('keeps a problem past the end of the query inside it', () => {
    const doc = Text.of(['tags("garden")'])

    const [diagnostic] = getQueryDiagnostics(doc, [{ line: 4, column: 40, message: 'late' }])

    expect(diagnostic!.from).toBe(doc.length)
    expect(diagnostic!.to).toBe(doc.length)
  })
})

describe('queryLanguage', () => {
  it('tells apart strings, function names, operators, numbers and stages', () => {
    const styles = new Map(getStyles('!words("seed") || tags("garden").top(5)'))

    expect(styles.get('"seed"')).toBeTruthy()
    expect(styles.get('words')).toBeTruthy()
    expect(styles.get('words')).not.toBe(styles.get('"seed"'))
    expect(styles.get('||')).toBeTruthy()
    expect(styles.get('!')).toBe(styles.get('||'))
    expect(styles.get('top')).toBeTruthy()
    expect(styles.get('top')).not.toBe(styles.get('words'))
    expect(styles.get('5')).toBeTruthy()
  })
})
