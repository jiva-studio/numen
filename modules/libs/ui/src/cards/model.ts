/**
 * What a stencil and a deck are, as plain values. No DOM, no measurement, no
 * clock.
 *
 * A stencil declares named slots and the faces that show them; a deck is cards
 * standing what a person typed in those slots. What either of them means is
 * the caller's.
 */

import { previewed, strayIn } from './fill'

/** One named slot and what stands in it. */
export interface Filled {
  readonly field: string
  readonly text: string
}

/** One card as the deck draws it. */
export interface Drawn {
  readonly id: string
  readonly name: string
  /** What the card is cut by, as a word to show. Empty where it is cut by nothing. */
  readonly stencil: string
  readonly filled: readonly Filled[]
}

/** One way a stencil shows a card. */
export interface Shown {
  readonly id: string
  readonly name: string
  readonly front: string
  readonly back: string
}

/** One stencil a card may be cut by: the word it is shown as, and its slots. */
export interface Cut {
  readonly name: string
  /** The slots it names, in the order a person is asked for them. */
  readonly fields: readonly string[]
}

/** Where a carried entry lands: before the entry named, or at the end. */
export type Landing = string | null

/**
 * The order a carried entry lands in. The entry is taken out first, so landing
 * before itself, before nothing, or before a name that is not there leaves the
 * order as it was.
 */
export function ordered(
  names: readonly string[],
  carried: string,
  at: Landing,
): readonly string[] {
  if (!names.includes(carried)) return names

  const left = names.filter((name) => name !== carried)
  if (at === null) return [...left, carried]

  const before = left.indexOf(at)
  if (before === -1) return names
  return [...left.slice(0, before), carried, ...left.slice(before)]
}

/**
 * A carried field may land where it was let go. The first field names every
 * card the stencil cuts, so it stays first: it does not move, and nothing lands
 * above it. A field let go where it stands moves nothing either.
 */
export function landing(
  fields: readonly string[],
  carried: string,
  at: Landing,
): boolean {
  const first = fields[0]
  if (first === undefined) return false
  if (carried === first || at === first) return false
  return carried !== at
}

/** Which way along the order something is carried by the keyboard. */
export type Way = 'up' | 'down'

/** The way along the order an arrow carries what is held, and nothing for any other key. */
export const wayOf = (key: string): Way | null =>
  key === 'ArrowUp' ? 'up' : key === 'ArrowDown' ? 'down' : null

/**
 * Where a carried entry lands one place along the order, and nothing where
 * there is no place that way. Landing before the entry past the next one is
 * what puts it one place further down, the entry being taken out first.
 */
export function stepped(
  names: readonly string[],
  carried: string,
  way: Way,
): Landing | undefined {
  const at = names.indexOf(carried)
  if (at === -1) return undefined
  if (way === 'up') return at === 0 ? undefined : (names[at - 1] ?? undefined)
  if (at === names.length - 1) return undefined
  return names[at + 2] ?? null
}

/** The order a carried field lands in, with the first field left where it is. */
export const reordered = (
  fields: readonly string[],
  carried: string,
  at: Landing,
): readonly string[] => (landing(fields, carried, at) ? ordered(fields, carried, at) : fields)

/** Why a name cannot be used, and nothing where it can. */
export type Objection = 'blank' | 'taken' | 'braced'

/**
 * What is wrong with a name. A name is what a slot is written by, so a name
 * carrying a brace cannot be written, and one already taken names two slots.
 */
export function objection(name: string, taken: readonly string[]): Objection | null {
  const said = name.trim()
  if (said === '') return 'blank'
  if (said.includes('{') || said.includes('}')) return 'braced'
  if (taken.some((each) => each.trim() === said)) return 'taken'
  return null
}

/**
 * One line of what was typed. The field a card is named by is written in a
 * heading, so the breaks in it close up.
 */
export const oneLine = (text: string): string => text.replace(/\r\n|[\n\r]/g, ' ')

/** The words a stencil is drawn with, declared once. */
export interface StencilWords {
  readonly fields: string
  readonly faces: string
  readonly front: string
  readonly back: string
  readonly preview: string
  readonly addField: string
  readonly addFace: string
  readonly remove: string
  readonly carry: string
  readonly insert: string
  readonly noFields: string
  readonly noFaces: string
  readonly fieldStem: string
  readonly faceStem: string
  /** What is said of the first field, which is not moved and not removed. */
  readonly pinned: string
  /** What is said of the slots a face names that the fields do not. */
  readonly stray: (fields: readonly string[]) => string
  /** What is said of a field's name that cannot be used. */
  readonly objection: (why: Objection) => string
  /** What is said of a face's name that cannot be used. */
  readonly faceObjection: (why: Objection) => string
}

export const STENCIL_WORDS: StencilWords = {
  fields: 'Fields',
  faces: 'Faces',
  front: 'Front',
  back: 'Back',
  preview: 'Preview',
  addField: 'Add a field',
  addFace: 'Add a face',
  remove: 'Remove',
  carry: 'Reorder',
  insert: 'Insert',
  noFields: 'No fields yet',
  noFaces: 'No faces yet',
  fieldStem: 'Field',
  faceStem: 'Face',
  pinned: 'The first field names every card, and stays first',
  stray: (fields) => `Not a field: ${fields.join(', ')}`,
  objection: (why) =>
    why === 'blank'
      ? 'A field needs a name'
      : why === 'taken'
        ? 'That name is taken'
        : 'A name cannot hold a brace',
  faceObjection: (why) =>
    why === 'blank'
      ? 'A face needs a name'
      : why === 'taken'
        ? 'That name is taken'
        : 'A name cannot hold a brace',
}

/** The words a card is drawn with, declared once. */
export interface FaceWords {
  /** What is said in place of a half with nothing in it. */
  readonly silence: string
  /** What the button turning the card says. */
  readonly turning: string
}

export const FACE_WORDS: FaceWords = {
  silence: 'Nothing here',
  turning: 'Turn',
}

/** The words a deck is drawn with, declared once. */
export interface DeckWords {
  readonly add: string
  readonly remove: string
  readonly carry: string
  readonly cut: string
  readonly name: string
  /** What is said of a card holding no value at all. */
  readonly nothing: string
  /** What is said of a card whose stencil the vault does not hold. */
  readonly unknown: (stencil: string) => string
  /** What is said of a value standing in the field the card is named by. */
  readonly twice: string
  readonly cardStem: string
}

export const DECK_WORDS: DeckWords = {
  add: 'Add a card',
  remove: 'Remove',
  carry: 'Reorder',
  cut: 'Stencil',
  name: 'Name',
  nothing: 'Nothing in it',
  unknown: (stencil) => (stencil ? `No stencil called ${stencil}` : 'Cut by no stencil'),
  twice: 'The card is named by this field',
  cardStem: 'Card',
}

/** A name that is free, made by numbering from a stem: `Field 1`, `Field 2`, … */
export function freeName(taken: readonly string[], stem: string): string {
  const held = new Set(taken.map((name) => name.trim()))
  for (let at = 1; ; at += 1) {
    const tried = `${stem} ${at}`
    if (!held.has(tried)) return tried
  }
}

/** A name being typed over the one a field carries. */
export interface Draft {
  /** The field it is being typed over. */
  readonly over: string
  readonly text: string
}

/** One field of a stencil, as its row is drawn. */
export interface FieldRow {
  readonly field: string
  /** Where it stands, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many fields stand with it. */
  readonly of: number
  /** What is in its box: the name it carries, or what is being typed over it. */
  readonly text: string
  /** Why what is in its box cannot be used, and nothing while it can. */
  readonly objection: Objection | null
  /** It stands first, so it is what a card cut by this stencil is named by. */
  readonly names: boolean
  /** It is on its way somewhere else in the order. */
  readonly carried: boolean
}

/**
 * A name is compared as written and the first of two stands, so a name a
 * stencil declares twice is one field.
 */
export const declared = (fields: readonly string[]): readonly string[] => [...new Set(fields)]

/**
 * The rows a stencil's fields are drawn as, one to a field. A name is measured
 * against every other field's, so a field keeping its own name objects to
 * nothing.
 */
export function fieldRows(
  fields: readonly string[],
  draft: Draft | null,
  carried: string | null,
): readonly FieldRow[] {
  const stood = declared(fields)
  return stood.map((field, index) => {
    const typed = draft?.over === field ? draft.text : null
    return {
      field,
      at: index + 1,
      of: stood.length,
      text: typed ?? field,
      objection:
        typed === null ? null : objection(typed, stood.filter((each) => each !== field)),
      names: index === 0,
      carried: field === carried,
    }
  })
}

/** One face of a stencil, as its block is drawn. */
export interface FaceBlock {
  readonly id: string
  readonly name: string
  /** What is in its name box: the name it carries, or what is being typed over it. */
  readonly text: string
  /** Why what is in its name box cannot be used, and nothing while it can. */
  readonly objection: Objection | null
  /** Where it stands, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many faces stand with it. */
  readonly of: number
  /** What is written in the two boxes. */
  readonly front: string
  readonly back: string
  /** The same two, with the sample values standing in the braces. */
  readonly frontShown: string
  readonly backShown: string
  /** The slots each half names that the fields do not, each said once. */
  readonly frontStray: readonly string[]
  readonly backStray: readonly string[]
}

/**
 * The blocks a stencil's faces are drawn as, each carrying what its preview
 * shows and what is wrong in each half of it. A name is measured against every
 * other face's, so a face keeping its own name objects to nothing. What is
 * typed over a name is drawn in its box, and the name it carries is what the
 * face is announced by until the typing is committed.
 */
export function faceBlocks(
  faces: readonly Shown[],
  fields: readonly string[],
  sample: readonly Filled[],
  draft: Draft | null = null,
): readonly FaceBlock[] {
  return faces.map((face, index) => {
    const typed = draft?.over === face.id ? draft.text : null
    return {
      id: face.id,
      name: face.name,
      text: typed ?? face.name,
      objection:
        typed === null
          ? null
          : objection(typed, faces.filter((each) => each.id !== face.id).map((each) => each.name)),
      at: index + 1,
      of: faces.length,
      front: face.front,
      back: face.back,
      frontShown: previewed(face.front, sample, fields),
      backShown: previewed(face.back, sample, fields),
      frontStray: strayIn(face.front, fields),
      backStray: strayIn(face.back, fields),
    }
  })
}

/** The box a face's fields are written into. */
export interface Aim {
  readonly face: string
  readonly half: Half
}

/**
 * The half of a face a field is written into: the one last typed in, and the
 * front while nothing has been.
 */
export const aimedAt = (aim: Aim | null, face: string): Half =>
  aim?.face === face ? aim.half : 'front'

/** One value of a card, laid out under the stencil that cuts it. */
export interface Laid extends Filled {
  /** The stencil names this slot. */
  readonly declared: boolean
}

/**
 * A card's values in the order the stencil asks for them, empty where the card
 * leaves a slot out, and a slot the card writes twice standing twice. A slot
 * the stencil names twice is one slot and stands once. The values the stencil
 * names nothing for come after the rest, marked as named by nothing, and what
 * is drawn of them is the caller's.
 */
export function laid(filled: readonly Filled[], fields: readonly string[]): readonly Laid[] {
  const stood = declared(fields).flatMap((field) => {
    const written = filled.filter((each) => each.field === field)
    if (!written.length) return [{ field, text: '', declared: true }]
    return written.map((each) => ({ field, text: each.text, declared: true }))
  })
  const stray = filled
    .filter((each) => !fields.includes(each.field))
    .map((each) => ({ field: each.field, text: each.text, declared: false }))
  return [...stood, ...stray]
}

/** The empty values a stencil's slots make, for a card nobody has typed into. */
export const blanks = (fields: readonly string[]): readonly Filled[] =>
  fields.map((field) => ({ field, text: '' }))

/** One value of a card as its tile draws it. */
export interface Stood extends Laid {
  /** Where it stands among the values, counting from one. */
  readonly at: number
  /**
   * What tells it from every other value of its tile. Values standing under
   * one field are told apart by their order under it.
   */
  readonly key: string
  /** Where it stands among the values written under its own field, from one. */
  readonly nth: number
  /** It is the field the card is named by, which is the stencil's first. */
  readonly names: boolean
  /**
   * It names the card and stands as a value as well. A card is named once, so
   * the value is kept and shown, and no face lays it out.
   */
  readonly twice: boolean
}

/** One card as a tile of the grid. */
export interface Tile {
  readonly id: string
  readonly name: string
  readonly stencil: string
  /** Its values, the field naming the card first, laid out under its stencil. */
  readonly filled: readonly Stood[]
  /** Where it stands among the tiles, counting from one, which is what it is announced as. */
  readonly at: number
  /** The stencil it names is among the ones handed in. */
  readonly known: boolean
  /** A field of its stencil names it. Where none does, the tile says the name it was handed. */
  readonly named: boolean
  /** It is on its way somewhere else in the order. */
  readonly carried: boolean
}

/** The grid a deck draws: the cards as tiles, and where the plus stands. */
export interface Grid {
  readonly tiles: readonly Tile[]
  /** Where the plus stands, counting from one. It stands last. */
  readonly plusAt: number
  /** How many stand in the grid, the plus among them. */
  readonly of: number
}

/**
 * The tiles of a deck, in the order the cards were handed in, each laid out
 * under the stencil it names. A card naming a stencil that was not handed in
 * keeps every value it has.
 *
 * A stencil's first field names the card. Its value stands in the name the card
 * was handed and in none of its values, and it is put back at the head of them
 * so the tile draws it as it draws every other field.
 */
export function grid(
  cards: readonly Drawn[],
  cuts: readonly Cut[],
  carried: string | null,
): Grid {
  const tiles = cards.map((card, index) => {
    const cut = cuts.find((each) => each.name === card.stencil)
    const fields = declared(cut?.fields ?? [])
    const first = fields[0]

    const rest = laid(card.filled, fields.slice(1)).map((each) => ({
      ...each,
      names: false,
      twice: first !== undefined && each.field === first,
    }))
    const named = first === undefined ? [] : [
      { field: first, text: card.name, declared: true, names: true, twice: false },
    ]

    const under = new Map<string, number>()
    const told = (field: string): number => {
      const nth = (under.get(field) ?? 0) + 1
      under.set(field, nth)
      return nth
    }

    return {
      id: card.id,
      name: card.name,
      stencil: card.stencil,
      // A value the stencil does not name is the person's and stays in the
      // file, and nothing here draws it or says a word about it. A card whose
      // stencil the vault does not hold draws no value at all, and says which
      // stencil it is waiting for.
      filled: [...named, ...rest]
        .filter((each) => each.declared || each.twice)
        .map((each, place) => {
          const nth = told(each.field)
          return { ...each, at: place + 1, nth, key: `${each.field}#${nth}` }
        }),
      at: index + 1,
      known: cut !== undefined,
      named: first !== undefined,
      carried: card.id === carried,
    }
  })
  return { tiles, plusAt: cards.length + 1, of: cards.length + 1 }
}

/** Which half of a face is drawn. */
export type Half = 'front' | 'back'

/** Both halves, in the order they are drawn. */
export const HALVES: readonly Half[] = ['front', 'back']

/** What one part of the window a face is edited in holds. */
export type Shows = 'written' | 'preview'

/** Both parts of a half, in the order they are drawn. */
export const SHOWS: readonly Shows[] = ['written', 'preview']

/** One part of the window a face is edited in. */
export interface Pane {
  readonly half: Half
  readonly shows: Shows
  /** What the part is called while nothing stands in it. */
  readonly said: string
  /** What the part is announced as. */
  readonly named: string
  /** The markup as it is written, or the sample standing in its braces. */
  readonly text: string
  /** Nothing but space stands in it. */
  readonly blank: boolean
  /** The slots the half names that the fields do not, said under the markup. */
  readonly stray: readonly string[]
}

/**
 * The four parts of one face, in the order they are drawn: each half's markup
 * and then what that markup comes to. Two parts to a row stand the front above
 * the back; one to a row stands each preview under the half it is of.
 */
export function panes(block: FaceBlock, words: StencilWords = STENCIL_WORDS): readonly Pane[] {
  return HALVES.flatMap((half): readonly Pane[] => {
    const said = half === 'front' ? words.front : words.back
    const written = half === 'front' ? block.front : block.back
    const shown = half === 'front' ? block.frontShown : block.backShown
    return [
      {
        half,
        shows: 'written',
        said,
        named: said,
        text: written,
        blank: written.trim() === '',
        stray: half === 'front' ? block.frontStray : block.backStray,
      },
      {
        half,
        shows: 'preview',
        said: words.preview,
        named: `${words.preview}: ${block.name} ${said}`,
        text: shown,
        blank: shown.trim() === '',
        stray: [],
      },
    ]
  })
}

/** One half of a card as it is drawn. */
export interface Part {
  readonly half: Half
  /** Markdown, assembled already. */
  readonly text: string
  /** Nothing but space stands in it. */
  readonly blank: boolean
}

/**
 * What a face draws, in the order it draws it. The back stands once the card
 * is turned and not before.
 */
export function parts(front: string, back: string, turned: boolean): readonly Part[] {
  const part = (half: Half, text: string): Part => ({
    half,
    text,
    blank: text.trim() === '',
  })
  return turned ? [part('front', front), part('back', back)] : [part('front', front)]
}
