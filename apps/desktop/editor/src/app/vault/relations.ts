/**
 * Relationship methods of the window core: the calls of the note service that read and write
 * connections between notes.
 */
import {
  asFailure,
  asValue,
  findErrorCode,
  type Result,
  type VaultRelations,
} from '@numen/plugin-sdk'
import { notes } from '@/shared/clients'
import {
  CONNECTION_ROLES,
  findRelationError,
  mapBacklinks,
  mapConnection,
  mapConnectionToWire,
  roleCodes,
  writeMarker,
} from './words/relationWords'

/** The answer of a write: nothing where it landed, and why where it did not. */
const toResult = <E>(error: E | null): Result<void, E> =>
  error === null ? asValue(undefined) : asFailure(error)

export const relationsCore: VaultRelations = {
  roles: CONNECTION_ROLES,
  getConnection: async (first, second, type = '') => {
    const answer = await notes.getConnection({ first, second, type })
    const error = findErrorCode(answer)
    if (error !== null) return asFailure(error)
    if (answer.connection === undefined) return asFailure('unreadable')
    return asValue(mapConnection(answer.connection, type))
  },
  saveConnection: async (connection) =>
    toResult(
      findRelationError(
        await notes.saveConnection({ connection: mapConnectionToWire(connection) }),
      ),
    ),
  removeSide: async (from, to, type = '') =>
    toResult(findErrorCode(await notes.removeSide({ path: from, to, type }))),
  removeConnection: async (first, second, type = '') =>
    toResult(findErrorCode(await notes.removeConnection({ first, second, type }))),
  join: async (from, to, side) =>
    toResult(
      findRelationError(
        await notes.joinNotes({
          from,
          to,
          side: {
            role: roleCodes[side.role],
            type: side.type ?? '',
            note: side.reason ?? '',
            label: side.label ?? '',
            marker: writeMarker(side.marker),
          },
        }),
      ),
    ),
  createRelated: async (request) => {
    const answer = await notes.createRelatedNote({
      from: request.from,
      role: roleCodes[request.role],
      title: request.title ?? '',
      type: request.type ?? '',
      marker: writeMarker(request.marker),
    })
    if (answer.isNameExhausted) return asFailure('nameExhausted')
    const error = findErrorCode(answer)
    return error === null ? asValue(answer.path) : asFailure(error)
  },
  listBacklinks: async (path) => mapBacklinks((await notes.listBacklinks({ path })).backlinks),
}
