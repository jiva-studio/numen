package service

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"

// anyTextKind is the kind that stands for every text kind.
const anyTextKind domain.TextKind = "*"

// getProofreadKinds is the text kinds the backend proofreads. The host narrows the versions a file
// menu offers by `when.liveTextKinds` of the proofread command in manifest.json.
func getProofreadKinds() map[domain.TextKind]bool {
	return map[domain.TextKind]bool{anyTextKind: true}
}
