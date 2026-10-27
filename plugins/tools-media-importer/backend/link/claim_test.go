package link

import "testing"

func TestTheSameAddressInAnotherVaultIsOtherWork(t *testing.T) {
	l := &Links{running: map[string]bool{}}
	physics := Link{Vault: "Physics", Key: "k1"}
	chemistry := Link{Vault: "Chemistry", Key: "k1"}

	if !l.claim("text", physics, physics.Key) || !l.claim("text", chemistry, chemistry.Key) {
		t.Fatal("a fetch of one address in a second vault waits for the first")
	}
	if l.claim("text", physics, physics.Key) {
		t.Error("a second fetch of one address in one vault is let run")
	}
	l.release("text", physics, physics.Key)
	if !l.claim("text", physics, physics.Key) {
		t.Error("a released fetch is still held")
	}
}
