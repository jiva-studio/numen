import os from 'node:os'
import path from 'node:path'

/** The file the debug server listening on `port` writes its TLS certificate to, in PEM. */
export const getCertificateFile = (port) => path.join(os.tmpdir(), `numen-e2e-certificate-${port}.pem`)
