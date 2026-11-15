import { describe, expect, it } from 'vitest'
import { CompletionContext } from '@codemirror/autocomplete'
import { ensureSyntaxTree } from '@codemirror/language'
import { EditorState, Text } from '@codemirror/state'
import { highlightTree, tags as t } from '@lezer/highlight'
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
    summary: 'passages with these words',
    description: 'Finds passages that contain the words you give.',
    example: 'words("seed tray")',
  },
  {
    name: 'tags',
    kind: 'function',
    summary: 'notes with this tag',
    description: 'Finds notes that carry the tag you give.',
    example: 'tags("garden")',
  },
  {
    name: 'path',
    kind: 'variable',
    summary: 'where the file lies',
    description: 'Where the file lies in the vault.',
    example: '',
  },
  {
    name: 'rerank',
    kind: 'stage',
    summary: 'reorder the best results with a reranking model',
    description: 'Reorders the best results with a reranking model.',
    example: 'words("spade").rerank()',
  },
  {
    name: 'top',
    kind: 'stage',
    summary: 'keep the first N',
    description: 'Keeps only the first results; give how many.',
    example: 'words("spade").top(5)',
  },
  {
    name: 'sortBy',
    kind: 'stage',
    summary: 'put the results in another order',
    description: 'Puts the results in order.',
    example: 'words("spade").sortBy("title")',
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
  it('lists the functions and variables of the catalog, each with what it does in a few words', () => {
    const result = complete('wo')

    expect(result?.from).toBe(0)
    expect(result?.options.map((option) => option.label)).toStrictEqual(['words', 'tags', 'path'])
    expect(result?.options[0]).toMatchObject({
      label: 'words',
      detail: 'passages with these words',
    })
  })

  it('carries the description and the example of each entry, and opens no info panel', () => {
    const option = complete('wo')?.options[0]

    expect(option).toMatchObject({
      entry: {
        description: 'Finds passages that contain the words you give.',
        example: 'words("seed tray")',
      },
    })
    expect(option?.info).toBeUndefined()
    expect(option?.type).toBeUndefined()
  })

  it('lists exactly the stages, in plain words, after a dot that follows a search', () => {
    const result = complete('meaning("x").')

    expect(result?.from).toBe(13)
    expect(result?.options.map((option) => [option.label, option.detail])).toStrictEqual([
      ['rerank', 'reorder the best results with a reranking model'],
      ['top', 'keep the first N'],
      ['sortBy', 'put the results in another order'],
    ])
  })

  it('lists the stages after a dot on the next line', () => {
    expect(complete('meaning("x")\n  .')?.options.map((option) => option.label)).toStrictEqual([
      'rerank',
      'top',
      'sortBy',
    ])
  })

  it('lists the methods of a text after a dot that follows a variable', () => {
    const result = complete('path.')

    expect(result?.options.map((option) => [option.label, option.detail])).toStrictEqual([
      ['startsWith', 'begins with this text'],
    ])
  })

  it('opens no list after the dot of a number', () => {
    expect(complete('top(1.')).toBeNull()
  })

  it('lists only the stages after a dot', () => {
    const result = complete('words("spade").t')

    expect(result?.from).toBe(15)
    expect(result?.options.map((option) => option.label)).toStrictEqual(['rerank', 'top', 'sortBy'])
  })

  it('returns no completion after a space unless the completion is explicit', () => {
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

  it('underlines the last character for a problem past the end of the query', () => {
    const doc = Text.of(['tags("garden")'])

    const [diagnostic] = getQueryDiagnostics(doc, [{ line: 4, column: 40, message: 'late' }])

    expect(doc.sliceString(diagnostic!.from, diagnostic!.to)).toBe(')')
  })

  it('underlines to the end of the line from a place that is not a name or a string', () => {
    const doc = Text.of(['words("gale") && ) tags("pier")'])

    const [diagnostic] = getQueryDiagnostics(doc, [{ line: 1, column: 18, message: 'stray' }])

    expect(doc.sliceString(diagnostic!.from, diagnostic!.to)).toBe(') tags("pier")')
  })

  it('underlines a problem with an end from its start through its last character', () => {
    const doc = Text.of(['meaning("a") && Мадхавендра Пури'])

    const [diagnostic] = getQueryDiagnostics(doc, [
      {
        line: 1,
        column: 17,
        endLine: 1,
        endColumn: 32,
        message: 'Put this text in quotes: Мадхавендра Пури',
      },
    ])

    expect(doc.sliceString(diagnostic!.from, diagnostic!.to)).toBe('Мадхавендра Пури')
  })

  it('underlines a problem with an end over the lines it spans', () => {
    const doc = Text.of(['words("a") && Мадхавендра', 'Пури || tags("b")'])

    const [diagnostic] = getQueryDiagnostics(doc, [
      { line: 1, column: 15, endLine: 2, endColumn: 4, message: 'Put this text in quotes' },
    ])

    expect(doc.sliceString(diagnostic!.from, diagnostic!.to)).toBe('Мадхавендра\nПури')
  })

  it('shows the message of the core as it is', () => {
    const doc = Text.of(['words("gale")', 'e'])

    const [diagnostic] = getQueryDiagnostics(doc, [
      { line: 2, column: 1, message: 'Put this text in quotes: e' },
    ])

    expect(doc.sliceString(diagnostic!.from, diagnostic!.to)).toBe('e')
    expect(diagnostic!.message).toBe('Put this text in quotes: e')
  })
})

describe('queryLanguage', () => {
  it('styles strings, names, operators and numbers apart, and stages as names', () => {
    const styles = new Map(getStyles('!words("seed") || tags("garden").top(5)'))

    expect(styles.get('"seed"')).toBeTruthy()
    expect(styles.get('words')).toBeTruthy()
    expect(styles.get('words')).not.toBe(styles.get('"seed"'))
    expect(styles.get('||')).toBeTruthy()
    expect(styles.get('!')).toBe(styles.get('||'))
    expect(styles.get('||')).not.toBe(styles.get('words'))
    expect(styles.get('top')).toBe(styles.get('words'))
    expect(styles.get('5')).toBeTruthy()
    expect(styles.get('5')).not.toBe(styles.get('"seed"'))
  })

  it.each([
    ['// x', '// x'],
    ['meaning("a") // x', '// x'],
  ])('styles the comment of %s as a comment', (query, comment) => {
    const styles = getStyles(query)
    const commentStyle = queryHighlighting.style([t.comment])

    expect(commentStyle).toBeTruthy()
    expect(new Map(styles).get(comment)).toBe(commentStyle)
    expect(styles.filter(([text]) => text.includes('x'))).toStrictEqual([[comment, commentStyle]])
  })
})
