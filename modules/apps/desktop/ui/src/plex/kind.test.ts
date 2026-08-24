/**
 * What a gesture in a plex comes to, asked without a screen.
 *
 * A note made from a node other than the focus is the one to watch: it is
 * written into the vault whatever happens, and a picture one seat deep draws
 * it only from the node it was made from.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { NeighbourhoodResponse } from '@numen/protocol'
import { plexKind, plexing, type Held, type Making, type Plexing } from './kind'
import type { Standing } from '../standing'
import type { Host } from '../windowing'

/** A neighbourhood as the vault answers one: a focus, and what is around it. */
const around = (focus: string, related: readonly string[] = []): NeighbourhoodResponse =>
  ({
    focus: { path: focus, title: focus.replace(/\.md$/, '') },
    related: related.map((path) => ({
      seat: 2,
      through: '',
      label: '',
      note: { path, title: path.replace(/\.md$/, '') },
    })),
  }) as unknown as NeighbourhoodResponse

/** A plex standing on a note, which records every note it was sent to. */
const standing = (at: string, related: readonly string[] = []) => {
  const went: string[] = []
  const view = {
    neighbourhood: ref(around(at, related)),
    here: ref(at),
    trouble: ref(''),
    go: async (path: string) => {
      went.push(path)
      view.here.value = path
      view.neighbourhood.value = around(path)
    },
    close: () => {},
  }
  return { view: view as unknown as Standing, went }
}

/** A vault that takes every note it is asked to make, and records the asking. */
const making = (takes = true) => {
  const made: [string, string][] = []
  const joined: [string, string, string][] = []
  const makes: Making = {
    make: async (from, seat) => {
      made.push([from, seat])
      return takes ? { path: 'Made.md', title: 'Made' } : null
    },
    join: async (from, to, seat) => {
      joined.push([from, to, seat])
      return takes
    },
  }
  return { makes, made, joined }
}

/** A plex tab with the window it is drawn in written down. */
const tab = (at: string, related: readonly string[] = [], takes = true) => {
  const plex = standing(at, related)
  const vault = making(takes)
  const opened: [string, string, string][] = []
  const asked: string[] = []
  const deps: Plexing = {
    makes: vault.makes,
    ready: () => true,
    opens: (path, title, showing) => opened.push([path, title, showing]),
    asks: (text) => asked.push(text),
    opening: () => 'Opening.md',
    first: async () => 'Opening.md',
  }
  return { held: plexing(plex.view, deps), went: plex.went, ...vault, opened, asked }
}

describe('a note made from a node', () => {
  it('stands the plex on the node it was made from', async () => {
    const one = tab('Root.md', ['Child.md'])

    await one.held.made('Child.md', 'child')

    expect(one.made).toEqual([['Child.md', 'child']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('leaves the plex where it is when that node is the focus', async () => {
    const one = tab('Root.md')

    await one.held.made('Root.md', 'child')

    expect(one.went).toEqual(['Root.md'])
    expect(one.held.view.here.value).toBe('Root.md')
  })

  it('travels nowhere when the vault made nothing', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.held.made('Child.md', 'parent')

    expect(one.went).toEqual([])
  })
})

describe('two notes a line was drawn between', () => {
  it('stands the plex on the note the line was drawn from', async () => {
    const one = tab('Root.md', ['Child.md', 'Other.md'])

    await one.held.joined('Child.md', 'Other.md', 'jump')

    expect(one.joined).toEqual([['Child.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('travels nowhere when nothing was written', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.held.joined('Child.md', 'Root.md', 'jump')

    expect(one.went).toEqual([])
  })
})

describe('the menu on a node', () => {
  const asked = (path: string) => ({
    path,
    at: { x: 1, y: 2 },
    from: null,
    opening: 'pointer' as const,
  })

  it('opens the note it was asked for on, called what the picture calls it', () => {
    const one = tab('Root.md', ['Child.md'])
    one.held.asks(asked('Child.md'))

    one.held.chose('open')

    expect(one.opened).toEqual([['Child.md', 'Child', 'here']])
    expect(one.held.menu.value).toBeNull()
  })

  it('makes a child of it, and the plex stands on it', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.held.asks(asked('Child.md'))

    one.held.chose('child')
    await Promise.resolve()

    expect(one.made).toEqual([['Child.md', 'child']])
  })

  it('puts a question about it to the agent', () => {
    const one = tab('Root.md', ['Child.md'])
    one.held.asks(asked('Child.md'))

    one.held.chose('ask')

    expect(one.asked).toEqual(['Child.md — '])
  })

  it('does nothing when it stands on nothing', () => {
    const one = tab('Root.md')

    one.held.chose('open')

    expect(one.opened).toEqual([])
  })

  it('goes when the picture under it does', () => {
    const one = tab('Root.md', ['Child.md'])
    one.held.asks(asked('Child.md'))

    one.held.dismiss()

    expect(one.held.menu.value).toBeNull()
  })
})

describe('what a note in the picture is called', () => {
  it('is the title the vault gave the focus', () => {
    expect(tab('Root.md').held.nameOf('Root.md')).toBe('Root')
  })

  it('is the title of a note around it', () => {
    expect(tab('Root.md', ['Deep/Child.md']).held.nameOf('Deep/Child.md')).toBe('Deep/Child')
  })

  it('is the file it is filed under, for a note the picture does not name', () => {
    expect(tab('Root.md').held.nameOf('Deep/Elsewhere.md')).toBe('Elsewhere')
  })
})

describe('the picture', () => {
  it('is nothing while the window has nothing true to draw', () => {
    const plex = standing('Root.md')
    const held = plexing(plex.view, {
      makes: making().makes,
      ready: () => false,
      opens: () => {},
      asks: () => {},
      opening: () => '',
      first: async () => '',
    })

    expect(held.picture.value).toBeNull()
  })
})

/**
 * The plexes of a window, each standing where it was told to.
 *
 * A note is asked for somewhere the window cannot see — the palette, an agent
 * working the vault beside the person — and one of these has to take it.
 */
const window = (opening = 'Opening.md') => {
  const views: ReturnType<typeof standing>[] = []
  const opened: string[] = []
  const shown: string[] = []
  /** Every time the vault was asked where it opens, and what it answered then. */
  const asked: string[] = []
  let first = opening
  const host: Host = {
    opens: async (kind, at = '') => {
      opened.push(`${kind} ${at}`)
      return ''
    },
    beside: async () => '',
    shows: (id) => shown.push(id),
    closes: () => {},
  }
  const makes = () => {
    const view = standing('')
    views.push(view)
    return view.view
  }
  const plexes = plexKind(host, makes, {
    makes: making().makes,
    ready: () => true,
    opens: () => {},
    asks: () => {},
    opening: () => first,
    first: async () => {
      asked.push(first)
      return first
    },
  })
  /** A tab of this window under an identity of its own, and what it holds. */
  const holds = (id: string, at = '') => plexes.kind.opens(at, id) as Held
  /** The vault gained a note, which is what it opens with from now on. */
  const gains = (path: string) => {
    first = path
  }
  return { ...plexes, holds, gains, views, opened, shown, asked }
}

describe('a plex tab as it opens', () => {
  it('stands on the note the vault opens with', () => {
    const one = window('Opening.md')

    const held = one.holds('plex:one')

    expect(held.view.here.value).toBe('Opening.md')
  })

  it('stands where it was told to, whatever the vault opens with', () => {
    const one = window('Opening.md')

    const held = one.holds('plex:one', 'Told.md')

    expect(held.view.here.value).toBe('Told.md')
  })

  it('stands where the person is looking, when another one is open', () => {
    const one = window('Opening.md')
    const first = one.holds('plex:one')
    void first.view.go('Here.md')

    const second = one.holds('plex:two')

    expect(second.view.here.value).toBe('Here.md')
  })

  it('stands nowhere while the vault opens with nothing', () => {
    const one = window('')

    expect(one.holds('plex:one').view.here.value).toBe('')
  })
})

describe('the plex the person is looking at', () => {
  it('is the one they were last in, and the window says its trouble', () => {
    const one = window()
    const first = one.holds('plex:one', 'One.md')
    const second = one.holds('plex:two', 'Two.md')
    first.view.trouble.value = 'One.md is not in the vault'

    one.kind.shown?.(first, 'plex:one')
    expect(one.looking()).toBe('One.md')
    expect(one.trouble()).toBe('One.md is not in the vault')

    one.kind.shown?.(second, 'plex:two')
    expect(one.looking()).toBe('Two.md')
    expect(one.trouble()).toBe('')
  })

  it('is the one before it when the tab in front closes', () => {
    const one = window()
    const first = one.holds('plex:one', 'One.md')
    const second = one.holds('plex:two', 'Two.md')
    const third = one.holds('plex:three', 'Three.md')
    one.kind.shown?.(second, 'plex:two')
    one.kind.shown?.(third, 'plex:three')

    one.kind.shuts?.(third, 'plex:three')

    expect(one.looking()).toBe('Two.md')
    expect(first.view.here.value).toBe('One.md')
  })

  it('is nothing at all in a window holding no plex', () => {
    const one = window()

    expect(one.looking()).toBe('')
    expect(one.trouble()).toBe('')
  })
})

describe('a note put in front of the person', () => {
  it('is where the plex they are looking at travels', async () => {
    const one = window()
    const first = one.holds('plex:one', 'One.md')
    const second = one.holds('plex:two', 'Two.md')
    one.kind.shown?.(second, 'plex:two')

    await one.travel('Wanted.md')

    expect(second.view.here.value).toBe('Wanted.md')
    expect(first.view.here.value).toBe('One.md')
  })

  it('opens a plex of its own in a window holding none', async () => {
    const one = window()

    await one.travel('Wanted.md')

    expect(one.opened).toStrictEqual(['plex Wanted.md'])
  })
})

describe('every plex asked for its picture again', () => {
  it('asks for the note it is standing on, each of its own', async () => {
    const one = window()
    const first = one.holds('plex:one', 'One.md')
    const second = one.holds('plex:two', 'Two.md')
    first.view.here.value = 'One.md'

    await one.again()

    expect(one.views[0]?.went).toContain('One.md')
    expect(one.views[1]?.went).toContain('Two.md')
    expect(second.view.here.value).toBe('Two.md')
  })

  it('gives one standing nowhere the note an empty vault has just gained', async () => {
    const one = window('')
    const held = one.holds('plex:one')
    one.gains('First.md')

    await one.again()

    expect(held.view.here.value).toBe('First.md')
  })

  it('asks the vault where it opens once, however many stand nowhere', async () => {
    const one = window('')
    one.holds('plex:one')
    one.holds('plex:two')
    one.holds('plex:three')

    await one.again()

    expect(one.asked).toHaveLength(1)
  })

  it('asks it not at all while every one of them is standing somewhere', async () => {
    const one = window()
    one.holds('plex:one', 'One.md')
    one.holds('plex:two', 'Two.md')

    await one.again()

    expect(one.asked).toStrictEqual([])
  })

  it('asks nothing for a plex whose tab has closed', async () => {
    const one = window()
    const first = one.holds('plex:one', 'One.md')
    const second = one.holds('plex:two', 'Two.md')
    one.kind.shuts?.(second, 'plex:two')

    await one.again()

    expect(one.views[0]?.went).toContain('One.md')
    expect(one.views[1]?.went.filter((where) => where === 'Two.md')).toHaveLength(1)
    expect(first.view.here.value).toBe('One.md')
  })
})
