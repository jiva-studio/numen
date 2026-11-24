const BYTES_IN_KB = 2 ** 10
const BYTES_IN_MB = 2 ** 20
const BYTES_IN_GB = 2 ** 30

const trim = (value: number): number => Number(value.toFixed(1))

/** The size a vault takes in the cloud, such as "0 MB", "512 KB", "3.5 MB" or "1.2 GB". */
export function formatCloudSize(bytes: number): string {
  if (bytes >= BYTES_IN_GB) return `${trim(bytes / BYTES_IN_GB)} GB`
  if (bytes >= BYTES_IN_MB) return `${trim(bytes / BYTES_IN_MB)} MB`
  if (bytes > 0) return `${Math.ceil(bytes / BYTES_IN_KB)} KB`
  return '0 MB'
}
