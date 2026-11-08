package service

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"

// proofreadingKind is the kind of the text versions the plugin writes, a kind of its own, so that
// the condition of proofreading.delete in manifest.json names them and no version of another plugin.
const proofreadingKind domain.TextKind = "proofreading"

// anyTextKind is the kind that stands for every text kind.
const anyTextKind domain.TextKind = "*"

func getProofreadKinds() map[domain.TextKind]bool {
	return map[domain.TextKind]bool{anyTextKind: true}
}
