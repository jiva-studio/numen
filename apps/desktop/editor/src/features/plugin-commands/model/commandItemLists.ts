/**
 * The item lists of the commands the core lists them for, as the menus of the window read them: the
 * commands requested over one file together are read in one request, after the one read of the kind
 * of the file requested with them. Every menu open reads its lists again.
 */
import type { CommandItem } from '@numen/plugin-loader'
import type { ItemListAnswer, ItemListRequest } from '@/entities/plugin'
import type { FileKind } from '@/shared/model/fileTypes'

export interface CommandItemListsDeps {
  readonly listLists: (
    path: string,
    commands: readonly ItemListRequest[],
  ) => Promise<readonly ItemListAnswer[]>
  readonly fileKinds: (paths: readonly string[]) => Promise<ReadonlyMap<string, FileKind>>
  readonly schedule?: (run: () => void) => void
}

/** The reads over one file that go out together. */
interface Session {
  kinds?: Promise<ReadonlyMap<string, FileKind>>
  readonly requests: Map<string, ItemListRequest>
  readonly answers: Promise<readonly ItemListAnswer[]>
}

const getKey = (pluginId: string, command: string): string => `${pluginId}\n${command}`

const scheduleTask = (run: () => void): void => {
  setTimeout(run, 0)
}

export function createCommandItemLists(deps: CommandItemListsDeps) {
  const schedule = deps.schedule ?? scheduleTask
  const sessions = new Map<string, Session>()
  const waitTurn = (): Promise<void> => new Promise((resolve) => schedule(resolve))

  /** Sends the lists the session requested in one request once the kind of its file is read. */
  const send = async (
    path: string,
    turn: Promise<void>,
    readSession: () => Session,
  ): Promise<readonly ItemListAnswer[]> => {
    await turn
    const session = readSession()
    if (session.kinds) {
      await session.kinds.then(
        () => undefined,
        () => undefined,
      )
      await waitTurn()
    }
    if (sessions.get(path) === session) sessions.delete(path)
    const pending = [...session.requests.values()]
    if (pending.length === 0) return []
    return deps.listLists(path, pending)
  }

  /** The reads over the file that go out together, begun now where none is open. */
  const getSession = (path: string): Session => {
    const open = sessions.get(path)
    if (open) return open
    const turn = waitTurn()
    const session: Session = {
      requests: new Map(),
      answers: send(path, turn, () => session),
    }
    sessions.set(path, session)
    return session
  }

  return {
    /** The items the command offers over the file, read with the others requested now. */
    listItems: async (
      command: { readonly pluginId: string; readonly id: string },
      path: string,
    ): Promise<readonly CommandItem[]> => {
      const key = getKey(command.pluginId, command.id)
      const session = getSession(path)
      if (!session.requests.has(key)) {
        session.requests.set(key, { pluginId: command.pluginId, command: command.id })
      }
      const answers = await session.answers
      const answer = answers.find(
        (each) => each.pluginId === command.pluginId && each.command === command.id,
      )
      if (!answer) throw new Error(`no list of the items of ${command.id}`)
      if (!answer.result.ok) throw new Error(answer.result.error)
      return answer.result.value
    },
    /** The kinds of the files; the kind of one file is read once for the lists read over it. */
    fileKinds: (paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>> => {
      const [only] = paths
      if (only === undefined || paths.length > 1) return deps.fileKinds(paths)
      const session = getSession(only)
      session.kinds ??= deps.fileKinds([only])
      return session.kinds
    },
  }
}

export type CommandItemLists = ReturnType<typeof createCommandItemLists>
