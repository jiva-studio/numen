package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	errNotFound  = errors.New("not found")
	errDuplicate = errors.New("duplicate")
)

const (
	statusRunning = "running"
	statusStopped = "stopped"
	statusError   = "error"
	statusUnknown = "unknown"
)

type bootstrapInstance struct {
	Name          string `json:"name"`
	MetadataURL   string `json:"metadata-url"`
	InstanceToken string `json:"instance-token"`
	OSType        string `json:"os_type"`
	OSArch        string `json:"arch"`
	Image         string `json:"image"`
	PoolID        string `json:"pool_id"`
}

type providerInstance struct {
	ProviderID    string `json:"provider_id"`
	Name          string `json:"name"`
	OSType        string `json:"os_type,omitempty"`
	OSName        string `json:"os_name,omitempty"`
	OSArch        string `json:"os_arch,omitempty"`
	Status        string `json:"status"`
	ProviderFault []byte `json:"provider_fault,omitempty"`
}

type provider struct {
	cfg             config
	tart            tartClient
	processes       processTable
	store           stateStore
	controllerID    string
	exitWaitSeconds int
}

func (p provider) createInstance(ctx context.Context, raw []byte) (providerInstance, error) {
	var params bootstrapInstance
	if err := json.Unmarshal(raw, &params); err != nil {
		return providerInstance{}, fmt.Errorf("parse bootstrap params: %w", err)
	}
	if !isValidInstanceName(params.Name) {
		return providerInstance{}, fmt.Errorf("invalid instance name %q", params.Name)
	}
	golden, ok := p.cfg.Images[params.Image]
	if !ok {
		return providerInstance{}, fmt.Errorf("image %q is not in the provider config", params.Image)
	}
	if p.store.hasInstance(params.Name) {
		return providerInstance{}, fmt.Errorf("instance %s: %w", params.Name, errDuplicate)
	}
	vms, err := p.tart.listVMs(ctx)
	if err != nil {
		return providerInstance{}, err
	}
	if _, exists := vms[params.Name]; exists {
		return providerInstance{}, fmt.Errorf("tart VM %s: %w", params.Name, errDuplicate)
	}
	state := instanceState{
		Name:         params.Name,
		VM:           params.Name,
		ControllerID: p.controllerID,
		PoolID:       params.PoolID,
		Image:        params.Image,
		OSType:       params.OSType,
		OSArch:       params.OSArch,
		IsReady:      false,
	}
	if err := p.store.writeInstance(state); err != nil {
		return providerInstance{}, err
	}
	if err := p.startNewInstance(ctx, state, golden, raw, params); err != nil {
		cleanupErr := p.deleteInstance(context.WithoutCancel(ctx), state.Name)
		return providerInstance{}, errors.Join(err, cleanupErr)
	}
	return createProviderInstance(state, statusRunning), nil
}

func (p provider) startNewInstance(ctx context.Context, state instanceState, golden string, raw []byte, params bootstrapInstance) error {
	if err := p.store.writeSharedFile(state.Name, "bootstrap.json", raw, 0o600); err != nil {
		return err
	}
	if err := p.store.writeSharedFile(state.Name, "bootstrap.sh", createBootstrapScript(params), 0o700); err != nil {
		return err
	}
	if err := p.tart.cloneVM(ctx, golden, state.VM); err != nil {
		return err
	}
	if err := p.tart.setResources(ctx, state.VM, p.cfg.CPU, p.cfg.Memory); err != nil {
		return err
	}
	if err := p.startVM(ctx, state); err != nil {
		return err
	}
	state.IsReady = true
	return p.store.writeInstance(state)
}

func (p provider) startVM(ctx context.Context, state instanceState) error {
	pid, err := p.tart.startVM(ctx, state.VM, p.store.getSharedDir(state.Name), p.store.getLogPath(state.Name))
	if err != nil {
		return err
	}
	return p.store.writePID(state.Name, pid)
}

func (p provider) readOwnInstance(name string) (instanceState, error) {
	state, err := p.store.readInstance(name)
	if err != nil {
		return instanceState{}, err
	}
	if state.ControllerID != p.controllerID {
		return instanceState{}, fmt.Errorf("instance %s belongs to another controller: %w", name, errNotFound)
	}
	return state, nil
}

func (p provider) deleteInstance(ctx context.Context, name string) error {
	state, err := p.readOwnInstance(name)
	if errors.Is(err, errNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	vms, err := p.tart.listVMs(ctx)
	if err != nil {
		return err
	}
	if vm, exists := vms[state.VM]; exists {
		if err := p.deleteVM(ctx, state, vm); err != nil {
			return err
		}
	}
	return p.store.deleteInstance(state.Name)
}

func (p provider) deleteVM(ctx context.Context, state instanceState, vm tartVM) error {
	if vm.Running {
		if err := p.stopVM(ctx, state); err != nil {
			return err
		}
	}
	return p.tart.deleteVM(ctx, state.VM)
}

func (p provider) getInstance(ctx context.Context, name string) (providerInstance, error) {
	state, err := p.readOwnInstance(name)
	if err != nil {
		return providerInstance{}, err
	}
	vms, err := p.tart.listVMs(ctx)
	if err != nil {
		return providerInstance{}, err
	}
	return describeInstance(state, vms), nil
}

func (p provider) listInstances(ctx context.Context, poolID string) ([]providerInstance, error) {
	states, err := p.store.listInstances()
	if err != nil {
		return nil, err
	}
	vms, err := p.tart.listVMs(ctx)
	if err != nil {
		return nil, err
	}
	instances := []providerInstance{}
	for _, state := range states {
		if state.ControllerID != p.controllerID || state.PoolID != poolID {
			continue
		}
		instances = append(instances, describeInstance(state, vms))
	}
	return instances, nil
}

func (p provider) removeAllInstances(ctx context.Context) error {
	states, err := p.store.listInstances()
	if err != nil {
		return err
	}
	var errs []error
	for _, state := range states {
		if state.ControllerID != p.controllerID {
			continue
		}
		errs = append(errs, p.deleteInstance(ctx, state.Name))
	}
	return errors.Join(errs...)
}

func (p provider) startInstance(ctx context.Context, name string) error {
	state, err := p.readOwnInstance(name)
	if err != nil {
		return err
	}
	vms, err := p.tart.listVMs(ctx)
	if err != nil {
		return err
	}
	vm, exists := vms[state.VM]
	if !exists {
		return fmt.Errorf("tart VM %s: %w", state.VM, errNotFound)
	}
	if vm.Running {
		return nil
	}
	return p.startVM(ctx, state)
}

func (p provider) stopInstance(ctx context.Context, name string) error {
	state, err := p.readOwnInstance(name)
	if err != nil {
		return err
	}
	vms, err := p.tart.listVMs(ctx)
	if err != nil {
		return err
	}
	vm, exists := vms[state.VM]
	if !exists || !vm.Running {
		return nil
	}
	return p.stopVM(ctx, state)
}

func describeInstance(state instanceState, vms map[string]tartVM) providerInstance {
	vm, exists := vms[state.VM]
	switch {
	case !exists && !state.IsReady:
		return createProviderInstance(state, statusUnknown)
	case !exists:
		instance := createProviderInstance(state, statusError)
		instance.ProviderFault = []byte("tart VM " + state.VM + " does not exist")
		return instance
	case vm.Running:
		return createProviderInstance(state, statusRunning)
	default:
		return createProviderInstance(state, statusStopped)
	}
}

func createProviderInstance(state instanceState, status string) providerInstance {
	return providerInstance{
		ProviderID:    state.Name,
		Name:          state.Name,
		OSType:        state.OSType,
		OSName:        "macos",
		OSArch:        state.OSArch,
		Status:        status,
		ProviderFault: nil,
	}
}

func createBootstrapScript(params bootstrapInstance) []byte {
	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("set -euo pipefail\n")
	b.WriteString("metadata_url=" + quoteShell(strings.TrimSuffix(params.MetadataURL, "/")) + "\n")
	b.WriteString("instance_token=" + quoteShell(params.InstanceToken) + "\n")
	b.WriteString("script=\"$(mktemp -t garm-install)\"\n")
	b.WriteString("curl --fail --silent --show-error --retry 30 --retry-delay 5 --retry-connrefused \\\n")
	b.WriteString("  -H \"Authorization: Bearer ${instance_token}\" \\\n")
	b.WriteString("  -o \"$script\" \"${metadata_url}/install-script/\"\n")
	b.WriteString("chmod +x \"$script\"\n")
	b.WriteString("exec \"$script\"\n")
	return []byte(b.String())
}

func quoteShell(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
