/**
 * The item lists of the commands the core lists them for, as the menus of the window read them: the
 * commands requested over one file together are read in one request, after the one read of the kind
 * of the file requested with them, and each list is stored by its file, plugin and command until the
 * file or its text versions change, the vault is read again or the commands change.
 */
import type { CommandItem } from '@numen/plugin-host'
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

/** A command requested in a read, and how its answer is delivered. */
interface PendingList {
  readonly request: ItemListRequest
  readonly promise: Promise<readonly CommandItem[]>
  readonly resolve: (items: readonly CommandItem[]) => void
  readonly reject: (error: Error) => void
}

/** The reads over one file that go out together. */
interface Session {
  kinds?: Promise<ReadonlyMap<string, FileKind>>
  readonly lists: Map<string, PendingList>
}

const getKey = (pluginId: string, command: string): string => `${pluginId}\n${command}`

const scheduleTask = (run: () => void): void => {
  setTimeout(run, 0)
}

function createPendingList(request: ItemListRequest): PendingList {
  let resolve: PendingList['resolve'] = () => undefined
  let reject: PendingList['reject'] = () => undefined
  const promise = new Promise<readonly CommandItem[]>((onResolve, onReject) => {
    resolve = onResolve
    reject = onReject
  })
  return { request, promise, resolve, reject }
}

function createListCache() {
  const lists = new Map<string, Map<string, readonly CommandItem[]>>()
  const versions = new Map<string, number>()
  let epoch = 0
  const getVersion = (path: string): string => `${epoch}:${versions.get(path) ?? 0}`
  return {
    get: (path: string, key: string) => lists.get(path)?.get(key),
    /** What a read started now is checked against before its lists are stored. */
    getVersion,
    store: (path: string, version: string, key: string, items: readonly CommandItem[]) => {
      if (version !== getVersion(path)) return
      lists.set(path, new Map(lists.get(path)).set(key, items))
    },
    dropPaths: (paths: readonly string[]) => {
      for (const path of paths) {
        lists.delete(path)
        versions.set(path, (versions.get(path) ?? 0) + 1)
      }
    },
    dropAll: () => {
      lists.clear()
      epoch += 1
    },
  }
}

type ListCache = ReturnType<typeof createListCache>

/** One request of the lists of a session: its file, the version of the file, the lists requested. */
interface ListRead {
  readonly path: string
  readonly version: string
  readonly pending: readonly PendingList[]
}

function settleLists(cache: ListCache, read: ListRead, answers: readonly ItemListAnswer[]): void {
  for (const one of read.pending) {
    const { pluginId, command } = one.request
    const answer = answers.find((each) => each.pluginId === pluginId && each.command === command)
    if (!answer) one.reject(new Error(`no list of the items of ${command}`))
    else if (!answer.result.ok) one.reject(new Error(answer.result.error))
    else {
      cache.store(read.path, read.version, getKey(pluginId, command), answer.result.value)
      one.resolve(answer.result.value)
    }
  }
}

function failLists(read: ListRead, error: unknown): void {
  for (const one of read.pending) {
    one.reject(error instanceof Error ? error : new Error(String(error)))
  }
}

export function createCommandItemLists(deps: CommandItemListsDeps) {
  const schedule = deps.schedule ?? scheduleTask
  const sessions = new Map<string, Session>()
  const cache = createListCache()

  /** Sends the lists the session requested in one request, and settles each of them. */
  const send = (path: string, session: Session): void => {
    if (sessions.get(path) === session) sessions.delete(path)
    const pending = [...session.lists.values()]
    if (pending.length === 0) return
    const read: ListRead = { path, version: cache.getVersion(path), pending }
    deps
      .listLists(
        path,
        pending.map((one) => one.request),
      )
      .then(
        (answers) => settleLists(cache, read, answers),
        (error: unknown) => failLists(read, error),
      )
  }

  /** Sends the session once the kind of its file is read, after the reads that waited for it. */
  const flush = (path: string, session: Session): void => {
    const start = () => send(path, session)
    if (!session.kinds) {
      start()
      return
    }
    session.kinds.then(
      () => schedule(start),
      () => schedule(start),
    )
  }

  /** The reads over the file that go out together, begun now where none is open. */
  const getSession = (path: string): Session => {
    const open = sessions.get(path)
    if (open) return open
    const session: Session = { lists: new Map() }
    sessions.set(path, session)
    schedule(() => flush(path, session))
    return session
  }

  return {
    /** The items the command offers over the file: those stored, or read with the others requested now. */
    listItems: (
      command: { readonly pluginId: string; readonly id: string },
      path: string,
    ): Promise<readonly CommandItem[]> => {
      const key = getKey(command.pluginId, command.id)
      const items = cache.get(path, key)
      if (items) return Promise.resolve(items)
      const session = getSession(path)
      const existing = session.lists.get(key)
      if (existing) return existing.promise
      const pending = createPendingList({ pluginId: command.pluginId, command: command.id })
      session.lists.set(key, pending)
      return pending.promise
    },
    /** The kinds of the files; the kind of one file is read once for the lists read over it. */
    fileKinds: (paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>> => {
      const [only] = paths
      if (only === undefined || paths.length > 1) return deps.fileKinds(paths)
      const session = getSession(only)
      session.kinds ??= deps.fileKinds([only])
      return session.kinds
    },
    /** Forgets the lists of the files, whose content or text versions changed. */
    dropPaths: cache.dropPaths,
    /** Forgets every list, as the commands changed. */
    dropAll: cache.dropAll,
  }
}

export type CommandItemLists = ReturnType<typeof createCommandItemLists>
