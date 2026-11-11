import type { CommandTarget, Values } from '@/entities/command'
import type { ResultChoice } from '../search/lookup'

const toResultTarget = (place: NonNullable<ResultChoice['over']>) => ({
  tab: '',
  kind: null,
  path: place.path,
  file: place.file,
  title: place.title,
  ...(place.line === undefined ? {} : { line: place.line }),
  ...(place.selector === undefined ? {} : { selector: place.selector }),
})

/** Runs the command of a chosen action over what it was chosen on, with the text typed for it. */
export const createChoiceRunner =
  (
    runCommand: (id: string, target: CommandTarget, input?: string, values?: Values) => void,
    getTarget: () => CommandTarget,
  ) =>
  (choice: ResultChoice): void =>
    runCommand(
      choice.command,
      choice.over ? { ...getTarget(), ...toResultTarget(choice.over) } : getTarget(),
      choice.input || undefined,
      choice.values,
    )
