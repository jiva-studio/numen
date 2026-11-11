import { describe, expect, it, vi } from 'vitest'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { WORDS } from '@/shared/ui/words'
import { createWindowSettingsPart } from '@/testing/windowSettings'

describe('settings step groups', () => {
  it('returns no step groups for a command that is no setting', () => {
    const log = {
      getWriter: () => vi.fn(),
    } as never
    const windowTabs = {
      groups: {
        handle: {
          front: () => null,
          open: vi.fn(),
          close: vi.fn(),
        },
      },
    } as never

    const settings = createWindowSettingsPart({
      rest: createSettingsStore(SETTINGS_WORDS, vi.fn()),
      words: WORDS,
      log,
      windowTabs,
      onSizeChanged: vi.fn(),
      getSettingsViews: () => new Map(),
      getPlatform: () => 'linux',
    })

    const unknownGroups = settings.paletteLists.getStepGroups('unknown', '')
    expect(unknownGroups).toStrictEqual([])
  })
})
