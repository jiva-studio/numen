import { describe, expect, it } from 'vitest'
import { EDITOR_MODULES } from './editorModules'

describe('editor host modules', () => {
  it.each(Object.entries(EDITOR_MODULES))('loads %s from the loader package', async (_name, load) => {
    expect(Object.keys(await load()).length).toBeGreaterThan(0)
  })
})
