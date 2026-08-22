/**
 * What each tab holds, and what it lets go of when it closes.
 *
 * A tab that closes without letting go of what it held is a window that looks
 * right: the plex is off the screen and still being asked for, and the talk is
 * gone and still listening.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { paneWithTab, type Turn } from '@numen/ui'
import { holding, type Makes } from './holding'
import type { Conversation } from './conversation'
import type { Reading } from './reading'
import type { Plexed } from './showing'

/** A plex that stands where it was told to and records what became of it. */
const plexed = (closed: string[], looked: string[], at: string) => {
  const view = {
    neighbourhood: ref(null),
    here: ref(at),
    trouble: ref(''),
    go: async (path: string) => {
      view.here.value = path
    },
    looking: () => looked.push(view.here.value),
    close: () => closed.push(view.here.value),
  }
  return view as unknown as Plexed
}

/**
 * A talk that records having been let go of and having been told it is over,
 * under the name it answers by. A talk that is over is let go of as well.
 */
const talked = (stopped: string[], over: string[], conversation: string): Conversation => ({
  turns: ref<Turn[]>([]),
  working: ref(false),
  ask: async () => {},
  place: () => null,
  places: () => [],
  stop: () => stopped.push(conversation),
  finish: () => {
    stopped.push(conversation)
    over.push(conversation)
  },
})

/** A document that records having been let go of, under the path it is read from. */
const opened = (dropped: string[], path: string) => {
  const view = {
    path,
    pages: ref(0),
    at: ref(0),
    picture: ref(''),
    close: () => dropped.push(path),
  }
  return view as unknown as Reading
}

/** A window whose ports record what they were asked for. */
function window(made: { note?: string } = {}) {
  const closed: string[] = []
  const looked: string[] = []
  const stopped: string[] = []
  const over: string[] = []
  const conversations: string[] = []
  const dropped: string[] = []
  const read: string[] = []
  const makes: Makes = {
    plex: (at = '') => plexed(closed, looked, at),
    talk: (conversation) => {
      conversations.push(conversation)
      return talked(stopped, over, conversation)
    },
    note: async () => made.note ?? '',
    document: (path) => {
      read.push(path)
      return opened(dropped, path)
    },
  }
  return { held: holding(makes), closed, looked, stopped, over, conversations, dropped, read }
}

describe('the window as it opens', () => {
  it('opens with one plex and one agent, both in the layout', () => {
    const { held } = window()

    const [plex] = [...held.plexes.value.keys()]
    const [agent] = [...held.agents.value.keys()]
    expect(paneWithTab(held.layout.value.root, plex ?? '')).not.toBeNull()
    expect(paneWithTab(held.layout.value.root, agent ?? '')).not.toBeNull()
  })

  /**
   * A nameless question is no conversation at all, and a name the window has
   * given out before is somebody else's conversation resumed.
   */
  it('names every conversation, apart from the tab it is talked in', () => {
    const { held, conversations } = window()
    held.agentTab()

    const tabs = [...held.agents.value.keys()]
    expect(new Set(conversations).size).toBe(2)
    for (const name of conversations) {
      expect(name).not.toBe('')
      expect(tabs).not.toContain(name)
    }
  })
})

describe('the tab now on screen', () => {
  it('is the plex the person is in, and the agent a question goes to', () => {
    const { held, looked } = window()
    const plex = held.plexTab('Note.md')
    const agent = held.agentTab()

    held.shown(plex)
    held.shown(agent)

    expect(looked).toStrictEqual(['Note.md'])
    expect(held.talking.value).toBe(agent)
  })
})

describe('a plex tab that closes', () => {
  it('lets go of the plex, which is what stops it being asked for', () => {
    const { held, closed } = window()
    const id = held.plexTab('Note.md')

    expect(held.shut(id)).toBe(true)

    expect(closed).toContain('Note.md')
    expect(held.plexes.value.has(id)).toBe(false)
  })
})

describe('an agent tab that closes', () => {
  it('lets go of the talk, which is what stops it listening', () => {
    const { held, stopped, conversations } = window()
    const id = held.agentTab()

    expect(held.shut(id)).toBe(true)

    expect(stopped).toContain(conversations.at(-1))
    expect(held.agents.value.has(id)).toBe(false)
  })

  it('says the conversation is over, which is what the agent lets go of', () => {
    const { held, over, conversations } = window()
    const id = held.agentTab()

    held.shut(id)

    expect(over).toStrictEqual([conversations.at(-1)])
  })

  it('is no longer where a question about a note is put', () => {
    const { held } = window()
    const id = held.agentTab()
    held.shown(id)
    expect(held.talking.value).toBe(id)

    held.shut(id)
    held.askAbout('Note.md — ')

    expect(held.agents.value.has(id)).toBe(false)
    expect(held.agents.value.size).toBe(2)
  })
})

describe('a document opened', () => {
  it('is read in a tab of its own, under the path it is filed at', () => {
    const { held, read } = window()

    held.reads('Books/Manual.pdf')

    expect(read).toStrictEqual(['Books/Manual.pdf'])
    expect(held.documents.value.has('Books/Manual.pdf')).toBe(true)
    expect(paneWithTab(held.layout.value.root, 'Books/Manual.pdf')).not.toBeNull()
  })

  it('is the one tab it already has when it is opened again', () => {
    const { held, read } = window()
    held.reads('Books/Manual.pdf')

    held.reads('Books/Manual.pdf')

    expect(read).toStrictEqual(['Books/Manual.pdf'])
    expect(held.documents.value.size).toBe(1)
  })
})

describe('a document tab that closes', () => {
  it('lets go of the document, which is what lets go of what it drew', () => {
    const { held, dropped } = window()
    const id = held.documentTab('Books/Manual.pdf')

    expect(held.shut(id)).toBe(true)

    expect(dropped).toStrictEqual(['Books/Manual.pdf'])
    expect(held.documents.value.has(id)).toBe(false)
  })
})

describe('a blank tab that closes', () => {
  it('is no longer waiting to be told what it holds', () => {
    const { held } = window()
    held.blanked(held.layout.value.focus)
    const [id] = held.blanks.value

    expect(held.shut(id ?? '')).toBe(true)
    expect(held.blanks.value).toStrictEqual([])
  })
})

describe('a tab the window holds nothing for', () => {
  it("says so, since a note tab is the window's to shut", () => {
    const { held } = window()

    expect(held.shut('Note.md')).toBe(false)
  })
})

describe('the window going', () => {
  it('lets go of every plex, every talk and every document', () => {
    const { held, closed, stopped, dropped } = window()
    held.plexTab('One.md')
    held.agentTab()
    held.documentTab('Manual.pdf')

    held.close()

    expect(closed).toContain('One.md')
    expect(stopped).toHaveLength(2)
    expect(dropped).toStrictEqual(['Manual.pdf'])
  })

  it('says every conversation it had open is over', () => {
    const { held, over, conversations } = window()
    held.agentTab()

    held.close()

    expect(over).toStrictEqual(conversations)
    expect(over).toHaveLength(2)
  })
})

describe('a blank tab told what to be', () => {
  it('holds a plex where it stood, and is gone', async () => {
    const { held } = window()
    held.blanked(held.layout.value.focus)
    const [blank = ''] = held.blanks.value
    const pane = paneWithTab(held.layout.value.root, blank)?.id

    await held.becomeIt(blank, 'plex')

    expect(held.blanks.value).toStrictEqual([])
    expect(paneWithTab(held.layout.value.root, blank)).toBeNull()
    const opened = [...held.plexes.value.keys()].at(-1) ?? ''
    expect(paneWithTab(held.layout.value.root, opened)?.id).toBe(pane)
  })

  it('holds an agent where it stood', async () => {
    const { held } = window()
    held.blanked(held.layout.value.focus)
    const [blank = ''] = held.blanks.value

    await held.becomeIt(blank, 'agent')

    expect(held.agents.value.size).toBe(2)
    expect(held.blanks.value).toStrictEqual([])
  })

  it('holds the note that was made, in the tab the note is filed under', async () => {
    const { held } = window({ note: 'Untitled note.md' })
    held.blanked(held.layout.value.focus)
    const [blank = ''] = held.blanks.value

    await held.becomeIt(blank, 'note')

    expect(paneWithTab(held.layout.value.root, 'Untitled note.md')).not.toBeNull()
    expect(held.blanks.value).toStrictEqual([])
  })

  it('stands where it was when no note was made', async () => {
    const { held } = window({ note: '' })
    held.blanked(held.layout.value.focus)
    const [blank = ''] = held.blanks.value

    await held.becomeIt(blank, 'note')

    expect(held.blanks.value).toStrictEqual([blank])
    expect(paneWithTab(held.layout.value.root, blank)).not.toBeNull()
  })
})

describe('a question about a note', () => {
  it('is written in the agent tab the person was last in', () => {
    const { held } = window()
    const one = [...held.agents.value.keys()][0] ?? ''
    const two = held.agentTab()
    held.shown(two)

    held.askAbout('Note.md — ')

    expect(held.agents.value.get(two)?.asked.value).toBe('Note.md — ')
    expect(held.agents.value.get(one)?.asked.value).toBe('')
  })

  it('opens an agent to carry it when the window has none', () => {
    const { held } = window()
    for (const id of [...held.agents.value.keys()]) held.shut(id)

    held.askAbout('Note.md — ')

    expect(held.agents.value.size).toBe(1)
    const [talk] = [...held.agents.value.values()]
    expect(talk?.asked.value).toBe('Note.md — ')
  })
})

describe('a plex opened for a note asked for from outside', () => {
  it('stands on that note, in a tab of its own', () => {
    const { held } = window()

    held.shows('Wanted.md')

    const opened = [...held.plexes.value.values()].at(-1)
    expect(opened?.view.here.value).toBe('Wanted.md')
    expect(paneWithTab(held.layout.value.root, [...held.plexes.value.keys()].at(-1) ?? '')).not.toBeNull()
  })
})
