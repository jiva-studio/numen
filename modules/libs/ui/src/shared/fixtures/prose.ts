/**
 * Text that is awkward on purpose.
 *
 * Each breaks a different assumption: a script that is not Latin, a script
 * that runs the other way, a word with nowhere to break, a paragraph past any
 * width, a character that is several code units, and nothing at all.
 */

export const DEVANAGARI =
  'बगीचे की बाड़ के पास एक पुराना लकड़ी का शेड है, जिसमें बीज और औज़ार रखे रहते हैं। ' +
  'हर वसंत में सब लोग वहाँ मिलते हैं और नई क्यारियाँ बाँटते हैं।'

export const ARABIC = 'الكتابة من اليمين إلى اليسار داخل فقاعة، لنرى أين ينكسر السطر.'

export const RUSSIAN =
  'В сарае на краю участка лежат лопаты, лейки и мешок с семенами. Ключ от ' +
  'двери висит на гвозде.'

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
