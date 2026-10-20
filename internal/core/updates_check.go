//go:build windows

package core

import (
	"fmt"
	"sync"

	"goampp/internal/packages"
)

func RefreshAllServiceUpdates() {
	if !packages.UpdateOperationMu.TryLock() {
		App.AppendLog("updates: another package operation is already in progress")
		return
	}
	defer packages.UpdateOperationMu.Unlock()

	App.AppendLog("updates: resolving latest package versions...")
	packages.ResetGitHubCache()
	SetStatus("Checking package versions...")

	if err := packages.RefreshVariantCatalog(true); err != nil {
		App.AppendLog("updates: variant resolution: " + err.Error())
	}

	type updateRequest struct {
		name    string
		variant string
	}
	var requests []updateRequest
	for _, ms := range App.Services {
		if _, ok := packages.DownloadCatalog[ms.Conf.Name]; !ok {
			continue
		}
		requests = append(requests, updateRequest{
			name:    ms.Conf.Name,
			variant: ms.Conf.ActiveVersion,
		})
	}

	results := make(chan packages.ServiceUpdateInfo, len(requests))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, req := range requests {
		req := req
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results <- packages.CheckServiceUpdate(req.name, req.variant, App.BaseDir, App.AppendLog)
		}()
	}
	wg.Wait()
	close(results)

	fresh := make(map[string]packages.ServiceUpdateInfo, len(requests))
	available := 0
	installed := 0
	failed := 0
	for info := range results {
		fresh[info.Name] = info
		if info.Installed {
			installed++
		}
		if info.Available {
			available++
		}
		if info.Err != nil {
			failed++
			App.AppendLog(fmt.Sprintf("updates: %s: %v", info.Name, info.Err))
		}
	}

	packages.UpdateStateMu.Lock()
	packages.ServiceUpdates = fresh
	packages.UpdateStateMu.Unlock()

	if err := packages.PersistLatestCatalog(App.BaseDir, fresh); err != nil {
		App.AppendLog("updates: save cached versions: " + err.Error())
	}

	App.AppendLog(fmt.Sprintf("updates: resolved %d package(s), %d installed, %d update(s), %d error(s)",
		len(requests), installed, available, failed))

	uiUpdateCheckDone()
	switch {
	case failed > 0:
		SetStatus(fmt.Sprintf("Versions refreshed — %d error(s)", failed))
	case available == 0:
		SetStatus("Package versions refreshed")
	default:
		SetStatus(fmt.Sprintf("%d installed update(s)", available))
	}
}
