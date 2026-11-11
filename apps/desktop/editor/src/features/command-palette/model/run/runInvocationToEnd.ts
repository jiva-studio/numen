import type { CommandDeps, CommandInvocation } from '@/entities/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { runInvocation } from '../handlers/handlers'

/**
 * Runs the invocation and answers with the path of the note or folder it made, empty where it made
 * none. Its errors are not written in the window: the first rejects the run.
 */
export async function runInvocationToEnd(
  invocation: CommandInvocation,
  deps: CommandDeps,
): Promise<string> {
  const errors: string[] = []
  const writeMessage: MessageWriter = (text, severity) => {
    if (severity === 'error') errors.push(text)
    else deps.writeMessage(text, severity)
  }
  const made = await runInvocation(invocation, { ...deps, writeMessage })
  if (errors[0] !== undefined) throw new Error(errors[0])
  return made
}
