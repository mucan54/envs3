package engine

import (
	"context"
	"errors"
	"sort"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// EnvStatus holds status info for a single environment.
type EnvStatus struct {
	Name      string
	Version   int
	KeyCount  int
	UpdatedAt string
	HasAccess bool
}

// StatusResult holds the overall project status.
type StatusResult struct {
	Project      format.ProjectFile
	Environments []EnvStatus
}

// Status fetches project status information.
func (e *Engine) Status(ctx context.Context, pub [32]byte) (*StatusResult, error) {
	var project format.ProjectFile
	if _, err := e.getJSON(ctx, storage.ProjectPath(e.Project), &project); err != nil {
		return nil, err
	}

	var members format.MembersFile
	if _, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members); err != nil {
		return nil, err
	}

	// Determine which environments the user has access to
	accessibleEnvs := make(map[string]bool)
	for _, m := range members.Members {
		for _, env := range m.Environments {
			accessibleEnvs[env] = true
		}
	}

	envNames := project.Environments
	sort.Strings(envNames)

	var envStatuses []EnvStatus
	for _, envName := range envNames {
		status := EnvStatus{Name: envName}

		var bundle format.Bundle
		_, err := e.getJSON(ctx, storage.CurrentPath(e.Project, envName), &bundle)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			envStatuses = append(envStatuses, status)
			continue
		}
		if err == nil {
			status.Version = bundle.Version
			status.KeyCount = len(bundle.Secrets)
			status.UpdatedAt = bundle.UpdatedAt
		}
		status.HasAccess = accessibleEnvs[envName]
		envStatuses = append(envStatuses, status)
	}

	return &StatusResult{
		Project:      project,
		Environments: envStatuses,
	}, nil
}
