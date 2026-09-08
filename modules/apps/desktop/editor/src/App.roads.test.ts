/**
 * Every road to a file, and what the window tells whoever answers for the
 * person about what they have open.
 *
 * A road holds a path and no choice, so the same file reached by the palette,
 * by the search, by a link in an answer or from outside the window opens in the
 * same editor every time.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { type VueWrapper } from '@vue/test-utils'
import { Agent, Editor, Palette, Plex, Reader, Tree, WorkspaceLayout, type Workspace } from '@numen/ui'
import DocumentTab from './features/document/DocumentTab.vue'
import NoteTab from './features/note/NoteTab.vue'
import RecordingTab from './features/media/recording-tab/RecordingTab.vue'
import {
  asked,
  cards,
  DEBOUNCE,
  drawn,
  folders,
  nameSaid,
  nodeInPlex,
  outside,
  paneKinds,
  passageSaid,
  said,
  settles,
} from './shared/testing/window'

describe('the window with no note to show', () => {
  it('says the vault could not be read, and that nothing was read from it', async () => {
    said.opening = null
    said.failed = 'the vault folder is not there'

    const window = await drawn()

    const corner = cards(window)
    expect(corner[0]).toContain('the vault folder is not there')
    expect(corner[1]).toBe('nothing was read')
  })

  it('says nothing when the vault was read and holds none', async () => {
    said.opening = null
    said.failed = ''

    const window = await drawn()

    expect(cards(window)).toStrictEqual([])
  })
})

describe('every road to a file', () => {
  /**
   * Each road, by what it is called, as a gesture on a window standing on one
   * file. They are listed here so that every one of them is asked the same
   * four questions, and a road opening a file some other way is a road missing
   * from this list.
   */
  const ROADS: Record<string, (window: VueWrapper, path: string) => Promise<void>> = {
    'the plex': async (window) => {
      window.findComponent(Plex).vm.$emit('show', nodeInPlex(window), 'here')
    },
    'the tree': async (window, path) => {
      window.findComponent(Tree).vm.$emit('activate', path)
    },
    'the palette': async (window, path) => {
      await typedIn(window, 'ani')
      window.findComponent(Palette).vm.$emit('choose', path, 'note')
    },
    'the search': async (window, path) => {
      await typedIn(window, 'ani')
      window.findComponent(Palette).vm.$emit('choose', `text:${path}:0`, 'note')
    },
    'a command': async (window) => {
      pressing('p')
      await settles()
      window.findComponent(Palette).vm.$emit('choose', 'read', 'read')
    },
    // The two roads that arrive naming a stretch of a source's own text: a link
    // in an answer the agent wrote, and a place asked for from outside the
    // window altogether.
    'a link in an answer': async (window, path) => {
      const turn = { id: 'a', voice: 'answered' as const, text: '' }
      const link = `numen:${encodeURIComponent(path)}?start=0&length=4`
      window
        .findComponent(Agent)
        .vm.$emit('follow', turn, link, new MouseEvent('click', { cancelable: true }))
    },
    'a place asked for from outside': async (_, path) => {
      outside.asks({ path, start: 0, length: 4 })
    },
  }

  /** A keystroke the window answers, which the palette and the commands are. */
  const pressing = (key: string) =>
    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true }),
    )

  /** The search open, with words typed into it and the answers back. */
  const typedIn = async (window: VueWrapper, typed: string) => {
    pressing('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', typed)
    await new Promise((done) => setTimeout(done, DEBOUNCE))
  }

  /** The root of the vault holds one file of each of four. */
  const root = folders['']
  beforeEach(() => {
    folders[''] = [
      { path: 'Animals.md', name: 'Animals.md', folder: false, kind: 'note', type: 'deck' },
      { path: 'Animal.md', name: 'Animal.md', folder: false, kind: 'note', type: 'stencil' },
      { path: 'Ants.md', name: 'Ants.md', folder: false, kind: 'note', type: 'note' },
      { path: 'Ants.epub', name: 'Ants.epub', folder: false, kind: 'book', type: 'note' },
    ]
  })
  afterEach(() => {
    folders[''] = root ?? []
  })

  /** What the window drew, having been taken to one file by one road. */
  const taken = async (
    road: (window: VueWrapper, path: string) => Promise<void>,
    path: string,
    type: 'note' | 'deck' | 'stencil',
  ): Promise<readonly string[]> => {
    said.types = { [path]: type }
    said.opening = path
    said.names = [nameSaid(path, path, type)]
    said.passages = [passageSaid(path, path, type)]

    const window = await drawn()
    await road(window, path)
    await settles()
    await settles()
    return paneKinds(window).flat()
  }

  for (const [name, road] of Object.entries(ROADS)) {
    it(`opens a deck in the editor of its cards, reached by ${name}`, async () => {
      expect(await taken(road, 'Animals.md', 'deck')).toContain('deck')
    })

    it(`opens a stencil in the editor of its fields and faces, reached by ${name}`, async () => {
      expect(await taken(road, 'Animal.md', 'stencil')).toContain('stencil')
    })

    it(`opens an ordinary note in the editor of its prose, reached by ${name}`, async () => {
      const drew = await taken(road, 'Ants.md', 'note')

      expect(drew).toContain('note')
      expect(drew).not.toContain('deck')
      expect(drew).not.toContain('stencil')
    })

    // The vault is never asked about the book by anything but the window, and
    // the palette is told it turned up a note: the path alone has to be enough.
    it(`opens a book in the reader, reached by ${name}`, async () => {
      const drew = await taken(road, 'Ants.epub', 'note')

      expect(drew).toContain('document')
      expect(drew).not.toContain('note')
    })
  }
})

describe('what the person has open, as whoever answers for them is told it', () => {
  /** The last the window said about it, and nothing where it has said nothing. */
  const reported = () => asked.attending.at(-1) ?? null

  /** The tab the window said is in front, of the last it said. */
  const front = () => {
    const open = reported()
    return open?.tabs.find((one) => one.id === open.front) ?? null
  }

  it('lists every tab the window holds, the plex it opens on in front', async () => {
    await drawn()

    expect([...(reported()?.tabs ?? [])].map((one) => one.kind).sort()).toEqual([
      'agent',
      'files',
      'plex',
    ])
    expect(front()?.kind).toBe('plex')
    expect(front()?.path).toBe('Root.md')
  })

  it('names the tab beside the agent, where the person is writing in one', async () => {
    const window = await drawn()
    const workspace = window.findComponent(WorkspaceLayout)

    // The person is in the agent, which is where a question is written.
    workspace.vm.$emit('update:modelValue', {
      ...(workspace.props('modelValue') as Workspace),
      focus: 'aside',
    })
    await settles()

    expect(front()?.kind).toBe('plex')
  })

  it('names the document in front, and the note the plex stands on beside it', async () => {
    const window = await drawn()

    outside.asks({ path: 'Ants.epub', start: 0, length: 4 })
    await settles()
    await settles()
    window.unmount()

    expect(front()?.kind).toBe('document')
    expect(front()?.path).toBe('Ants.epub')
    expect(reported()?.tabs.some((one) => one.kind === 'plex' && one.path === 'Root.md')).toBe(true)
  })
})

describe('a note asked for in the plex', () => {
  it('is drawn in a tab of its own', async () => {
    const window = await drawn()

    window.findComponent(Plex).vm.$emit('show', nodeInPlex(window), 'here')
    await settles()

    expect(window.findComponent(NoteTab).exists()).toBe(true)
    expect(window.findComponent(NoteTab).findComponent(Editor).exists()).toBe(true)
  })

  it('is asked for by the node, and a path opens nothing', async () => {
    const window = await drawn()

    window.findComponent(Plex).vm.$emit('show', 'Root.md', 'here')
    await settles()

    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })
})

describe('a place an answer names', () => {
  it('is drawn in the document it stands in', async () => {
    const window = await drawn()

    window
      .findComponent(Agent)
      .vm.$emit(
        'follow',
        { id: 'said', voice: 'said', text: '[here](numen:Source.pdf?start=0&length=4)' },
        'numen:Source.pdf?start=0&length=4',
        { preventDefault: () => {} },
      )
    await settles()

    expect(window.findComponent(DocumentTab).exists()).toBe(true)
    expect(window.findComponent(DocumentTab).findComponent(Reader).exists()).toBe(true)
  })
})

// A recording is the one file the window reaches two ports for: the player is
// loaded from one, and what the file carries is asked of the other. Neither is
// reached by any other kind of tab.
describe('a recording put in front', () => {
  const RECORDING = '730709BG.LON.mp3'

  /** The window with that recording open, asked for from outside it. */
  const playing = async () => {
    const window = await drawn()
    outside.asks({ path: RECORDING, start: 0, length: 4 })
    await settles()
    await settles()
    return window
  }

  it('is played from where the application answers, on the words written down', async () => {
    const window = await playing()

    const state = window.findComponent(RecordingTab).props('state') as {
      address: { value: string }
      prose: { value: string }
      editable: { value: boolean }
    }
    expect(asked.listened).toStrictEqual([RECORDING])
    expect(state.address.value).toBe(said.transcribed.mediaUrl)
    expect(state.prose.value).toBe(said.transcribed.cues[0]?.text)
    expect(state.editable.value).toBe(true)
    // How long it runs and how far the words reach are the application's
    // answer, and what the window says the person has open carries them.
    expect(asked.attending.at(-1)?.tabs.at(-1)).toMatchObject({
      path: RECORDING,
      recording: {
        transcribedDurationMs: said.transcribed.cues[0]?.to,
        durationMs: said.transcribed.duration,
      },
    })
  })

  // What is offered over a recording follows what has been made from it, and
  // not its kind alone: one already written down is not offered to be written
  // down again.
  it('is offered being written down while it carries no transcript', async () => {
    said.carries = { [RECORDING]: { transcript: 'none' } }

    const window = await playing()

    expect(asked.carried).toStrictEqual([RECORDING])
    expect(await runsOffered(window)).toStrictEqual(['transcribe'])
  })

  it('is offered putting the words right once a run has written them', async () => {
    said.carries = { [RECORDING]: { transcript: 'done' } }

    const window = await playing()

    expect(asked.carried).toStrictEqual([RECORDING])
    expect(await runsOffered(window)).toStrictEqual(['proofread', 'deleteText'])
  })
})

/** The runs the palette offers over the file in front, in the order it draws them. */
const runsOffered = async (window: VueWrapper): Promise<readonly string[]> => {
  globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, cancelable: true }))
  await settles()
  const groups = window.findComponent(Palette).props('groups') as readonly {
    id: string
    items: readonly { id: string }[]
  }[]
  return groups.find((one) => one.id === 'file')?.items.map((one) => one.id) ?? []
}

