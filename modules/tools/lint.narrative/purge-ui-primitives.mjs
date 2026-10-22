#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2)
const isDryRun = args.includes('--dry-run')
const pathArg = args.find(a => !a.startsWith('--'))
const targetDir = pathArg ? path.resolve(pathArg) : path.resolve(import.meta.dirname, '../../..')

const IGNORED_DIRS = new Set([
  'node_modules',
  '.git',
  'dist',
  '.direnv',
  '.claude',
  'test-results',
  'coverage',
  'testdata',
])

const EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.vue'])

let totalFixedIssues = 0
let totalModifiedFiles = 0

// File-specific exact or pattern-based transformations
function transformFile(relPath, content) {
  let modified = false
  let count = 0
  let newContent = content

  // 1. namedAt / named_at -> nameAt (in agent conversation tests)
  if (relPath.includes('useAgentConversation') && newContent.includes('namedAt')) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly namedAt: readonly string[]', 'readonly nameAt: readonly string[]')
      .replaceAll('namedAt: readonly string[]', 'nameAt: readonly string[]')
      .replaceAll('namedAt,', 'nameAt,')
      .replaceAll('namedAt.', 'nameAt.')
      .replaceAll('one.namedAt', 'one.nameAt')
      .replaceAll('(namedAt: readonly string[]', '(nameAt: readonly string[]')
    if (newContent !== before) {
      count += (before.match(/namedAt/g) || []).length - (newContent.match(/namedAt/g) || []).length
      modified = true
    }
  }

  // 2. turnTo -> goToPage
  if (newContent.includes('turnTo')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export function turnTo(', 'export function goToPage(')
      .replaceAll('import { turnTo }', 'import { goToPage }')
      .replaceAll(', turnTo }', ', goToPage }')
      .replaceAll(', turnTo,', ', goToPage,')
      .replaceAll('{ turnTo,', '{ goToPage,')
      .replaceAll("describe('turnTo',", "describe('goToPage',")
      .replaceAll('turnTo(way,', 'goToPage(way,')
      .replaceAll('const to = turnTo(', 'const to = goToPage(')
    if (newContent !== before) {
      count += (before.match(/\bturnTo\b/g) || []).length - (newContent.match(/\bturnTo\b/g) || []).length
      modified = true
    }
  }

  // 3. swipeTurn -> swipePage
  if (newContent.includes('swipeTurn')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export function swipeTurn(', 'export function swipePage(')
      .replaceAll('swipeTurn(to - from)', 'swipePage(to - from)')
      .replaceAll('swipeTurn(by: number)', 'swipePage(delta: number)')
      .replaceAll('swipeTurn(by)', 'swipePage(by)')
      .replaceAll('import { EDGE, SWIPE, handTurn, pressTurn, swipeTurn, turnTo }', 'import { EDGE, SWIPE, goToPage, handTurn, pressTurn, swipePage }')
      .replaceAll('import { EDGE, SWIPE, handTurn, pressTurn, swipeTurn, goToPage }', 'import { EDGE, SWIPE, goToPage, handTurn, pressTurn, swipePage }')
      .replaceAll("describe('swipeTurn and pressTurn',", "describe('swipePage and pressTurn',")
      .replaceAll('swipeTurn(-SWIPE)', 'swipePage(-SWIPE)')
      .replaceAll('swipeTurn(SWIPE)', 'swipePage(SWIPE)')
      .replaceAll('swipeTurn(0)', 'swipePage(0)')
      .replaceAll('swipeTurn(SWIPE - 1)', 'swipePage(SWIPE - 1)')
      .replaceAll('swipeTurn(1 - SWIPE)', 'swipePage(1 - SWIPE)')
      .replaceAll('swipeTurn(', 'swipePage(')
    if (newContent !== before) {
      count += (before.match(/\bswipeTurn\b/g) || []).length - (newContent.match(/\bswipeTurn\b/g) || []).length
      modified = true
    }
  }

  // 4. ticket -> token (in views.plex.graph)
  if (relPath.includes('views.plex.graph') && (newContent.includes('ticket:') || newContent.includes('ticket(') || newContent.includes('ticket'))) {
    const before = newContent
    newContent = newContent
      .replaceAll('ticket: (path: string) => string,', 'token: (path: string) => string,')
      .replaceAll('id: ticket(neighbourhood.focus.path),', 'id: token(neighbourhood.focus.path),')
      .replaceAll('const id = ticket(related.path)', 'const id = token(related.path)')
      .replaceAll('from: ticket(through)', 'from: token(through)')
      .replaceAll('ticket: (path: string)', 'token: (path: string)')
      .replaceAll("it('is the ticket the note holds, and never the file it is filed under'", "it('is the token the note holds, and never the file it is filed under'")
      .replaceAll('A ticket no note holds', 'A token no note holds')
      .replaceAll("what a note's ticket is", "what a note's token is")
      .replaceAll('ticket its note holds', 'token its note holds')
    if (newContent !== before) {
      count += (before.match(/\bticket\b/g) || []).length - (newContent.match(/\bticket\b/g) || []).length
      modified = true
    }
  }

  // 5. closer / further -> zoomIn / zoomOut
  if (
    relPath.includes('features/reader') ||
    relPath.includes('readers.epub/frontend/words.ts') ||
    relPath.includes('readers.pdf/frontend/words.ts')
  ) {
    const before = newContent
    if (relPath.endsWith('strip.ts')) {
      newContent = newContent
        .replaceAll('readonly closer: string\n  readonly further: string', 'readonly zoomIn: string\n  readonly zoomOut: string')
        .replaceAll("closer: 'Closer',\n  further: 'Further',", "zoomIn: 'Closer',\n  zoomOut: 'Further',")
    }
    if (relPath.endsWith('ZoomControls.vue')) {
      newContent = newContent
        .replaceAll(':aria-label="words.further"', ':aria-label="words.zoomOut"')
        .replaceAll(':aria-label="words.closer"', ':aria-label="words.zoomIn"')
    }
    if (relPath.endsWith('ReaderToolbar.test.ts')) {
      newContent = newContent
        .replaceAll('READER_WORDS.closer', 'READER_WORDS.zoomIn')
        .replaceAll('READER_WORDS.further', 'READER_WORDS.zoomOut')
    }
    if (relPath.includes('readers.epub') && relPath.endsWith('words.ts')) {
      newContent = newContent
        .replaceAll("closer: 'Larger',", "zoomIn: 'Larger',")
        .replaceAll("further: 'Smaller',", "zoomOut: 'Smaller',")
    }
    if (relPath.includes('readers.pdf') && relPath.endsWith('words.ts')) {
      newContent = newContent
        .replaceAll("closer: 'Closer',", "zoomIn: 'Closer',")
        .replaceAll("further: 'Further',", "zoomOut: 'Further',")
    }
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  // 6. nth -> index (in cards deck editor & views.flashcards.studio)
  if (
    (relPath.includes('features/cards') || relPath.includes('views.flashcards.studio/frontend/deck')) &&
    !relPath.endsWith('.vue')
  ) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly nth: number', 'readonly index: number')
      .replaceAll('nth: number', 'index: number')
      .replaceAll('const nth = countOff(each.field)', 'const index = countOff(each.field)')
      .replaceAll('const nth = (underItem.get(field) ?? 0) + 1', 'const index = (underItem.get(field) ?? 0) + 1')
      .replaceAll('underItem.set(field, nth)', 'underItem.set(field, index)')
      .replaceAll('return nth', 'return index')
      .replaceAll('nth, key: `${each.field}#${nth}`', 'index, key: `${each.field}#${index}`')
      .replaceAll('isLast: each.nth === (underItem.get(each.field) ?? 0),', 'isLast: each.index === (underItem.get(each.field) ?? 0),')
      .replaceAll('each.nth', 'each.index')
      .replaceAll('nth: 1,', 'index: 1,')
      .replaceAll('nth: 1 },', 'index: 1 },')
      .replaceAll('underCount === nth', 'underCount === index')
      .replaceAll('underCount !== nth', 'underCount !== index')
      .replaceAll('underCount < nth', 'underCount < index')
      .replaceAll('onWrite(field, nth, text)', 'onWrite(field, index, text)')
      .replaceAll('{ field, nth },', '{ field, index },')
      .replaceAll('{ field, nth }', '{ field, index }')
      .replaceAll('{ field, nth, text }', '{ field, index, text }')
      .replaceAll('target.nth', 'target.index')
      .replaceAll('nth: props.value.nth', 'index: props.value.index')
      .replaceAll('const { field, nth, text } = target', 'const { field, index, text } = target')
      .replaceAll('const onWrite = (field: string, nth: number, text: string)', 'const onWrite = (field: string, index: number, text: string)')
      .replaceAll('onWrite: (id: string, field: string, nth: number, text: string)', 'onWrite: (id: string, field: string, index: number, text: string)')
      .replaceAll('writeCardField: (...[card, field, nth, text]: [string, string, number, string]) =>', 'writeCardField: (...[card, field, index, text]: [string, string, number, string]) =>')
      .replaceAll('writeCardField(card: string, field: string, nth: number, text: string): void', 'writeCardField(card: string, field: string, index: number, text: string): void')
      .replaceAll('const replaceCardValue = (deck: BufferDeck, id: string, field: string, nth: number, text: string)', 'const replaceCardValue = (deck: BufferDeck, id: string, field: string, index: number, text: string)')
      .replaceAll('replaceCardValueInternal(deck, id, { field, nth, text })', 'replaceCardValueInternal(deck, id, { field, index, text })')
      .replaceAll('updateDeck(id, replaceCardValue(getDeck(id), card, { field, nth, text }))', 'updateDeck(id, replaceCardValue(getDeck(id), card, { field, index, text }))')
      .replaceAll('target: { field: string; nth: number }', 'target: { field: string; index: number }')
    if (newContent !== before) {
      count += (before.match(/\bnth\b/g) || []).length - (newContent.match(/\bnth\b/g) || []).length
      modified = true
    }
  }

  // Vue files for nth: DeckEditor.vue, CardValue.vue, DeckTab.vue
  if (relPath.endsWith('DeckEditor.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll("(event: 'write', card: string, field: string, nth: number, text: string): void", "(event: 'write', card: string, field: string, index: number, text: string): void")
      .replaceAll("write: (card, target, text) => emit('write', card, target.field, target.nth, text),", "write: (card, target, text) => emit('write', card, target.field, target.index, text),")
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }
  if (relPath.endsWith('CardValue.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('{ field: props.value.field, nth: props.value.nth }', '{ field: props.value.field, index: props.value.index }')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }
  if (relPath.endsWith('DeckTab.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('function onWriteCard(...[id, field, nth, text]: [string, string, number, string]) {', 'function onWriteCard(...[id, field, index, text]: [string, string, number, string]) {')
      .replaceAll('props.state.writeCardField(id, field, nth, text)', 'props.state.writeCardField(id, field, index, text)')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  // 7. Parameter Prepositions: (by: number) -> (delta: number)
  if (relPath.includes('features/book/lib/turn.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('swipePage(by: number)', 'swipePage(delta: number)')
      .replaceAll('swipeTurn(by: number)', 'swipePage(delta: number)')
      .replaceAll('if (by <= -SWIPE)', 'if (delta <= -SWIPE)')
      .replaceAll('if (by >= SWIPE)', 'if (delta >= SWIPE)')
      .replaceAll('readonly width: number', 'readonly wide: number')
      .replaceAll('pressTurn(to, options.width)', 'pressTurn(to, options.wide)')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/editor/lib/grid.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const shiftCells = (row: Row, by: number): Row =>', 'const shiftCells = (row: Row, delta: number): Row =>')
      .replaceAll('cell.from + by, to: cell.to + by', 'cell.from + delta, to: cell.to + delta')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/notices/model/cards.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const wind = async (by: number): Promise<void> => {', 'const wind = async (delta: number): Promise<void> => {')
      .replaceAll('clock.at += by', 'clock.at += delta')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/notices/ui/Notices.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('wind: (by: number) => (clock.at += by)', 'wind: (delta: number) => (clock.at += delta)')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/palette/lib/item.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const getSteppedIndex = (total: number, from: number, by: number): number => {', 'export const getSteppedIndex = (total: number, from: number, delta: number): number => {')
      .replaceAll('(((from + by) % total) + total) % total', '(((from + delta) % total) + total) % total')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/palette/lib/parts.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const clampToText = (text: string, at: number): number =>', 'const clampToText = (text: string, pos: number): number =>')
      .replaceAll('Math.trunc(at) || 0', 'Math.trunc(pos) || 0')
      .replaceAll('const whole = (text: string, at: number, by: 1 | -1): number => {', 'const whole = (text: string, pos: number, delta: 1 | -1): number => {')
      .replaceAll('let here = at\n  while (isInsideCharacter(text, here)) here += by', 'let here = pos\n  while (isInsideCharacter(text, here)) here += delta')
      .replaceAll('const isInsideCharacter = (text: string, at: number): boolean =>', 'const isInsideCharacter = (text: string, pos: number): boolean =>')
      .replaceAll('isTrailing(text.charCodeAt(at)) && isLeading(text.charCodeAt(at - 1))', 'isTrailing(text.charCodeAt(pos)) && isLeading(text.charCodeAt(pos - 1))')
    if (newContent !== before) {
      count += 3
      modified = true
    }
  }

  if (relPath.includes('features/palette/model/keys.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly stepBy: (by: number, from?: number) => void', 'readonly stepBy: (delta: number, from?: number) => void')
      .replaceAll('const stepBy = (by: number, from = places.here.value): void => {', 'const stepBy = (delta: number, from = places.here.value): void => {')
      .replaceAll('{ from, by }, canLandOn', '{ from, by: delta }, canLandOn')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('features/palette/ui/palette-actions/PaletteActions.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const step = (by: number, from = here.value) => {', 'const step = (delta: number, from = here.value) => {')
      .replaceAll('getSteppedIndex(actions.value.length, from, by)', 'getSteppedIndex(actions.value.length, from, delta)')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/plex/model/plexScale.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const larger = (size: Size, by: number): Size => ({', 'const larger = (size: Size, delta: number): Size => ({')
      .replaceAll('size.width * by', 'size.width * delta')
      .replaceAll('size.height * by', 'size.height * delta')
      .replaceAll('export const scaleOptions = (options: PlexOptions, by: number): PlexOptions => ({', 'export const scaleOptions = (options: PlexOptions, delta: number): PlexOptions => ({')
      .replaceAll('larger(options.focusSize, by)', 'larger(options.focusSize, delta)')
      .replaceAll('larger(options.nodeSize, by)', 'larger(options.nodeSize, delta)')
      .replaceAll('options.minWidth * by', 'options.minWidth * delta')
      .replaceAll('options.iconWidth * by', 'options.iconWidth * delta')
      .replaceAll('options.headingHeight * by', 'options.headingHeight * delta')
      .replaceAll('options.headingIndent * by', 'options.headingIndent * delta')
      .replaceAll('options.gap * by', 'options.gap * delta')
      .replaceAll('options.lineGap * by', 'options.lineGap * delta')
      .replaceAll('options.focusGap * by', 'options.focusGap * delta')
      .replaceAll('options.margin * by', 'options.margin * delta')
      .replaceAll('options.routing.minReach * by', 'options.routing.minReach * delta')
      .replaceAll('options.routing.arrowRoom * by', 'options.routing.arrowRoom * delta')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('shared/lib/bounds.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const stepBy = (value: number, by: number, bounds: Bounds): number => {', 'export const stepBy = (value: number, delta: number, bounds: Bounds): number => {')
      .replaceAll('if (bounds.step <= 0 || by === 0) return from', 'if (bounds.step <= 0 || delta === 0) return from')
      .replaceAll('const place = (by > 0 ? Math.floor(at) : Math.ceil(at)) + by', 'const place = (delta > 0 ? Math.floor(at) : Math.ceil(at)) + delta')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('shared/ui/menu/keys.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const step = (by: number, from: number): void => {', 'const step = (delta: number, from: number): void => {')
      .replaceAll('{ from, by }, canLandOn', '{ from, by: delta }, canLandOn')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('shared/ui/number-field/number.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const stepBy = (value: number | null, by: number, bounds: Bounds): number =>', 'export const stepBy = (value: number | null, delta: number, bounds: Bounds): number =>')
      .replaceAll('snapToBounds((value ?? bounds.min) + by * bounds.step, bounds)', 'snapToBounds((value ?? bounds.min) + delta * bounds.step, bounds)')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  // 8. Parameter Prepositions: (wide: number) -> (width: number)
  if (relPath.includes('features/book/lib/turn.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export function pressTurn(x: number, wide: number): PageNavigationAction | undefined {', 'export function pressTurn(x: number, width: number): PageNavigationAction | undefined {')
      .replaceAll('if (wide <= 0) return undefined', 'if (width <= 0) return undefined')
      .replaceAll('if (x <= wide * EDGE) return \'back\'', 'if (x <= width * EDGE) return \'back\'')
      .replaceAll('if (x >= wide * (1 - EDGE)) return \'next\'', 'if (x >= width * (1 - EDGE)) return \'next\'')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/book/ui/Book.stories.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('(wide: number): Decorator =>', '(width: number): Decorator =>')
      .replaceAll('inline-size: ${wide}px;', 'inline-size: ${width}px;')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/book/ui/Book.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const setSize = (area: HTMLElement, wide: number, high: number): void => {', 'const setSize = (area: HTMLElement, width: number, high: number): void => {')
      .replaceAll("Object.defineProperty(area, 'clientWidth', { value: wide, configurable: true })", "Object.defineProperty(area, 'clientWidth', { value: width, configurable: true })")
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/reader/ui/Reader.stories.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const renderPage = (label: string, wide: number): string => {', 'const renderPage = (label: string, width: number): string => {')
      .replaceAll('Math.round((wide * 792) / 612)', 'Math.round((width * 792) / 612)')
      .replaceAll('width="${wide}"', 'width="${width}"')
      .replaceAll('font-size="${Math.round(wide / 5)}"', 'font-size="${Math.round(width / 5)}"')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/reader/ui/Reader.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const reader = async (wide: number, high: number) => {', 'const reader = async (width: number, high: number) => {')
      .replaceAll('const reader = async (width: number, high: number) => {\n  const wrapper = mount(Reader, {\n    props: { pages: PAGES, picture: (page: number) => `/p/${page}` },\n    attachTo: document.body,\n  })\n  await room(wrapper, width, height)',
                  'const reader = async (width: number, high: number) => {\n  const wrapper = mount(Reader, {\n    props: { pages: PAGES, picture: (page: number) => `/p/${page}` },\n    attachTo: document.body,\n  })\n  await room(wrapper, width, high)')
      .replaceAll('const room = async (wrapper: ReturnType<typeof mount>, wide: number, high: number) => {', 'const room = async (wrapper: ReturnType<typeof mount>, width: number, height: number) => {')
      .replaceAll('const room = async (wrapper: ReturnType<typeof mount>, width: number, high: number) => {', 'const room = async (wrapper: ReturnType<typeof mount>, width: number, height: number) => {')
      .replaceAll("Object.defineProperty(area, 'clientWidth', { value: wide, configurable: true })", "Object.defineProperty(area, 'clientWidth', { value: width, configurable: true })")
      .replaceAll("Object.defineProperty(area, 'clientHeight', { value: high, configurable: true })", "Object.defineProperty(area, 'clientHeight', { value: height, configurable: true })")
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('readers.pdf/frontend/DocumentTab.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('function onWiden(wide: number) {\n  activeState.value?.widen(wide)\n}', 'function onWiden(width: number) {\n  activeState.value?.widen(width)\n}')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('readers.pdf/frontend/api/wire.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('getPageUrl: (...[path, pageNumber, wide, fingerprint = \'\']: [string, number, number, string?]) =>\n    `${getAssetUrl(path)}/pages/${pageNumber}?wide=${wide}&${getBytesQuery(fingerprint)}`,',
                  'getPageUrl: (...[path, pageNumber, width, fingerprint = \'\']: [string, number, number, string?]) =>\n    `${getAssetUrl(path)}/pages/${pageNumber}?wide=${width}&${getBytesQuery(fingerprint)}`,')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('readers.pdf/frontend/model/useDocumentReader.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('getPageUrl: (path, pageNumber, wide) => `${path} ${pageNumber} ${wide}`,',
                  'getPageUrl: (path, pageNumber, width) => `${path} ${pageNumber} ${width}`,')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('views.flashcards.studio/frontend/preset/lib/label.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const box = (x: number, y: number, wide: number, high: number): Box => ({ x, y, wide, high })',
                  'const box = (x: number, y: number, width: number, height: number): Box => ({ x, y, wide: width, high: height })')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  // 9. Parameter Prepositions: (at: number) -> (pos: number)
  if (relPath.includes('features/book/model/layout.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly onMove: (at: number) => void', 'readonly onMove: (pos: number) => void')
      .replaceAll('readonly getSpreadOf: (at: number) => number', 'readonly getSpreadOf: (pos: number) => number')
      .replaceAll('(at) => {\n      if (at < props.span.from || at >= props.span.to) return\n      const want = context.getSpreadOf(at)',
                  '(pos) => {\n      if (pos < props.span.from || pos >= props.span.to) return\n      const want = context.getSpreadOf(pos)')
    if (newContent !== before) {
      count += 3
      modified = true
    }
  }

  if (relPath.includes('features/book/model/links.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const reportMove = vi.fn<(at: number) => void>()', 'const reportMove = vi.fn<(pos: number) => void>()')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/book/model/links.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly reportMove: (at: number) => void', 'readonly reportMove: (pos: number) => void')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/book/model/turns.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const onMove = vi.fn<(at: number) => void>()', 'const onMove = vi.fn<(pos: number) => void>()')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/book/model/turns.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly onMove: (at: number) => void', 'readonly onMove: (pos: number) => void')
      .replaceAll('const getSpreadOf = (at: number) => findSpreadAt(options.marks.value, options.flow.value, at)',
                  'const getSpreadOf = (pos: number) => findSpreadAt(options.marks.value, options.flow.value, pos)')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('features/book/ui/Book.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const press = (at: number) => {', 'const press = (pos: number) => {')
      .replaceAll("clientX: at", "clientX: pos")
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/book/ui/Book.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll("(event: 'move', at: number): void", "(event: 'move', pos: number): void")
      .replaceAll("reportMove: (at) => emit('move', at),", "reportMove: (pos) => emit('move', pos),")
      .replaceAll("onMove: (at) => emit('move', at),", "onMove: (pos) => emit('move', pos),")
    if (newContent !== before) {
      count += 3
      modified = true
    }
  }

  if (relPath.includes('features/book/ui/book-contents/BookContents.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll("(event: 'go', at: number): void", "(event: 'go', pos: number): void")
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/book/ui/book-contents/contents.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export function findLineAt(entries: readonly ContentsEntry[], at: number): number {\n  let found = -1\n  for (let i = 0; i < entries.length; i++) {\n    if (entries[i]!.at > at) break',
                  'export function findLineAt(entries: readonly ContentsEntry[], pos: number): number {\n  let found = -1\n  for (let i = 0; i < entries.length; i++) {\n    if (entries[i]!.at > pos) break')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/cards/lib/fill.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export function insert(template: string, at: number, field: string): InsertResult {\n  const where = Math.max(0, Math.min(at, template.length))',
                  'export function insert(template: string, pos: number, field: string): InsertResult {\n  const where = Math.max(0, Math.min(pos, template.length))')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/cards/lib/safe.stories.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const createArrangement = (next: () => number, at: number): string => {\n  if (at < KNOWN.length) return KNOWN[at]!',
                  'const createArrangement = (next: () => number, pos: number): string => {\n  if (pos < KNOWN.length) return KNOWN[pos]!')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/editor/lib/address.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const open = (at: number | null, doc = SOURCE) => {', 'const open = (pos: number | null, doc = SOURCE) => {')
      .replaceAll("mockReturnValue(at)", "mockReturnValue(pos)")
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/editor/lib/marks.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('add: (at: number, first: boolean, isLast: boolean) => void,', 'add: (pos: number, first: boolean, isLast: boolean) => void,')
      .replaceAll('(at) => this.found.push(line(\'cm-quote\').range(at))', '(pos) => this.found.push(line(\'cm-quote\').range(pos))')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('features/editor/lib/replace.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const getLineColumn = (doc: Text, at: number): Place => {\n  const line = doc.lineAt(at)\n  return { line: line.number, column: at - line.from }',
                  'const getLineColumn = (doc: Text, pos: number): Place => {\n  const line = doc.lineAt(pos)\n  return { line: line.number, column: pos - line.from }')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/editor/lib/replace.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const place = (at: number) => {\n    const line = updatedState.doc.lineAt(at)\n    return [line.number, at - line.from]',
                  'const place = (pos: number) => {\n    const line = updatedState.doc.lineAt(pos)\n    return [line.number, pos - line.from]')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/notices/lib/duration.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('notices: readonly Notice[],\n  at: number,\n): ReadonlyMap<string, number> =>\n  new Map(getReadable(notices).map((notice) => [notice.id, was.get(notice.id) ?? at]))',
                  'notices: readonly Notice[],\n  pos: number,\n): ReadonlyMap<string, number> =>\n  new Map(getReadable(notices).map((notice) => [notice.id, was.get(notice.id) ?? pos]))')
      .replaceAll('const isRead = (notice: Notice, firstSeen: ReadonlyMap<string, number>, at: number): boolean =>\n  at - (firstSeen.get(notice.id) ?? at) >= getNoticeDuration(notice.text, notice.about)',
                  'const isRead = (notice: Notice, firstSeen: ReadonlyMap<string, number>, pos: number): boolean =>\n  pos - (firstSeen.get(notice.id) ?? pos) >= getNoticeDuration(notice.text, notice.about)')
      .replaceAll('const isRead = (notice: Notice, firstSeen: ReadonlyMap<string, number>, pos: number): boolean =>\n  pos - (firstSeen.get(notice.id) ?? pos) >= getNoticeDuration(notice.text, notice.about)',
                  'const isRead = (notice: Notice, firstSeen: ReadonlyMap<string, number>, pos: number): boolean =>\n  pos - (firstSeen.get(notice.id) ?? pos) >= getNoticeDuration(notice.text, notice.about)')
      .replaceAll('notices: readonly Notice[],\n  at: number,\n  options: ShownNoticeOptions,',
                  'notices: readonly Notice[],\n  pos: number,\n  options: ShownNoticeOptions,')
      .replaceAll('!isRead(notice, options.firstSeen, at)', '!isRead(notice, options.firstSeen, pos)')
      .replaceAll('return notice.isUserStarted || at - (options.firstSeen.get(notice.id) ?? at) >= wait',
                  'return notice.isUserStarted || pos - (options.firstSeen.get(notice.id) ?? pos) >= wait')
      .replaceAll('notices: readonly Notice[],\n  firstSeen: ReadonlyMap<string, number>,\n  at: number,',
                  'notices: readonly Notice[],\n  firstSeen: ReadonlyMap<string, number>,\n  pos: number,')
      .replaceAll('isRead(notice, firstSeen, at)', 'isRead(notice, firstSeen, pos)')
    if (newContent !== before) {
      count += 5
      modified = true
    }
  }

  if (relPath.includes('features/palette/lib/place.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const optionId = (uid: string, at: number): string => `${uid}-option-${at}`',
                  'export const optionId = (uid: string, pos: number): string => `${uid}-option-${pos}`')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/palette/model/places.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly goTo: (at: number) => void', 'readonly goTo: (pos: number) => void')
      .replaceAll('readonly lightAt: (at: number, event: PointerEvent) => void', 'readonly lightAt: (pos: number, event: PointerEvent) => void')
      .replaceAll('readonly chooseAt: (at: number, second: boolean) => void', 'readonly chooseAt: (pos: number, second: boolean) => void')
      .replaceAll('const goTo = (at: number): void => {\n    target.value = places.value[at]?.item.id ?? \'\'\n  }',
                  'const goTo = (pos: number): void => {\n    target.value = places.value[pos]?.item.id ?? \'\'\n  }')
      .replaceAll('const lightAt = (at: number, event: PointerEvent): void => {\n    if (event.clientX === status.x && event.clientY === status.y) return\n    status = { x: event.clientX, y: event.clientY }\n    // An item the keyboard steps over is one the pointer passes over.\n    const item = places.value[at]?.item\n    if (item && canChoose(item)) goTo(at)\n  }',
                  'const lightAt = (pos: number, event: PointerEvent): void => {\n    if (event.clientX === status.x && event.clientY === status.y) return\n    status = { x: event.clientX, y: event.clientY }\n    // An item the keyboard steps over is one the pointer passes over.\n    const item = places.value[pos]?.item\n    if (item && canChoose(item)) goTo(pos)\n  }')
      .replaceAll('const chooseAt = (at: number, second: boolean): void => {\n    const item = places.value[at]?.item\n    const action = getActionId(item, second)\n    if (!item || !action) return\n    options.tellChoice(item.id, action)\n  }',
                  'const chooseAt = (pos: number, second: boolean): void => {\n    const item = places.value[pos]?.item\n    const action = getActionId(item, second)\n    if (!item || !action) return\n    options.tellChoice(item.id, action)\n  }')
    if (newContent !== before) {
      count += 6
      modified = true
    }
  }

  if (relPath.includes('features/palette/ui/Palette.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const passOver = async (item: Element | undefined, at: number) => {\n    item?.dispatchEvent(\n      new PointerEvent(\'pointermove\', { bubbles: true, clientX: at, clientY: at }),\n    )',
                  'const passOver = async (item: Element | undefined, pos: number) => {\n    item?.dispatchEvent(\n      new PointerEvent(\'pointermove\', { bubbles: true, clientX: pos, clientY: pos }),\n    )')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/palette/ui/Palette.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const onOver = (at: number, event: PointerEvent) => {\n  if (!isPanelOpen.value) places.lightAt(at, event)\n}',
                  'const onOver = (pos: number, event: PointerEvent) => {\n  if (!isPanelOpen.value) places.lightAt(pos, event)\n}')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/palette/ui/palette-actions/PaletteActions.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const actionName = (at: number): string => `${uid}-action-${at}`',
                  'const actionName = (pos: number): string => `${uid}-action-${pos}`')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/palette/ui/palette-results/PaletteResults.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll("(event: 'point-at', at: number, hasMoved: PointerEvent): void", "(event: 'point-at', pos: number, hasMoved: PointerEvent): void")
      .replaceAll("(event: 'choose', at: number, second: boolean): void", "(event: 'choose', pos: number, second: boolean): void")
      .replaceAll('@point-at="(at, moved) => emit(\'point-at\', at, moved)"', '@point-at="(pos, moved) => emit(\'point-at\', pos, moved)"')
      .replaceAll('@choose="(at, second) => emit(\'choose\', at, second)"', '@choose="(pos, second) => emit(\'choose\', pos, second)"')
    if (newContent !== before) {
      count += 4
      modified = true
    }
  }

  if (relPath.includes('features/palette/ui/palette-results/palette-group/PaletteGroup.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll("(event: 'point-at', at: number, hasMoved: PointerEvent): void", "(event: 'point-at', pos: number, hasMoved: PointerEvent): void")
      .replaceAll("(event: 'choose', at: number, second: boolean): void", "(event: 'choose', pos: number, second: boolean): void")
      .replaceAll('const rowId = (at: number): string => optionId(props.uid, at)', 'const rowId = (pos: number): string => optionId(props.uid, pos)')
    if (newContent !== before) {
      count += 3
      modified = true
    }
  }

  if (relPath.includes('features/plex/lib/edge.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const getHeading = (edge: EdgeCurve, at: number): EdgeHeading => {\n  const way = getTangent(edge, getParameter(measureAlong(edge), at))',
                  'export const getHeading = (edge: EdgeCurve, pos: number): EdgeHeading => {\n  const way = getTangent(edge, getParameter(measureAlong(edge), pos))')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/plex/ui/render/PlexView.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const reach = (view: ReturnType<typeof mountTitled>, at: number) =>\n    view.findAll(\'.plex__edge-hit\')[at]!',
                  'const reach = (view: ReturnType<typeof mountTitled>, pos: number) =>\n    view.findAll(\'.plex__edge-hit\')[pos]!')
      .replaceAll('const title = (view: ReturnType<typeof mountTitled>, at: number) =>\n    view.findAll(\'.plex__edge-label\').slice(2 * at, 2 * at + 2)',
                  'const title = (view: ReturnType<typeof mountTitled>, pos: number) =>\n    view.findAll(\'.plex__edge-label\').slice(2 * pos, 2 * pos + 2)')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('features/reader/lib/strip.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export function getTurnTarget(key: string, at: number, pageCount: number): number | undefined {\n  const way = getPageNavigationAction(key)\n  if (!way || pageCount === 0) return undefined\n\n  const last = pageCount - 1\n  let to: number\n  if (way === \'first\') to = 0\n  else if (way === \'last\') to = last\n  else to = at + (way === \'next\' ? 1 : -1)',
                  'export function getTurnTarget(key: string, pos: number, pageCount: number): number | undefined {\n  const way = getPageNavigationAction(key)\n  if (!way || pageCount === 0) return undefined\n\n  const last = pageCount - 1\n  let to: number\n  if (way === \'first\') to = 0\n  else if (way === \'last\') to = last\n  else to = pos + (way === \'next\' ? 1 : -1)')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/tree/lib/step.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('type Move = (visibleRows: readonly ShownRow[], at: number, here: ShownRow) => Step',
                  'type Move = (visibleRows: readonly ShownRow[], pos: number, here: ShownRow) => Step')
      .replaceAll('function stepDown(visibleRows: readonly ShownRow[], at: number, here: ShownRow): Step {\n  return { at: visibleRows[at + 1]?.id ?? here.id, turn: null }\n}',
                  'function stepDown(visibleRows: readonly ShownRow[], pos: number, here: ShownRow): Step {\n  return { at: visibleRows[pos + 1]?.id ?? here.id, turn: null }\n}')
      .replaceAll('function stepUp(visibleRows: readonly ShownRow[], at: number, here: ShownRow): Step {\n  return { at: visibleRows[at - 1]?.id ?? here.id, turn: null }\n}',
                  'function stepUp(visibleRows: readonly ShownRow[], pos: number, here: ShownRow): Step {\n  return { at: visibleRows[pos - 1]?.id ?? here.id, turn: null }\n}')
      .replaceAll('function stepIn(visibleRows: readonly ShownRow[], at: number, here: ShownRow): Step {\n  if (here.hasChildren && !here.isOpen) return { at: here.id, turn: { row: here.id, isOpen: true } }\n  if (here.isOpen && here.hasRows) return stepDown(visibleRows, at, here)',
                  'function stepIn(visibleRows: readonly ShownRow[], pos: number, here: ShownRow): Step {\n  if (here.hasChildren && !here.isOpen) return { at: here.id, turn: { row: here.id, isOpen: true } }\n  if (here.isOpen && here.hasRows) return stepDown(visibleRows, pos, here)')
      .replaceAll('function stepOut(_visibleRows: readonly ShownRow[], _at: number, here: ShownRow): Step {',
                  'function stepOut(_visibleRows: readonly ShownRow[], _pos: number, here: ShownRow): Step {')
    if (newContent !== before) {
      count += 5
      modified = true
    }
  }

  if (relPath.includes('features/welcome/lib/letters.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const vaultLetter = (at: number): string => VAULT_LETTERS[at] ?? \'\'',
                  'export const vaultLetter = (pos: number): string => VAULT_LETTERS[pos] ?? \'\'')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/welcome/ui/welcome-vaults/vault-list/VaultList.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const getVaultKeys = (at: number, vault: VaultRow): PaletteKeys | undefined => {\n  const letter = vaultLetter(at)',
                  'const getVaultKeys = (pos: number, vault: VaultRow): PaletteKeys | undefined => {\n  const letter = vaultLetter(pos)')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/workspace/ui/branch/WorkspaceBranch.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const layout = (one: BranchFixture, at: number, sizes: number[]) =>\n  groups(one)[at]!.vm.$emit(\'layout\', sizes)',
                  'const layout = (one: BranchFixture, pos: number, sizes: number[]) =>\n  groups(one)[pos]!.vm.$emit(\'layout\', sizes)')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/workspace/ui/pane/keys.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const stepTo = (key: string, at: number, count: number): number | null => {\n  if (at < 0 || at >= count) return null\n\n  switch (key) {\n    case \'ArrowRight\':\n      return (at + 1) % count\n    case \'ArrowLeft\':\n      return (at - 1 + count) % count',
                  'export const stepTo = (key: string, pos: number, count: number): number | null => {\n  if (pos < 0 || pos >= count) return null\n\n  switch (key) {\n    case \'ArrowRight\':\n      return (pos + 1) % count\n    case \'ArrowLeft\':\n      return (pos - 1 + count) % count')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('features/workspace/ui/pane/names.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const getTabName = (pane: NodeId, at: number): string => `${pane}-tab-${at}`',
                  'export const getTabName = (pane: NodeId, pos: number): string => `${pane}-tab-${pos}`')
      .replaceAll('export const getPanelName = (pane: NodeId, at: number): string => `${pane}-panel-${at}`',
                  'export const getPanelName = (pane: NodeId, pos: number): string => `${pane}-panel-${pos}`')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('features/workspace/ui/pane/body/PaneBody.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const tabName = (at: number): string => getTabName(props.pane.id, at)',
                  'const tabName = (pos: number): string => getTabName(props.pane.id, pos)')
      .replaceAll('const panelName = (at: number): string => getPanelName(props.pane.id, at)',
                  'const panelName = (pos: number): string => getPanelName(props.pane.id, pos)')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('features/book/model/layout.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const SPREAD_OF = (at: number) => at + 1000', 'const SPREAD_OF = (pos: number) => pos + 1000')
      .replaceAll('const request = async (at: number, standing = 0) => {', 'const request = async (pos: number, standing = 0) => {')
      .replaceAll('props.at = at', 'props.at = pos')
    if (newContent !== before) {
      count += 3
      modified = true
    }
  }

  if (relPath.includes('features/workspace/ui/pane/strip/PaneStrip.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const tabName = (at: number): string => getTabName(props.pane.id, at)',
                  'const tabName = (pos: number): string => getTabName(props.pane.id, pos)')
      .replaceAll('const panelName = (at: number): string => getPanelName(props.pane.id, at)',
                  'const panelName = (pos: number): string => getPanelName(props.pane.id, pos)')
      .replaceAll('function onStripKey(at: number, event: KeyboardEvent): void {',
                  'function onStripKey(pos: number, event: KeyboardEvent): void {')
      .replaceAll('const next = stepTo(event.key, at, props.pane.tabs.length)',
                  'const next = stepTo(event.key, pos, props.pane.tabs.length)')
    if (newContent !== before) {
      count += 4
      modified = true
    }
  }

  if (relPath.includes('shared/fixtures/ink.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const getPageY = (at: number): number => box.top + (at / GRAIN) * box.height',
                  'const getPageY = (pos: number): number => box.top + (pos / GRAIN) * box.height')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('shared/lib/address.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('export const findWikilink = (text: string, at: number): Wikilink | null =>\n  getWikilinks(text).find((one) => at >= one.at && at <= one.to) ?? null',
                  'export const findWikilink = (text: string, pos: number): Wikilink | null =>\n  getWikilinks(text).find((one) => pos >= one.at && pos <= one.to) ?? null')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('shared/ui/menu/keys.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('readonly goTo: (at: number) => void', 'readonly goTo: (pos: number) => void')
      .replaceAll('const goTo = (at: number): void => {\n    here.value = at\n    const item = items()[at]',
                  'const goTo = (pos: number): void => {\n    here.value = pos\n    const item = items()[pos]')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('shared/ui/select/Select.stories.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const getChoice = (line: string, at: number) => {\n  const [text = \'\', group] = line.split(\'\\t\')\n  const id = text.trim().toLowerCase().replace(/\\s+/g, \'-\') || `at-${at}`',
                  'const getChoice = (line: string, pos: number) => {\n  const [text = \'\', group] = line.split(\'\\t\')\n  const id = text.trim().toLowerCase().replace(/\\s+/g, \'-\') || `at-${pos}`')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('shared/ui/weekday-chips/WeekdayChips.test.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const getStyle = (at: number) => chips[at]?.attributes(\'style\') ?? \'\'',
                  'const getStyle = (pos: number) => chips[pos]?.attributes(\'style\') ?? \'\'')
      .replaceAll('const openMenu = async (row: ReturnType<typeof mountChips>, at: number) => {\n    const chip = row.findAll(\'button\')[at]',
                  'const openMenu = async (row: ReturnType<typeof mountChips>, pos: number) => {\n    const chip = row.findAll(\'button\')[pos]')
    if (newContent !== before) {
      count += 2
      modified = true
    }
  }

  if (relPath.includes('views.flashcards.studio/frontend/deck/lib/view.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('const getHeading = (at: number): string => readItem.cards[at]?.heading ?? \'\'\n  if (screen.cards.every((card, at) => card.heading === getHeading(at))) return screen\n  return {\n    ...screen,\n    cards: screen.cards.map((card, at) => ({ ...card, heading: getHeading(at) })),\n  }',
                  'const getHeading = (pos: number): string => readItem.cards[pos]?.heading ?? \'\'\n  if (screen.cards.every((card, index) => card.heading === getHeading(index))) return screen\n  return {\n    ...screen,\n    cards: screen.cards.map((card, index) => ({ ...card, heading: getHeading(index) })),\n  }')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('views.flashcards.studio/frontend/preset/ui/PresetTab.stories.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('moveSlider: (at: number) => {', 'moveSlider: (pos: number) => {')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('views.flashcards.studio/frontend/preset/ui/preset-goal/PresetGoal.vue')) {
    const before = newContent
    newContent = newContent
      .replaceAll('function onMoveSlider(at: number) {\n  props.state.moveSlider(at)\n}',
                  'function onMoveSlider(pos: number) {\n  props.state.moveSlider(pos)\n}')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  if (relPath.includes('views.flashcards.studio/frontend/preset/fixtures/index.ts')) {
    const before = newContent
    newContent = newContent
      .replaceAll('moveSlider: (at) => {', 'moveSlider: (pos) => {')
    if (newContent !== before) {
      count += 1
      modified = true
    }
  }

  return { newContent, modified, count }
}

function processFile(filePath) {
  const ext = path.extname(filePath)
  if (!EXTENSIONS.has(ext)) return

  const relPath = path.relative(targetDir, filePath)

  let content = ''
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
    return
  }

  const { newContent, modified, count } = transformFile(relPath, content)

  if (modified) {
    totalModifiedFiles++
    totalFixedIssues += count
    console.log(`[FIXED] ${relPath} (${count} issues)`)
    if (!isDryRun) {
      fs.writeFileSync(filePath, newContent, 'utf8')
    }
  }
}

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  for (const entry of entries) {
    if (IGNORED_DIRS.has(entry.name)) continue
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(fullPath)
    } else if (entry.isFile()) {
      processFile(fullPath)
    }
  }
}

const stats = fs.statSync(targetDir)
if (stats.isFile()) {
  processFile(targetDir)
} else {
  // If run at repo root, target modules/libs/ui and modules/plugins
  const uiDir = path.join(targetDir, 'modules/libs/ui')
  const pluginsDir = path.join(targetDir, 'modules/plugins')
  if (fs.existsSync(uiDir) && fs.existsSync(pluginsDir) && !targetDir.includes('modules')) {
    walk(uiDir)
    walk(pluginsDir)
  } else {
    walk(targetDir)
  }
}

console.log('======================================================================')
console.log(`Total modified files : ${totalModifiedFiles}`)
console.log(`Total fixed issues   : ${totalFixedIssues}`)
console.log(`Mode                 : ${isDryRun ? 'DRY-RUN' : 'APPLIED'}`)
console.log('======================================================================')
