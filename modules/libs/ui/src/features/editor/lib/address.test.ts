import { afterEach, describe, expect, it, vi } from 'vitest'
import { StateEffect } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { getAddress, follow } from './address'
import { addressOpener } from './outside'
import { createState } from '../fixtures/state'

describe('the address under a position in the text', () => {
  const getAddressIn = (doc: string, text: string) =>
    getAddress(createState(doc), doc.indexOf(text) + 1)

  it('is what a link points at', () => {
    expect(getAddressIn('go [there](https://example.invalid)', 'there')).toBe(
      'https://example.invalid',
    )
  })

  it('is what a note in brackets points at', () => {
    expect(getAddressIn('under [[Thermodynamics]] it sits', 'Thermo')).toBe('name://Thermodynamics')
  })

  it('is the identifier where the brackets hold one', () => {
    expect(getAddressIn('under [[note://01J8]] it sits', 'note://')).toBe('note://01J8')
  })

  it('is nothing where the position remains in no link', () => {
    expect(getAddressIn('under nothing at all', 'nothing')).toBeNull()
  })

  it('is nothing inside code, where a link is an example of one', () => {
    expect(getAddressIn('```\n[[Thermodynamics]]\n```', 'Thermo')).toBeNull()
    expect(getAddressIn('write `[[Thermodynamics]]` for it', 'Thermo')).toBeNull()
  })
})

describe('the address under a position on a later line', () => {
  it('is what a note in brackets points at', () => {
    const doc = 'first line\nunder [[Thermodynamics]] it sits'
    expect(getAddress(createState(doc), doc.indexOf('Thermo') + 1)).toBe('name://Thermodynamics')
  })

  it('is what an address in angle brackets is', () => {
    const doc = 'go <https://example.invalid> now'
    expect(getAddress(createState(doc), doc.indexOf('example') + 1)).toBe('https://example.invalid')
  })

  it('is nothing inside an indented block of code', () => {
    const doc = '    [[Thermodynamics]]\n'
    expect(getAddress(createState(doc), doc.indexOf('Thermo') + 1)).toBeNull()
  })
})

describe('a link followed with the pointer', () => {
  const SOURCE = 'go [there](https://example.invalid) now'

  const open = (at: number | null, doc = SOURCE) => {
    const openState = vi.fn<(address: string) => void>()
    const passed = vi.fn<() => void>()
    const view = new EditorView({
      parent: document.body,
      state: createState(doc).update({
        effects: StateEffect.appendConfig.of([
          follow,
          addressOpener.of(openState),
          EditorView.domEventHandlers({ mousedown: passed }),
        ]),
      }).state,
    })
    const coordinates = vi.spyOn(view, 'posAtCoords').mockReturnValue(at)
    views.push(view)
    return { view, opened: openState, coordinates, passed }
  }

  const press = (view: EditorView, modifiers: MouseEventInit) => {
    const event = new MouseEvent('mousedown', {
      bubbles: true,
      cancelable: true,
      clientX: 12,
      clientY: 34,
      ...modifiers,
    })
    view.contentDOM.dispatchEvent(event)
    return event
  }

  const views: EditorView[] = []
  afterEach(() => {
    views.splice(0).forEach((view) => view.destroy())
  })

  it('is opened with the meta key retained', () => {
    const { view, opened, coordinates, passed } = open(SOURCE.indexOf('there') + 1)
    const event = press(view, { metaKey: true })
    expect(opened).toHaveBeenCalledWith('https://example.invalid')
    expect(event.defaultPrevented).toBe(true)
    expect(passed).not.toHaveBeenCalled()
    expect(coordinates).toHaveBeenCalledWith({ x: 12, y: 34 })
  })

  it('is opened with the control key retained', () => {
    const { view, opened, passed } = open(SOURCE.indexOf('there') + 1)
    const event = press(view, { ctrlKey: true })
    expect(opened).toHaveBeenCalledWith('https://example.invalid')
    expect(event.defaultPrevented).toBe(true)
    expect(passed).not.toHaveBeenCalled()
  })

  it('is left alone with no key retained', () => {
    const { view, opened, coordinates, passed } = open(SOURCE.indexOf('there') + 1)
    press(view, {})
    expect(opened).not.toHaveBeenCalled()
    expect(coordinates).not.toHaveBeenCalled()
    expect(passed).toHaveBeenCalledTimes(1)
  })

  it('is left alone where the pointer is over no text', () => {
    const { view, opened, passed } = open(null, '[there](https://example.invalid) now')
    press(view, { metaKey: true })
    expect(opened).not.toHaveBeenCalled()
    expect(passed).toHaveBeenCalledTimes(1)
  })

  it('is left alone where the pointer is over no link', () => {
    const { view, opened, passed } = open(1)
    press(view, { metaKey: true })
    expect(opened).not.toHaveBeenCalled()
    expect(passed).toHaveBeenCalledTimes(1)
  })
})

describe('the address of a link written in brackets alone', () => {
  it('is what is between the brackets', () => {
    const doc = 'see [thermodynamics] now'
    expect(getAddress(createState(doc), doc.indexOf('thermo') + 1)).toBe('thermodynamics')
  })
})
