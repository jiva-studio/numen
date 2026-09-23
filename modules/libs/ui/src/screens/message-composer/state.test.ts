import { describe, expect, it } from 'vitest'
import { COMPOSER_STATES, composerState, keyIntent, getMessage } from './state'

const press = (over: Partial<Parameters<typeof keyIntent>[0]> = {}) => ({
  key: 'Enter',
  shiftKey: false,
  ...over,
})

describe('what state a composer is in', () => {
  it('has nothing to send when nothing was typed', () => {
    expect(composerState('', false)).toBe('empty')
  })

  it('has nothing to send when only whitespace was typed', () => {
    expect(composerState('   \n\t  ', false)).toBe('empty')
  })

  it('is ready once something was typed', () => {
    expect(composerState('hello', false)).toBe('ready')
  })

  it('is writing whatever was typed, because the answer owns the end of the field', () => {
    expect(composerState('', true)).toBe('writing')
    expect(composerState('hello', true)).toBe('writing')
  })

  it('can be pressed with something to send, and while an answer arrives', () => {
    expect(COMPOSER_STATES.empty.canAct).toBe(false)
    expect(COMPOSER_STATES.ready.canAct).toBe(true)
    expect(COMPOSER_STATES.writing.canAct).toBe(true)
  })

  it('stops the answer on its way, and sends the rest of the time', () => {
    expect(COMPOSER_STATES.empty.action).toBe('send')
    expect(COMPOSER_STATES.ready.action).toBe('send')
    expect(COMPOSER_STATES.writing.action).toBe('stop')
  })
})

describe('what was said', () => {
  it('is what is left after the whitespace around it', () => {
    expect(getMessage('  hello  ')).toBe('hello')
  })

  it('keeps the whitespace inside it', () => {
    expect(getMessage(' one  two \n three ')).toBe('one  two \n three')
  })
})

describe('what a key means', () => {
  it('sends on Enter', () => {
    expect(keyIntent(press())).toBe('submit')
  })

  it('breaks the line on Shift+Enter', () => {
    expect(keyIntent(press({ shiftKey: true }))).toBe('newline')
  })

  it('leaves every other key alone', () => {
    expect(keyIntent(press({ key: 'a' }))).toBe('pass')
    expect(keyIntent(press({ key: 'Escape' }))).toBe('pass')
    expect(keyIntent(press({ key: 'Tab' }))).toBe('pass')
  })

  it('leaves Enter to the input method that is composing', () => {
    expect(keyIntent(press({ isComposing: true }))).toBe('pass')
  })

  it('leaves Enter alone when the key is reported as composing', () => {
    expect(keyIntent(press({ keyCode: 229 }))).toBe('pass')
  })

  it('leaves Shift+Enter to the input method as well', () => {
    expect(keyIntent(press({ shiftKey: true, isComposing: true }))).toBe('pass')
  })
})
