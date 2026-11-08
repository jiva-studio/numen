import services from './services.json'

export interface ServiceInfo {
  title: string
  description: string
}

const SERVICES: Readonly<Record<string, ServiceInfo>> = services

/** Reads the title and description of a well-known service by its group key. */
export function getServiceInfo(group: string): ServiceInfo {
  const key = group.trim().toLowerCase()
  const info = Object.hasOwn(SERVICES, key) ? SERVICES[key] : undefined
  return info ?? { title: group, description: '' }
}
