import { createRouterTransport, type ServiceImpl } from '@connectrpc/connect'
import {
  AgentService,
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
import { replaceTransport } from '../transport'

/**
 * The service of each client the core client exports, under the client's name. The keepalive
 * workspace client is answered as `workspace`.
 */
const SERVICES = {
  vault: VaultService,
  agentService: AgentService,
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

type ClientName = keyof typeof SERVICES

/**
 * How the core answers each call, by the name of the client that makes it. A call left out is
 * answered as unimplemented, which rejects the call.
 */
export type CoreAnswers = {
  readonly [Name in ClientName]?: Partial<ServiceImpl<(typeof SERVICES)[Name]>>
}

/**
 * Answers every call of every core client from `answers`, through the real clients, so a test
 * runs the encoding and the cancelling of a call and replaces only the core.
 * The returned function restores the transport installed before.
 */
export const installCoreAnswers = (answers: CoreAnswers): (() => void) =>
  replaceTransport(
    createRouterTransport(({ service }) => {
      for (const [name, answer] of Object.entries(answers)) {
        service(SERVICES[name as ClientName], answer as never)
      }
    }),
  )

/** A core method answered without its service named. */
type CoreHandler = (...request: never[]) => object | Promise<object>

/**
 * Answers calls like `installCoreAnswers`, from handlers named by their method alone: each goes
 * to the service that declares the method. A name no service declares throws, so a misspelled
 * handler cannot leave its call unanswered unnoticed.
 */
export const installCoreHandlers = (
  handlers: Readonly<Record<string, CoreHandler>>,
): (() => void) => {
  const services = new Set<(typeof SERVICES)[ClientName]>(Object.values(SERVICES))
  const declared = new Set<string>()
  for (const definition of services)
    for (const method of definition.methods) declared.add(method.localName)
  const misspelled = Object.keys(handlers).filter((name) => !declared.has(name))
  if (misspelled.length > 0) throw new Error(`no core method is named ${misspelled.join(', ')}`)
  return replaceTransport(
    createRouterTransport(({ service }) => {
      for (const definition of services) {
        const answer: Record<string, CoreHandler> = {}
        for (const method of definition.methods) {
          const handler = handlers[method.localName]
          if (handler) answer[method.localName] = handler
        }
        service(definition, answer as never)
      }
    }),
  )
}
