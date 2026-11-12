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

/** A URL, an unbreakable word that users paste. */
export const LINK =
  'https://example.invalid/a/very/long/path/that/keeps/going/and/going?with=query&and=more#and-a-fragment'

export const LONG =
  'The seed format is not settled, so a component built against today’s guess ' +
  'at it would be rewritten when the guess is replaced. That is why a component ' +
  'here takes props it defines itself and emits events carrying opaque ' +
  'identifiers: whoever renders it translates the domain into that shape, and ' +
  'translates the identifier back. The component cannot answer a question about ' +
  'the graph, and it does not need to. When what a link is finally gets decided, ' +
  'the adapter in the application changes and nothing here does.'

export const EMPTY = ''

/** The choices of a control written one a line, and the one chosen, for the controls that offer a few. */
export interface ChoiceWords {
  readonly words: string
  readonly chosen: string
}

/** Choices far longer than the box drawing them. */
export const LONG_CHOICES: ChoiceWords = {
  words: 'As small as it will go\nSomewhere between the two of them\nAs large as the room allows',
  chosen: 'somewhere-between-the-two-of-them',
}

/** Choices in another script. */
export const CYRILLIC_CHOICES: ChoiceWords = {
  words: 'Маленький\nСредний\nБольшой',
  chosen: 'средний',
}

/** A choice with nothing in it to break at, beside a short one. */
export const UNBROKEN_CHOICES: ChoiceWords = {
  words: 'supercalifragilisticexpialidocious\nShort',
  chosen: 'short',
}
