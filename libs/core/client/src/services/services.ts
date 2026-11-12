import services from './services.json'

export interface ServiceInfo {
  title: string
  description: string
}

const SERVICES: Readonly<Record<string, ServiceInfo>> = services

/** Finds the title and description of a well-known service by its group key; undefined for any other group. */
export function findServiceInfo(group: string): ServiceInfo | undefined {
  const key = group.trim().toLowerCase()
  return Object.hasOwn(SERVICES, key) ? SERVICES[key] : undefined
}

/** Reads the title and description of a well-known service by its group key. */
export function getServiceInfo(group: string): ServiceInfo {
  return findServiceInfo(group) ?? { title: group, description: '' }
}
