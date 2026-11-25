package vaultlink_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

func fileID(n int) []byte {
	id := make([]byte, 16)
	binary.BigEndian.PutUint64(id, uint64(n))
	return id
}

func TestLink_KeepsEveryRetiredFloorHoweverManyFilesWereRemoved(t *testing.T) {
	t.Parallel()
	var link vaultlink.Link
	for n := range 12000 {
		link.AddRetired(vaultlink.RetiredFile{FileID: fileID(n), Version: 5})
	}

	if got := link.GetFloor(fileID(0)); got != 5 {
		t.Errorf("floor of the oldest removal = %d, want 5 kept after 12000 removals", got)
	}
}

func TestLink_KeepsEveryFloorOfAFileThisDeviceRemovedHoweverManyWere(t *testing.T) {
	t.Parallel()
	var link vaultlink.Link
	for n := range 1500 {
		link.AddRemoved(vaultlink.RemovedFile{FileID: fileID(n), Version: 7})
	}

	if got := link.GetFloor(fileID(0)); got != 7 {
		t.Errorf("floor of the oldest own removal = %d, want 7 kept after 1500 removals", got)
	}
}

func TestLink_KeepsAFloorUntilTheEndOfItsRetentionAndDropsItAfter(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	link := vaultlink.Link{
		Retired: []vaultlink.RetiredFile{{FileID: fileID(1), Version: 3, At: start.Unix()}},
		Removed: []vaultlink.RemovedFile{{FileID: fileID(2), Version: 4, At: start.Unix()}},
	}

	link.PruneFloors(start.Add(vaultlink.FloorRetention))
	if link.GetFloor(fileID(1)) != 3 || link.GetFloor(fileID(2)) != 4 {
		t.Fatalf("floors = %d and %d at the last moment of the retention, want 3 and 4", link.GetFloor(fileID(1)), link.GetFloor(fileID(2)))
	}

	link.PruneFloors(start.Add(vaultlink.FloorRetention + time.Second))
	if link.GetFloor(fileID(1)) != 0 || link.GetFloor(fileID(2)) != 0 {
		t.Errorf("floors = %d and %d one second after the retention, want both dropped", link.GetFloor(fileID(1)), link.GetFloor(fileID(2)))
	}
}

func TestLink_DatesAFloorThatHasNoDateWhenItIsFirstPrunedAndKeepsItForTheRetention(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	var link vaultlink.Link
	link.AddRetired(vaultlink.RetiredFile{FileID: fileID(1), Version: 3})

	link.PruneFloors(start)
	link.PruneFloors(start.Add(vaultlink.FloorRetention))

	if got := link.GetFloor(fileID(1)); got != 3 {
		t.Errorf("floor = %d, want 3: the retention counts from the first look at it", got)
	}
}
