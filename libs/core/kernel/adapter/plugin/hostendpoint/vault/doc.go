// Package vault serves HostVaultService, the vault calls plugins make to the host, and declares the
// capabilities of the vault the host lends them. Each group of calls answers its procedures with its
// own state, and the service routes a procedure to the group that answers it.
package vault
