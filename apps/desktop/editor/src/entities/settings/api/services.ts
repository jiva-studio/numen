/**
 * The title and description of the services a secret group is known by.
 */
import {
  findServiceInfo as findClientServiceInfo,
  getServiceInfo as getClientServiceInfo,
} from '@numen/core-client'

/** What the window shows of a well-known service. */
interface ServiceInfo {
  readonly title: string
  readonly description: string
}

/** The title and description of a well-known service by its group key; nothing for any other group. */
export const findServiceInfo = (group: string): ServiceInfo | undefined => {
  const info = findClientServiceInfo(group)
  return info && { title: info.title, description: info.description }
}

/** The title and description of a service by its group key; the group itself titles a group no service is known by. */
export const getServiceInfo = (group: string): ServiceInfo => {
  const info = getClientServiceInfo(group)
  return { title: info.title, description: info.description }
}
