package container

import (
	"runtime"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"
)

// workspacePackages is the type-checked packages of every module of the workspace, tests included, by
// the directory of the module. The rules that walk it share one load and never change it.
type workspacePackages map[string][]*packages.Package

func (w workspacePackages) listModule(t *testing.T, dir string) []*packages.Package {
	t.Helper()
	pkgs, ok := w[dir]
	if !ok {
		t.Fatalf("%s is not a module of the workspace", dir)
	}
	return pkgs
}

// The rules that read the type-checked packages of the whole workspace run as subtests of this test, so
// the workspace is loaded and type-checked once and not once for each rule.
func TestTheTypeCheckedWorkspaceObeysItsRules(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks every module of the workspace")
	}
	workspace := loadWorkspace(t)
	t.Run("TheMachineClockIsReadOnlyByAClockPortImplementation", func(t *testing.T) {
		t.Parallel()
		checkMachineClockIsReadOnlyByAClockPortImplementation(t, workspace)
	})
	t.Run("TheSDKNamesNoProtocolType", func(t *testing.T) {
		t.Parallel()
		checkSDKNamesNoProtocolType(t, workspace)
	})
	t.Run("ProjectJSONKeysAreCamelCase", func(t *testing.T) {
		t.Parallel()
		checkProjectJSONKeysAreCamelCase(t, workspace)
	})
	t.Run("NoCoreTypeIsReexported", func(t *testing.T) {
		t.Parallel()
		checkNoCoreTypeIsReexported(t, workspace)
	})
	t.Run("EveryExhaustructIncludeMatchesAType", func(t *testing.T) {
		t.Parallel()
		checkExhaustructIncludesMatchTypes(t, workspace)
	})
	t.Run("InterfaceSizeCountsTheWholeMethodSet", func(t *testing.T) {
		t.Parallel()
		checkInterfaceSizeCountsTheWholeMethodSet(t, workspace)
	})
	t.Run("APortDeclaresAndDoesNotOrchestrate", func(t *testing.T) {
		t.Parallel()
		checkPortDeclaresAndDoesNotOrchestrate(t, workspace)
	})
	t.Run("TheCompositionRootIsOnePackage", func(t *testing.T) {
		t.Parallel()
		checkCompositionRootIsOnePackage(t, workspace)
	})
	t.Run("NoStructHoldsMoreThanFiveUseCases", func(t *testing.T) {
		t.Parallel()
		checkNoStructHoldsMoreThanFiveUseCases(t, workspace)
	})
	t.Run("NoWiringPackageExistsOutsideTheContainer", func(t *testing.T) {
		t.Parallel()
		checkNoWiringPackageExistsOutsideTheContainer(t, workspace)
	})
	t.Run("NoTypeExceedsItsLayersSize", func(t *testing.T) {
		t.Parallel()
		checkNoTypeExceedsItsLayersSize(t, workspace)
	})
	t.Run("DrivenAdaptersImportATechnology", func(t *testing.T) {
		t.Parallel()
		checkDrivenAdaptersImportATechnology(t, workspace)
	})
	t.Run("UseCasesImportOnlyDomainAndPorts", func(t *testing.T) {
		t.Parallel()
		checkUseCasesImportOnlyDomainAndPorts(t, workspace)
	})
	t.Run("EveryExhaustructPatternMatchesAType", func(t *testing.T) {
		t.Parallel()
		checkEveryExhaustructPatternMatchesAType(t, workspace)
	})
	t.Run("TheCompositionRootHoldsWiringOnly", func(t *testing.T) {
		t.Parallel()
		checkTheCompositionRootHoldsWiringOnly(t, workspace)
	})
	t.Run("ThePluginAdapterHasNoGodStruct", func(t *testing.T) {
		t.Parallel()
		checkThePluginAdapterHasNoGodStruct(t, workspace)
	})
	t.Run("TheContainerAssignsNoCapturedVariable", func(t *testing.T) {
		t.Parallel()
		checkTheContainerAssignsNoCapturedVariable(t, workspace)
	})
	t.Run("TheContainerDeclaresNoGenericFunction", func(t *testing.T) {
		t.Parallel()
		checkTheContainerDeclaresNoGenericFunction(t, workspace)
	})
	t.Run("NoUseCaseHasAMutatingWithMethod", func(t *testing.T) {
		t.Parallel()
		checkNoUseCaseHasAMutatingWithMethod(t, workspace)
	})
	t.Run("NoFunctionFieldIsAssignedAfterConstruction", func(t *testing.T) {
		t.Parallel()
		checkNoFunctionFieldIsAssignedAfterConstruction(t, workspace)
	})
}

func loadWorkspace(t *testing.T) workspacePackages {
	t.Helper()
	dirs := listWorkspaceModules(t)
	var (
		group   sync.WaitGroup
		mu      sync.Mutex
		byDir   = make(workspacePackages, len(dirs))
		failure error
	)
	slots := make(chan struct{}, max(1, runtime.GOMAXPROCS(0)/2))
	for _, dir := range dirs {
		group.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			pkgs, err := packages.Load(newTypedConfig(dir, nil), "./...")
			mu.Lock()
			defer mu.Unlock()
			if err != nil && failure == nil {
				failure = err
			}
			byDir[dir] = pkgs
		})
	}
	group.Wait()
	if failure != nil {
		t.Fatal(failure)
	}
	return byDir
}

func isTestVariant(pkg *packages.Package) bool {
	return strings.Contains(pkg.ID, " [") || strings.HasSuffix(pkg.ID, ".test")
}
