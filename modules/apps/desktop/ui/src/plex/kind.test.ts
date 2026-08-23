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
import { plexing, type Making, type Plexing } from './kind'
import type { Plexed } from '../showing'

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
    looking: () => {},
    close: () => {},
  }
  return { view: view as unknown as Plexed, went }
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
    })

    expect(held.picture.value).toBeNull()
  })
})
