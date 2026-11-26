package vaultlink_test

import (
	"encoding/binary"
	"encoding/json"
	"slices"
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
	var link vaultlink.Link
	link.AddRetired(vaultlink.CreateRetiredFile(fileID(1), 3, start))
	link.AddRemoved(vaultlink.CreateRemovedFile(fileID(2), "Draft.md", nil, 4, start))

	link.PruneFloors(start.Add(vaultlink.FloorRetention))
	if link.GetFloor(fileID(1)) != 3 || link.GetFloor(fileID(2)) != 4 {
		t.Fatalf("floors = %d and %d at the last moment of the retention, want 3 and 4", link.GetFloor(fileID(1)), link.GetFloor(fileID(2)))
	}

	link.PruneFloors(start.Add(vaultlink.FloorRetention + time.Second))
	if link.GetFloor(fileID(1)) != 0 || link.GetFloor(fileID(2)) != 0 {
		t.Errorf("floors = %d and %d one second after the retention, want both dropped", link.GetFloor(fileID(1)), link.GetFloor(fileID(2)))
	}
}

func TestLink_DropsTheFloorsOlderThanTheRetentionAndKeepsTheNewerOnes(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	later := start.Add(30 * 24 * time.Hour)
	var link vaultlink.Link
	link.AddRetired(vaultlink.CreateRetiredFile(fileID(1), 3, start))
	link.AddRetired(vaultlink.CreateRetiredFile(fileID(2), 5, later))
	link.AddRemoved(vaultlink.CreateRemovedFile(fileID(3), "Old.md", nil, 4, start))
	link.AddRemoved(vaultlink.CreateRemovedFile(fileID(4), "New.md", nil, 6, later))

	link.PruneFloors(start.Add(vaultlink.FloorRetention + time.Hour))

	got := []uint64{link.GetFloor(fileID(1)), link.GetFloor(fileID(2)), link.GetFloor(fileID(3)), link.GetFloor(fileID(4))}
	if want := []uint64{0, 5, 0, 6}; !slices.Equal(got, want) {
		t.Errorf("floors = %v, want %v: the removals older than the retention dropped, the newer kept", got, want)
	}
}

func TestLink_KeepsTheTimeOfAFloorThroughASaveAndARead(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	var link vaultlink.Link
	link.AddRetired(vaultlink.CreateRetiredFile(fileID(1), 3, start))
	link.AddRemoved(vaultlink.CreateRemovedFile(fileID(2), "Draft.md", nil, 4, start))
	saved, err := json.Marshal(link)
	if err != nil {
		t.Fatal(err)
	}
	var restored vaultlink.Link
	if err := json.Unmarshal(saved, &restored); err != nil {
		t.Fatal(err)
	}

	restored.PruneFloors(start.Add(vaultlink.FloorRetention))
	if restored.GetFloor(fileID(1)) != 3 || restored.GetFloor(fileID(2)) != 4 {
		t.Fatalf("floors = %d and %d at the last moment of the retention, want 3 and 4", restored.GetFloor(fileID(1)), restored.GetFloor(fileID(2)))
	}
	restored.PruneFloors(start.Add(vaultlink.FloorRetention + time.Second))
	if restored.GetFloor(fileID(1)) != 0 || restored.GetFloor(fileID(2)) != 0 {
		t.Errorf("floors = %d and %d one second after the retention, want both dropped: the saved time was lost", restored.GetFloor(fileID(1)), restored.GetFloor(fileID(2)))
	}
}
