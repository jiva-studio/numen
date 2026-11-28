package vaultwire

import (
	"context"
	"encoding/hex"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type transfers struct {
	downloads ports.DownloadEntries
	uploads   ports.UploadEntries
}

// SetTransfers gives the methods the unfinished downloads and uploads of the vaults.
func (v *VaultList) SetTransfers(downloads ports.DownloadEntries, uploads ports.UploadEntries) {
	v.moving = transfers{downloads: downloads, uploads: uploads}
}

func (v *VaultList) setProgress(ctx context.Context, id domain.VaultID, one *api.Vault) error {
	absent, err := v.deps.Host.GetAbsent(id).ListAbsentFiles(ctx)
	if err != nil {
		return err
	}
	onDemand := map[string]bool{}
	for _, file := range absent {
		onDemand[file.Path] = true
		one.OnDemandBytes += uint64(max(file.Size, 0))
	}
	records, err := v.deps.State.ReadRecords(ctx, id)
	if err != nil {
		return err
	}
	for _, record := range records {
		if !onDemand[record.Path] {
			one.SyncedBytes += getStoredSize(record)
		}
	}
	waiting, err := v.getWaitingPaths(ctx, id)
	if err != nil {
		return err
	}
	for path := range onDemand {
		if !waiting[path] {
			one.OnDemandFiles++
		}
	}
	one.PendingFiles = len(waiting)
	one.Syncing = v.deps.Locker.IsLocked(id) || len(waiting) > 0
	one.TotalBytes = one.SyncedBytes + one.OnDemandBytes
	return nil
}

func getStoredSize(record transfer.FileRecord) uint64 {
	if record.BlobSize != 0 {
		return record.BlobSize
	}
	return uint64(max(record.Size, 0))
}

func (v *VaultList) getWaitingPaths(ctx context.Context, id domain.VaultID) (map[string]bool, error) {
	waiting := map[string]bool{}
	if v.moving.downloads != nil {
		entries, err := v.moving.downloads.ReadDownloads(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			waiting[entry.Path] = true
		}
	}
	if v.moving.uploads != nil {
		entries, err := v.moving.uploads.ReadUploads(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			waiting[entry.Path] = true
		}
	}
	return waiting, nil
}

func (v *VaultList) setCloudTotals(ctx context.Context, listed []api.Vault) {
	if v.deps.Cloud == nil {
		return
	}
	isSyncing := false
	for _, one := range listed {
		isSyncing = isSyncing || (one.Syncing && one.RemoteID != "")
	}
	if !isSyncing {
		return
	}
	rows, err := v.deps.Cloud.GetVaultUsage(ctx)
	if err != nil {
		return
	}
	usage := make(map[string]uint64, len(rows))
	for _, row := range rows {
		usage[hex.EncodeToString(row.GetVaultId())] = row.GetUsedBytes()
	}
	for i := range listed {
		if listed[i].Syncing && listed[i].RemoteID != "" {
			listed[i].TotalBytes = max(listed[i].TotalBytes, usage[listed[i].RemoteID])
		}
	}
}
