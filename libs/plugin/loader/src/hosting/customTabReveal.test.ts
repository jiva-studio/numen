import { describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import type { EditorHandle } from '@numen/plugin-sdk'
import { createLineReveal } from './customTabReveal'

const createEditor = (isShown = true): EditorHandle & { reveal: ReturnType<typeof vi.fn> } => ({
  reveal: vi.fn(() => isShown),
})

const create = (isLoaded = ref(true)) => {
  const scope = effectScope()
  const lines = scope.run(() => createLineReveal(() => isLoaded.value))!
  return { lines, isLoaded }
}

describe('the line a note tab is asked to open on', () => {
  it('is revealed once the editor is attached', async () => {
    const { lines } = create()
    const editor = createEditor()

    lines.reveal(7)
    await nextTick()
    expect(editor.reveal).not.toHaveBeenCalled()
    lines.attach(editor)
    await nextTick()

    expect(editor.reveal).toHaveBeenCalledWith(7)
  })

  it('is revealed at once where the editor is already attached', async () => {
    const { lines } = create()
    const editor = createEditor()
    lines.attach(editor)

    lines.reveal(3)
    await nextTick()

    expect(editor.reveal).toHaveBeenCalledWith(3)
  })

  it('waits for the note to be read', async () => {
    const { lines, isLoaded } = create(ref(false))
    const editor = createEditor()
    lines.attach(editor)
    lines.reveal(5)
    await nextTick()
    expect(editor.reveal).not.toHaveBeenCalled()

    isLoaded.value = true
    await nextTick()
    await nextTick()

    expect(editor.reveal).toHaveBeenCalledWith(5)
  })

  it('is asked again until the editor takes it, and then not again', async () => {
    const { lines } = create()
    const editor = createEditor(false)
    lines.attach(editor)
    lines.reveal(2)
    await nextTick()

    expect(editor.reveal).toHaveBeenCalledWith(2)
    editor.reveal.mockClear()
    editor.reveal.mockReturnValue(true)
    lines.attach(editor)
    await nextTick()
    expect(editor.reveal).toHaveBeenCalledOnce()

    lines.attach(editor)
    await nextTick()

    expect(editor.reveal).toHaveBeenCalledOnce()
  })
})
