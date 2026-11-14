import { createRouterTransport, type ServiceImpl, type Transport } from '@connectrpc/connect'
import {
  CatalogService,
  CommandService,
  ConnectionCallService,
  FileService,
  NoteService,
  PluginHealthService,
  PluginService,
  ProviderConnectionService,
  SearchService,
  SettingsService,
  TextService,
  ThemeService,
  VaultService,
  VaultsService,
  WindowService,
  WorkspaceService,
} from '@numen/protocol'

/** Each service of the core, under the name a test answers it by. */
const SERVICES = {
  vault: VaultService,
  workspace: WorkspaceService,
  files: FileService,
  notes: NoteService,
  search: SearchService,
  vaultsService: VaultsService,
  settingsService: SettingsService,
  windowService: WindowService,
  theme: ThemeService,
  texts: TextService,
  pluginCommands: CommandService,
  pluginService: PluginService,
  pluginHealth: PluginHealthService,
  pluginCatalog: CatalogService,
  connections: ProviderConnectionService,
  connectionCalls: ConnectionCallService,
} as const

type ServiceName = keyof typeof SERVICES

/**
 * How the core answers each call, by the name of its service. A call left out is answered as
 * unimplemented, which rejects the call.
 */
export type CoreAnswers = {
  readonly [Name in ServiceName]?: Partial<ServiceImpl<(typeof SERVICES)[Name]>>
}

/**
 * A transport that answers every call from `answers`. A test gives it to the real clients, so it
 * runs the encoding and the cancelling of a call and replaces only the core.
 */
export const createCoreTransport = (answers: CoreAnswers): Transport =>
  createRouterTransport(({ service }) => {
    for (const [name, answer] of Object.entries(answers)) {
      service(SERVICES[name as ServiceName], answer as never)
    }
  })

/** A core method answered without its service named. */
type CoreHandler = (...request: never[]) => object | Promise<object>

/**
 * A transport that answers calls like `createCoreTransport`, from handlers named by their method
 * alone: each goes to the service that declares the method. A name no service declares throws, so a
 * misspelled handler cannot leave its call unanswered unnoticed.
 */
export const createCoreHandlerTransport = (
  handlers: Readonly<Record<string, CoreHandler>>,
): Transport => {
  const services = new Set<(typeof SERVICES)[ServiceName]>(Object.values(SERVICES))
  const declared = new Set<string>()
  for (const definition of services)
    for (const method of definition.methods) declared.add(method.localName)
  const misspelled = Object.keys(handlers).filter((name) => !declared.has(name))
  if (misspelled.length > 0) throw new Error(`no core method is named ${misspelled.join(', ')}`)
  return createRouterTransport(({ service }) => {
    for (const definition of services) {
      const answer: Record<string, CoreHandler> = {}
      for (const method of definition.methods) {
        const handler = handlers[method.localName]
        if (handler) answer[method.localName] = handler
      }
      service(definition, answer as never)
    }
  })
}
