import { Code, ConnectError } from '@connectrpc/connect'

/** The error a core call rejects with when the connection to the core is lost. */
export const createLostConnection = (message: string): ConnectError =>
  new ConnectError(message, Code.Unknown)
