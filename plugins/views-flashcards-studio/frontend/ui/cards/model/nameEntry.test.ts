import { afterEach, describe, expect, it } from 'vitest'
import { useNameEntry, type NameEntryState } from './nameEntry'
import {
  checkFieldName,
  checkHeadingName,
  type Objection,
  type HeadingObjection,
} from '../lib/order'

const overFields = (fields: readonly string[]) => {
  const renamedPairs: (readonly [string, string])[] = []
  const naming = useNameEntry<Objection>({
    getName: (field) => field,
    getTakenNames: (field) => fields.filter((each) => each !== field),
    checkName: checkFieldName,
    rename: (field, name) => {
      renamedPairs.push([field, name])
    },
  })
  return { naming, renamed: renamedPairs }
}

const overTemplates = (templates: ReadonlyMap<string, string>) => {
  const renamedPairs: (readonly [string, string])[] = []
  const naming = useNameEntry<HeadingObjection>({
    getName: (id) => templates.get(id) ?? '',
    getTakenNames: (id) => [...templates].filter(([each]) => each !== id).map(([, name]) => name),
    checkName: checkHeadingName,
    rename: (id, name) => {
      renamedPairs.push([id, name])
    },
  })
  return { naming, renamed: renamedPairs }
}

const press = <Why extends Objection>(
  naming: NameEntryState<Why>,
  targetId: string,
  key: string,
): { box: HTMLInputElement; event: KeyboardEvent } => {
  const box = document.createElement('input')
  document.body.append(box)
  box.addEventListener('keydown', (event) => naming.onKey(event, targetId))
  box.focus()
  const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true })
  box.dispatchEvent(event)
  return { box, event }
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('useNameEntry', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('remains the name a thing carries while nothing is being typed over it', () => {
    const { naming } = overFields(FIELDS)
    expect(naming.getText('Height')).toBe('Height')
    expect(naming.getObjection('Height')).toBeNull()
  })

  it('remains what is typed in the box it is typed in, and in no other', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tall')
    expect(naming.getText('Height')).toBe('Tall')
    expect(naming.getText('Weight')).toBe('Weight')
    expect(renamed).toEqual([])
  })

  it('renames the thing under the name typed, once it is committed', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    naming.commit('Height')
    expect(renamed).toEqual([['Height', 'Tallness']])
    expect(naming.getText('Height')).toBe('Height')
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
    expect(naming.getObjection('Height')).toBeNull()
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('objects to a name another one carries, and does not rename', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Weight')
    expect(naming.getObjection('Height')).toBe('taken')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('objects to a name with nothing in it, and does not rename', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', '   ')
    expect(naming.getObjection('Height')).toBe('blank')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('objects to a brace in a field’s name, which no slot could write', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'How {{tall}}')
    expect(naming.getObjection('Height')).toBe('braced')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('takes a brace in a template’s name, which remains in a heading', () => {
    const { naming, renamed } = overTemplates(new Map([['one', 'One']]))
    naming.setDraft('one', 'What {{Name}} is')
    expect(naming.getObjection('one')).toBeNull()
    naming.commit('one')
    expect(renamed).toEqual([['one', 'What {{Name}} is']])
  })

  it('measures a template’s name against the names the other templates carry', () => {
    const { naming, renamed } = overTemplates(
      new Map([
        ['one', 'One'],
        ['two', 'Two'],
      ]),
    )
    naming.setDraft('one', 'Two')
    expect(naming.getObjection('one')).toBe('taken')
    naming.commit('one')
    expect(renamed).toEqual([])
  })

  it('consults on a commit the objection that was drawn', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Weight')
    const objection = naming.getObjection('Height')
    naming.commit('Height')
    expect(objection).toBe('taken')
    expect(naming.getObjection('Height')).toBeNull()
    expect(renamed).toEqual([])
  })

  it('commits a name on a break, and gives up the box it was typed in', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    const { box, event } = press(naming, 'Height', 'Enter')
    expect(renamed).toEqual([['Height', 'Tallness']])
    expect(document.activeElement).not.toBe(box)
    expect(event.defaultPrevented).toBe(true)
  })

  it('abandons what is typed on escape, and does not rename', () => {
    const { naming, renamed } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    const { event } = press(naming, 'Height', 'Escape')
    expect(event.defaultPrevented).toBe(true)
    expect(naming.getText('Height')).toBe('Height')
    naming.commit('Height')
    expect(renamed).toEqual([])
  })

  it('leaves what is typed standing under any other key', () => {
    const { naming } = overFields(FIELDS)
    naming.setDraft('Height', 'Tallness')
    const { event } = press(naming, 'Height', 'a')
    expect(event.defaultPrevented).toBe(false)
    expect(naming.getText('Height')).toBe('Tallness')
  })
})
