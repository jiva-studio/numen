package service

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"

// anyTextKind is the kind that stands for every text kind.
const anyTextKind domain.TextKind = "*"

func getProofreadKinds() map[domain.TextKind]bool {
	return map[domain.TextKind]bool{anyTextKind: true}
}
