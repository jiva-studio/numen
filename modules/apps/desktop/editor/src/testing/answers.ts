/**
 * What the mocked application answers, and what a test sets before the window
 * draws.
 *
 * `said` is what the vault says about itself and what stands in it, `listed`
 * the vaults this installation holds, and `folders` what each folder holds.
 */
import type { Entry } from '@/entities/file'

/** What the mocked vault answers about itself, set before the window draws. */
export const said = {
  ready: true,
  error: '',
  opening: 'Root.md' as string | null,
  names: [] as {
    path: string
    title: string
    heading: string
    line: number
    at: []
    type: 'note' | 'deck' | 'stencil'
  }[],
  /** The passages the search answers with. */
  passages: [] as {
    path: string
    title: string
    isNote: boolean
    text: string
    start: number
    length: number
    line: number
    at: []
    type: 'note' | 'deck' | 'stencil'
    kind: 'note' | 'book' | 'recording' | 'other'
  }[],
  /**
   * Which of three the note at each path is, as the vault answers it. A path
   * it says nothing about is the ordinary note the window reads it as.
   */
  types: {} as Record<string, 'note' | 'deck' | 'stencil'>,
  /** How many of the spans the index holds carry a vector. */
  embedded: 0,
  /** Whether the list of vaults answers at all. */
  listable: true,
  /** What stopped a choice being written, which is a size outside its bounds. */
  writeError: '',
  /** What the settings say the window is drawn as, which a test may set. */
  applied: 'preset:numen',
  mode: 'system' as 'system' | 'light' | 'dark',
  sizes: { interfaceScale: 1, textScale: 1 },
  /** The recording the vault answers with, and the words written down in it. */
  transcribed: {
    duration: 60_000,
    mediaUrl: 'numen://recording/talk.mp3',
    mediaType: 'audio/mpeg',
    cues: [{ text: 'the first thing said', from: 0, to: 4000 }] as {
      text: string
      from: number
      to: number
    }[],
    /** False while a run writing the transcript holds it. */
    isEditable: true,
  },
  /**
   * What the file at each path carries, as the application answers it. A path
   * it says nothing about carries nothing.
   */
  carries: {} as Record<string, Record<string, string>>,
  /** Whether the application answers what a file carries at all. */
  carrying: true,
  /** What renaming a field of a stencil comes back with. */
  renaming: {
    decks: [] as string[],
    cards: 0,
    notWritten: [] as { path: string; text: string }[],
    error: null as import('@/shared/errors').ErrorCode | null,
    changed: false,
    at: 'a2',
  },
}

/** The vaults this installation holds, and the one the window is showing. */
export const listed = {
  vaults: [{ id: 'physics', name: 'Physics', path: '/vaults/Physics', missing: false }],
  showing: 'physics',
}

/** What each folder of the vault holds, as a listing answers it. */
export const folders: Record<string, readonly Entry[]> = {
  '': [
    { path: 'physics', name: 'physics', isFolder: true, kind: 'other', type: 'note' },
    { path: 'Root.md', name: 'Root.md', isFolder: false, kind: 'note', type: 'note' },
    { path: 'Cover.png', name: 'Cover.png', isFolder: false, kind: 'other', type: 'note' },
  ],
  physics: [
    {
      path: 'physics/Entropy.md',
      name: 'Entropy.md',
      isFolder: false,
      kind: 'note',
      type: 'note',
    },
    { path: 'physics/Kelvin.md', name: 'Kelvin.md', isFolder: false, kind: 'note', type: 'note' },
  ],
}

/**
 * What the vault holds at a path. A book and a note are told apart by the name
 * the file carries, the way the vault itself tells them apart, and which of
 * three a note is is what a test said.
 */
export const getFileKind = (path: string) => {
  if (/\.epub$/u.test(path)) {
    return { kind: 'book' as const, type: 'note' as const, format: 'epub' as const }
  }
  if (/\.pdf$/u.test(path)) {
    return { kind: 'book' as const, type: 'note' as const, format: 'pdf' as const }
  }
  if (/\.(mp3|m4a|wav)$/u.test(path)) return { kind: 'recording' as const, type: 'note' as const }
  if (!/\.(md|note)$/u.test(path)) return { kind: 'other' as const, type: 'note' as const }
  return { kind: 'note' as const, type: said.types[path] ?? ('note' as const) }
}

/** A stream that stays open, so nothing the window follows ever ends. */
export async function* held(): AsyncGenerator<never> {
  yield await new Promise<never>(() => {})
}

/**
 * The places something outside the window asks to be put in front of the
 * person, and the stream the window hears them on.
 */
export const outside = (() => {
  const queue: { path: string; spans: { from: number; to: number }[] }[] = []
  let wake: (() => void) | null = null
  return {
    ask: (at: { path: string; start: number; length: number }) => {
      queue.push({ path: at.path, spans: [{ from: at.start, to: at.start + at.length }] })
      wake?.()
      wake = null
    },
    forget: () => queue.splice(0),
    stream: async function* () {
      for (;;) {
        while (queue.length > 0) yield queue.shift()!
        await new Promise<void>((woken) => {
          wake = woken
        })
      }
    },
  }
})()

/**
 * The vault as it makes a deck or a stencil: the file is named after the title,
 * and the extension is the vault's own and no caller's. It is not markdown
 * here, so a window building the path for itself reaches nothing.
 */
export const maker = (() => {
  const filed = new Set<string>()
  /** Whether the vault answers what it is asked at all. */
  let reached = true
  return {
    forget: () => {
      filed.clear()
      reached = true
    },
    /** The vault is out of reach, so asking it for one reaches nothing. */
    fail: () => {
      reached = false
    },
    createFile: (title: string, folder: string) => {
      if (!reached) throw new Error('the vault could not be reached')
      const path = `${folder ? `${folder}/` : ''}${title}.note`
      if (filed.has(path)) return { ok: false as const, error: 'occupied' as const }
      filed.add(path)
      return { ok: true as const, value: { path } }
    },
  }
})()

/** One name the vault answers a search with, of a note of some kind. */
export const nameAnswer = (
  path: string,
  title: string,
  type: 'note' | 'deck' | 'stencil' = 'note',
) => ({
  path,
  title,
  heading: '',
  line: -1,
  at: [] as [],
  type,
})

/** One passage the search answers with, read out of a note of some kind. */
export const passageAnswer = (
  path: string,
  title: string,
  type: 'note' | 'deck' | 'stencil' = 'note',
) => ({
  path,
  title,
  isNote: true,
  text: 'what it says',
  start: 0,
  length: 4,
  line: 3,
  at: [] as [],
  type,
  kind: 'note' as 'note' | 'book' | 'recording' | 'other',
})

/** One passage read out of a source that is not a note. */
export const sourceAnswer = (path: string, kind: 'book' | 'recording') => ({
  ...passageAnswer(path, ''),
  isNote: false,
  kind,
})

/** Everything answered back where it opens, which is where every test begins. */
export const forgetAnswers = () => {
  said.ready = true
  said.opening = 'Root.md'
  said.names = []
  said.passages = []
  said.types = {}
  said.embedded = 0
  said.listable = true
  said.writeError = ''
  said.applied = 'preset:numen'
  said.mode = 'system'
  said.sizes = { interfaceScale: 1, textScale: 1 }
  said.transcribed = {
    duration: 60_000,
    mediaUrl: 'numen://recording/talk.mp3',
    mediaType: 'audio/mpeg',
    cues: [{ text: 'the first thing said', from: 0, to: 4000 }],
    isEditable: true,
  }
  said.carries = {}
  said.carrying = true
  said.renaming = {
    decks: [],
    cards: 0,
    notWritten: [],
    error: null,
    changed: false,
    at: 'a2',
  }
  maker.forget()
  outside.forget()
  listed.vaults = [{ id: 'physics', name: 'Physics', path: '/vaults/Physics', missing: false }]
  listed.showing = 'physics'
}
