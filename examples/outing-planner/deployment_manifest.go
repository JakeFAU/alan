package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/JakeFAU/alan/runtime"
)

// This historical authoring baseline is independent of build_revision.
const delegatedFeatureBase = "23e888c57d829e98f7a62f8417fc7e0d27bae667"

type deploymentManifest struct {
	Version          int                  `json:"version"`
	ConfigurationSHA string               `json:"configuration_sha256"`
	Candidates       []candidate          `json:"candidates"`
	Artifacts        []manifestArtifact   `json:"artifacts"`
	Deployments      []manifestDeployment `json:"deployments"`
	Source           manifestSource       `json:"source"`
}

type manifestArtifact struct {
	Role      string `json:"role"`
	ProgramID string `json:"program_id"`
	Semantics string `json:"semantics"`
	Canonical []byte `json:"canonical_base64"`
}

type manifestDeployment struct {
	Weather         string                        `json:"weather"`
	HelperAdmission int64                         `json:"helper_admission"`
	InitialRuns     []runtime.InitialRun          `json:"initial_runs"`
	Subjects        []runtime.Subject             `json:"subjects"`
	Permissions     []runtime.Permission          `json:"permissions"`
	Composition     *runtime.CompositionProvision `json:"composition"`
}

type manifestSource struct {
	FeatureBase   string `json:"feature_base"`
	BuildRevision string `json:"build_revision"`
	BuildModified string `json:"build_modified"`
	GoVersion     string `json:"go_version"`
}

// Run before either Store is created. The artifact byte slices use base64 so
// formatting the enclosing document cannot change content-addressed bytes.
func freezeDeploymentManifest(
	configBytes []byte, plan normalizedPlan, baseline, delegated runtime.Provision, options deploymentOptions,
) deploymentManifest {
	source := manifestSource{FeatureBase: delegatedFeatureBase, BuildRevision: "unknown", BuildModified: "unknown", GoVersion: "unknown"}
	if build, ok := debug.ReadBuildInfo(); ok {
		source.GoVersion = build.GoVersion
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				source.BuildRevision = setting.Value
			case "vcs.modified":
				source.BuildModified = setting.Value
			}
		}
	}
	provisions := []struct {
		provision runtime.Provision
		options   deploymentOptions
	}{
		{baseline, defaultDeploymentOptions(assessFirst)},
		{delegated, options},
	}
	m := deploymentManifest{
		Version: 1, ConfigurationSHA: digestText(string(configBytes)), Candidates: append([]candidate{}, plan.Candidates...),
		Source: source,
	}
	seen := map[string]bool{}
	for _, deployment := range provisions {
		p, opts := deployment.provision, deployment.options
		m.Deployments = append(m.Deployments, manifestDeployment{
			Weather: opts.Weather, HelperAdmission: opts.HelperAdmission, InitialRuns: p.Runs,
			Subjects: p.Subjects, Permissions: p.Permissions, Composition: p.Composition,
		})
		roles := map[string]string{}
		for _, run := range p.Runs {
			role := "outing"
			if run.ID == weatherRun {
				role = "weather:" + opts.Weather
			}
			roles[run.ProgramID] = role
		}
		for _, template := range p.Composition.Templates {
			roles[template.Members[helperMember].ProgramID] = "helper:" + helperMember
		}
		for _, artifact := range p.Programs {
			if !seen[artifact.ID] {
				m.Artifacts = append(m.Artifacts, manifestArtifact{
					Role: roles[artifact.ID], ProgramID: artifact.ID, Semantics: artifact.Semantics, Canonical: slices.Clone(artifact.Bytes),
				})
				seen[artifact.ID] = true
			}
		}
	}
	slices.SortFunc(m.Artifacts, func(a, b manifestArtifact) int { return strings.Compare(a.Role, b.Role) })
	return m
}

func writeDeploymentManifest(dir string, m deploymentManifest) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode deployment manifest: %w", err)
	}
	return writeFresh(filepath.Join(dir, "deployment-manifest.json"), append(raw, '\n'))
}
