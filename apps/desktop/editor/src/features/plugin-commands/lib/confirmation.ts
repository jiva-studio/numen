/**
 * The confirmation a command declares in its manifest entry, as the fields of a palette command: the
 * words are the ones the host commands give, so the palette asks both the same way.
 */
import type { CommandAnswers, CommandWarns } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'

interface Declared {
  readonly answers?: CommandAnswers
  readonly warns?: CommandWarns
}

/** What else the command asks of the person: a command with arguments asks no confirmation. */
interface Asks {
  readonly hasItems: boolean
  readonly hasArguments: boolean
}

/**
 * The confirmation a command asks for. A command with items asks the yes or no once its item is
 * picked; any other command asks the yes or no, or the name typed again, as its only step.
 */
export const getConfirmation = ({ answers, warns }: Declared, asks: Asks): Partial<Command> => {
  if (asks.hasArguments) return {}
  if (answers)
    return asks.hasItems ? { next: 'confirmation', answers } : { input: 'confirmation', answers }
  return warns && !asks.hasItems ? { input: 'retypedName', warns } : {}
}
