package hosttest

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

func (v *Vault) AddLinks(ctx context.Context, additions []host.LinkAddition) ([]host.LinkAdded, error) {
	return nil, errors.New("not implemented")
}

func (v *Vault) UpdateLinks(
	ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat,
) (domain.FileStat, error) {
	return domain.FileStat{}, errors.New("not implemented")
}

func (v *Vault) RemoveLinks(
	ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat,
) (domain.FileStat, error) {
	return domain.FileStat{}, errors.New("not implemented")
}
