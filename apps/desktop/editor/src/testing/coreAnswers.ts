/**
 * The window's clients and calls over a transport that answers the calls of the core from a test,
 * so the real clients encode each call and only the core is replaced.
 */
import type { Transport } from '@numen/core-client'
import {
  createCoreHandlerTransport,
  createCoreTransport,
  type CoreAnswers,
} from '@numen/core-client/testing'
import { createWindowApi } from '@/app/window/createWindowApi'
import {
  createPluginHostClients,
  createSettingsClients,
  createVaultClients,
  createWorkspaceClients,
  type WindowClients,
  type WindowTransports,
} from '@/shared/api/clients'

/** The transports of a window that send every request, keepalive or not, over `transport`. */
export const createTestTransports = (transport: Transport): WindowTransports => ({
  transport,
  keepalive: transport,
})

/** The transports of a window whose core answers as `answers` say; a call left out is rejected. */
export const createAnsweredTransports = (answers: CoreAnswers): WindowTransports =>
  createTestTransports(createCoreTransport(answers))

const createWindowClients = (transports: WindowTransports): WindowClients => ({
  ...createVaultClients(transports.transport),
  ...createWorkspaceClients(transports),
  ...createSettingsClients(transports.transport),
  ...createPluginHostClients(transports.transport),
})

/** The clients of a window whose core answers as `answers` say. */
export const createAnsweredClients = (answers: CoreAnswers) =>
  createWindowClients(createAnsweredTransports(answers))

/** Every call of a window whose core answers as `answers` say. */
export const createAnsweredApi = (answers: CoreAnswers) =>
  createWindowApi(createAnsweredTransports(answers))

/** The clients of a window whose core answers each call by the handler named by its method. */
export const createHandledClients = (handlers: Parameters<typeof createCoreHandlerTransport>[0]) =>
  createWindowClients(createTestTransports(createCoreHandlerTransport(handlers)))

/** Every call of a window whose core answers each call by the handler named by its method. */
export const createHandledApi = (handlers: Parameters<typeof createCoreHandlerTransport>[0]) =>
  createWindowApi(createTestTransports(createCoreHandlerTransport(handlers)))
