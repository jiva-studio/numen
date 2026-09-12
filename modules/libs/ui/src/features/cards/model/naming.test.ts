/**
 * A name typed over the one something carries: what stands in the box while it
 * is being typed, what is wrong with it, and what a commit comes to.
 *
 * A field is named by its own name and written in a slot; a face is named by
 * its own identifier and written in a heading. Both are here, because the two
 * are not written under the same rules.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { useNaming, type NamingState } from './naming'
import {
  checkFieldName,
  checkHeadingName,
  type Objection,
  type HeadingObjection,
} from '../lib/order'

/** A naming of the fields a stencil declares, with what it renamed. */
const overFields = (fields: readonly string[]) => {
  const renamed: (readonly [string, string])[] = []
  const naming = useNaming<Objection>({
    getName: (field) => field,
    getTakenNames: (field) => fields.filter((each) => each !== field),
    checkName: checkFieldName,
    rename: (field, name) => {
      renamed.push([field, name])
    },
  })
  return { naming, renamed }
}

/** A naming of the faces a stencil shows, each under the name it carries. */
const overFaces = (faces: ReadonlyMap<string, string>) => {
  const renamed: (readonly [string, string])[] = []
  const naming = useNaming<HeadingObjection>({
    getName: (id) => faces.get(id) ?? '',
    getTakenNames: (id) => [...faces].filter(([each]) => each !== id).map(([, name]) => name),
    checkName: checkHeadingName,
    rename: (id, name) => {
      renamed.push([id, name])
    },
  })
  return { naming, renamed }
}

/** A key struck in the box a name is typed in, which is what a break blurs. */
const press = <Why extends Objection>(
  naming: NamingState<Why>,
  over: string,
  key: string,
): HTMLInputElement => {
  const box = document.createElement('input')
  document.body.append(box)
  box.addEventListener('keydown', (event) => naming.onKey(event, over))
  box.focus()
  box.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
  return box
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('useNaming', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('stands the name a thing carries while nothing is being typed over it', () => {
    const { naming } = overFields(FIELDS)
    expect(naming.text('Height')).toBe('Height')
    expect(naming.objection('Height')).toBeNull()
  })

  it('stands what is typed in the box it is typed in, and in no other', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tall')
    expect(naming.text('Height')).toBe('Tall')
    expect(naming.text('Weight')).toBe('Weight')
    // It stands there until it is committed, and nothing is renamed meanwhile.
    expect(renamed).toEqual([])
  })

  it('renames the thing under the name typed, once it is committed', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    naming.commit('Height')
    expect(renamed).toEqual([['Height', 'Tallness']])
    expect(naming.text('Height')).toBe('Height')
  })

  it('drops the space around a name it commits', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', '  Tallness  ')
    naming.commit('Height')
    expect(renamed).toEqual([['Height', 'Tallness']])
  })

  it('objects to nothing where a thing is typed the name it already carries', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Height')
    expect(naming.objection('Height')).toBeNull()
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('objects to a name another one carries, and renames nothing', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Weight')
    expect(naming.objection('Height')).toBe('taken')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('objects to a name with nothing in it, and renames nothing', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', '   ')
    expect(naming.objection('Height')).toBe('blank')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('objects to a brace in a field’s name, which no slot could write', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'How {{tall}}')
    expect(naming.objection('Height')).toBe('braced')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('takes a brace in a face’s name, which stands in a heading', () => {
    const { naming, renamed } = overFaces(new Map([['one', 'One']]))
    naming.setDraft('one', 'What {{Name}} is')
    expect(naming.objection('one')).toBeNull()
    naming.commit('one')
    expect(renamed).toEqual([['one', 'What {{Name}} is']])
  })

  it('measures a face’s name against the names the other faces carry', () => {
    const { naming, renamed } = overFaces(
      new Map([
        ['one', 'One'],
        ['two', 'Two'],
      ]),
    )
    naming.setDraft('one', 'Two')
    expect(naming.objection('one')).toBe('taken')
    naming.commit('one')
    expect(renamed).toEqual([])
  })

  /* The commit clears what was typed, and a box nothing is being typed in
     objects to nothing. */
  it('consults on a commit the objection that was drawn', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Weight')
    const drawn = naming.objection('Height')
    naming.commit('Height')
    expect(drawn).toBe('taken')
    expect(naming.objection('Height')).toBeNull()
    expect(renamed).toEqual([])
  })

  it('commits a name on a break, and gives up the box it was typed in', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    const box = press(naming, 'Height', 'Enter')
    expect(renamed).toEqual([['Height', 'Tallness']])
    expect(document.activeElement).not.toBe(box)
  })

  it('abandons what is typed on escape, and renames nothing', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    press(naming, 'Height', 'Escape')
    expect(naming.text('Height')).toBe('Height')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('leaves what is typed standing under any other key', () => {
    const { naming } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    press(naming, 'Height', 'a')
    expect(naming.text('Height')).toBe('Tallness')
  })
})
