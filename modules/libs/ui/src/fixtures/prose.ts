/**
 * Text that is awkward on purpose.
 *
 * Each breaks a different assumption: a script that is not Latin, a script
 * that runs the other way, a word with nowhere to break, a paragraph past any
 * width, a character that is several code units, and nothing at all.
 */

export const DEVANAGARI =
  'सर्वस्य चाहं हृदि सन्निविष्टो मत्तः स्मृतिर्ज्ञानमपोहनं च। ' +
  'वेदैश्च सर्वैरहमेव वेद्यो वेदान्तकृद्वेदविदेव चाहम्॥'

export const ARABIC = 'الكتابة من اليمين إلى اليسار داخل فقاعة، لنرى أين ينكسر السطر.'

export const RUSSIAN =
  'Заметка — это обычный файл в обычной папке. Не база данных и не экспорт: ' +
  'файлы и есть заметки.'

/** No spaces, no hyphens, nothing to break at. */
export const UNBREAKABLE =
  'Donaudampfschiffahrtselektrizitaetenhauptbetriebswerkbauunterbeamtengesellschaft' +
  'Rindfleischetikettierungsueberwachungsaufgabenuebertragungsgesetz'

/** A URL, which is the unbreakable word anyone actually pastes. */
export const LINK =
  'https://example.invalid/a/very/long/path/that/keeps/going/and/going?with=query&and=more#and-a-fragment'

export const LONG =
  'The vault format is not settled, so a component built against today’s guess ' +
  'at it would be rewritten when the guess is replaced. That is why a component ' +
  'here takes props it defines itself and emits events carrying opaque ' +
  'identifiers: whoever renders it translates the domain into that shape, and ' +
  'translates the identifier back. The component cannot answer a question about ' +
  'the graph, and it does not need to. When what a link is finally gets decided, ' +
  'the adapter in the application changes and nothing here does.'

export const MULTILINE =
  'Three things follow:\n\n' +
  '  1. a decision cannot break a component\n' +
  '  2. a component cannot answer a question about the graph\n' +
  '  3. the dependency runs one way\n\n' +
  'and they are the point.'

/** One character each to a reader; several code units each to a machine. */
export const GRAPHEMES = '👩‍👩‍👧‍👦 👋🏽 🇮🇳 é'

export const EMPTY = ''
