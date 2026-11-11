import { describe, expect, it } from 'vitest'
import { findIcon } from '@/shared/model/command'
import { WORDS as words } from '@/shared/ui/words'
import { WELCOME_WORDS } from '../words'
import { createWelcomeActions } from './screen'

describe('the icon a way into a vault is drawn with', () => {
  it('is there for every way the welcome screen offers', () => {
    const ways = createWelcomeActions(
      { vault: 'physics', isReady: true },
      { ...words, ...WELCOME_WORDS },
      'Linux',
    )
    for (const one of ways) expect(findIcon(one.id), one.id).not.toBeNull()
  })
})
