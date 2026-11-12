/** How a plugin's note tab reads its note and shows the failures of its writes. */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createNoteWriter, startNoteRead } from './customTabNote'

const settle = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0))

const createReadState = (hasEdited = false) => ({
  note: ref({ path: 'a.md', body: '' }),
  errorMessage: ref(''),
  isLoading: ref(true),
  hasEdited: () => hasEdited,
})

describe('startNoteRead', () => {
  it('takes the body of the read and stops loading', async () => {
    const state = createReadState()
    startNoteRead('a.md', () => Promise.resolve({ body: '# A' }), state)
    await settle()
    expect(state.note.value.body).toBe('# A')
    expect(state.isLoading.value).toBe(false)
  })

  it('keeps what was typed while the read was in flight', async () => {
    const state = createReadState(true)
    startNoteRead('a.md', () => Promise.resolve('# A'), state)
    await settle()
    expect(state.note.value.body).toBe('')
    expect(state.isLoading.value).toBe(false)
  })

  it('shows the error of the read and preserves body when read returns no body', async () => {
    const state = createReadState()
    state.note.value = { path: 'a.md', body: 'initial' }
    startNoteRead('a.md', () => Promise.resolve({ error: 'unreadable' } as never), state)
    await settle()
    expect(state.errorMessage.value).toBe('unreadable')
    expect(state.note.value.body).toBe('initial')

    const stateUndefined = createReadState()
    stateUndefined.note.value = { path: 'a.md', body: 'kept' }
    startNoteRead('a.md', () => Promise.resolve(undefined), stateUndefined)
    await settle()
    expect(stateUndefined.note.value.body).toBe('kept')
    expect(stateUndefined.errorMessage.value).toBe('')
  })

  it('shows the message of a read that was rejected and stops loading', async () => {
    const state = createReadState()
    startNoteRead('a.md', () => Promise.reject(new Error('unreachable')), state)
    await settle()
    expect(state.errorMessage.value).toBe('unreachable')
    expect(state.isLoading.value).toBe(false)
  })
})

describe('createNoteWriter', () => {
  it('writes nothing when the host cannot write', () => {
    const write = createNoteWriter('a.md', undefined, ref(''))
    expect(() => write('body')).not.toThrow()
  })

  it('shows a rejected write until a newer write succeeds', async () => {
    const errorMessage = ref('')
    const answers = [Promise.reject(new Error('unreachable')), Promise.resolve()]
    const write = createNoteWriter('a.md', () => answers.shift() ?? Promise.resolve(), errorMessage)

    write('one')
    await settle()
    expect(errorMessage.value).toBe('unreachable')

    write('two')
    await settle()
    expect(errorMessage.value).toBe('')

    errorMessage.value = 'read-error'
    write('three')
    await settle()
    expect(errorMessage.value).toBe('read-error')
  })

  it('ignores a stale resolved write when a newer write has failed', async () => {
    const errorMessage = ref('')
    let finishStaleLoad: () => void = () => undefined
    const answers: Array<() => Promise<void>> = [
      () => Promise.reject(new Error('first error')),
      () =>
        new Promise<void>((resolve) => {
          finishStaleLoad = resolve
        }),
      () => Promise.reject(new Error('latest error')),
    ]
    const write = createNoteWriter(
      'a.md',
      () => answers.shift()?.() ?? Promise.resolve(),
      errorMessage,
    )

    write('one')
    await settle()
    expect(errorMessage.value).toBe('first error')

    write('two')
    write('three')
    await settle()
    expect(errorMessage.value).toBe('latest error')

    finishStaleLoad()
    await settle()
    expect(errorMessage.value).toBe('latest error')
  })
})
