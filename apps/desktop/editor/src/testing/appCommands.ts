/**
 * The commands of the editor as a window collects them from its features, for a test that reads
 * the rows. Nothing runs them, so the window they use is empty.
 */
import { collectFeatureCommands, type FeatureCommandsDeps } from '@/app/commands/featureCommands'
import type { Command } from '@/entities/command'
import { WORDS } from '@/shared/words'

const NO_WINDOW: FeatureCommandsDeps = { notes: {} as never, files: {} as never, vaults: {} as never }

/** Every command of the editor in the order the palette draws them. */
export const createAppCommands = (agent?: string): readonly Command[] =>
  collectFeatureCommands(NO_WINDOW, WORDS, agent).commands
