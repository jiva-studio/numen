import {
  COMMAND,
  type CommandDeps,
  type CommandTarget,
  type VaultRef,
} from '@/shared/model/command'
import { formatError, type Caught } from '@/shared/api/errors'
import { createInvocation } from '../../lib/entries/invocation'
import { runInvocation } from '../handlers/handlers'

/** Opens a vault through the command that opens one; an error is written in the window. */
export const createVaultOpener =
  (deps: CommandDeps, getTarget: () => CommandTarget) =>
  (vault: VaultRef): Promise<void> =>
    runInvocation(createInvocation(COMMAND.openVault, { ...getTarget(), vault }), deps).then(
      () => undefined,
      (error: Caught) => deps.writeMessage(formatError(error), 'error'),
    )
