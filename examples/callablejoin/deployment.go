// Package callablejoin retains the fixed Phase 1N callable join and its outer caller.
package callablejoin

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type Deployment struct {
	Provision                  runtime.Provision
	Candidate, Successor, Join author.Template
	JoinArtifact, Outer        author.Artifact
	FixedRuns                  int
}

// New uses the historical builder solely to obtain original S and validate the
// finite request. Its old parent/provision are discarded, never executed.
func New(candidate author.Template, config substitution.Config) (Deployment, error) {
	d := Deployment{Candidate: candidate, FixedRuns: 1}
	originalConfig := config
	originalConfig.Shape = "sequential"
	original, err := substitution.New(candidate, originalConfig)
	if err != nil {
		return d, err
	}
	d.Successor = original.Peer
	d.JoinArtifact, d.Join, err = NewJoin(candidate, d.Successor)
	if err != nil {
		return d, err
	}
	d.Outer, err = NewOuter(d.Join)
	if err != nil {
		return d, err
	}
	d.Provision, err = (author.Application{"root": {Artifact: d.Outer, Inputs: map[string]schema.Value{
		"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value), "request_mode": schema.Text(config.Request.Mode),
	}}}).Provision()
	if err != nil {
		return d, err
	}
	allowed := make([]string, 0, len(d.Provision.Composition.Templates))
	for _, t := range d.Provision.Composition.Templates {
		allowed = append(allowed, t.ID)
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: allowed, Limits: config.Limits}}
	return d, nil
}
