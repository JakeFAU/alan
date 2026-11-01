// Package runtime exposes the existing single-host durable executor. These
// provisional aliases delegate to the same Store, reducer, runner and replay.
package runtime

import (
	"context"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

type Asset = effects.Asset
type AdapterAttempt = executor.AdapterAttempt
type AdapterResult = executor.AdapterResult
type Store = executor.Store
type Limits = executor.Limits
type Artifact = executor.Artifact
type Provision = executor.Provision
type Subject = executor.Subject
type InitialRun = executor.InitialRun
type Permission = executor.Permission
type RunID = executor.RunID
type Projection = executor.Projection
type RunState = executor.RunState
type Authority = executor.Authority
type DispatchAuthority = executor.DispatchAuthority
type Input = executor.Input
type AdmissionResult = executor.AdmissionResult
type RunOptions = executor.RunOptions
type RunReport = executor.RunReport
type Inspection = executor.Inspection
type ReplayReport = executor.ReplayReport
type Value = schema.Value

var ErrLimit = executor.ErrLimit
var ErrVersion = executor.ErrVersion
var ErrCorrupt = executor.ErrCorrupt
var ErrConflict = executor.ErrConflict
var ErrStale = executor.ErrStale
var ErrLocked = executor.ErrLocked
var ErrRunActive = executor.ErrRunActive
var ErrDispatchMode = executor.ErrDispatchMode
var ErrDispatchAuthorityOpen = executor.ErrDispatchAuthorityOpen

func DefaultLimits() Limits { return executor.DefaultLimits() }
func Create(ctx context.Context, path string, p Provision, l Limits) (*Store, error) {
	return executor.Create(ctx, path, p, l)
}
func Open(ctx context.Context, path string, l Limits) (*Store, error) {
	return executor.Open(ctx, path, l)
}
func Inspect(ctx context.Context, path string, l Limits) (Inspection, error) {
	return executor.Inspect(ctx, path, l)
}
func Replay(ctx context.Context, path string, l Limits) (ReplayReport, error) {
	return executor.Replay(ctx, path, l)
}

type CompositionProvision = executor.CompositionProvision
type ResourceRoot = executor.ResourceRoot
type ResourceLimits = executor.ResourceLimits

type InvocationAdmission = executor.InvocationAdmission
type InvocationDescriptor = executor.InvocationDescriptor
type InvocationOutcome = executor.InvocationOutcome
type InvocationFreeze = executor.InvocationFreeze
type InvocationPreparation = executor.InvocationPreparation

type RootCompletion = executor.RootCompletion

// InvocationHandoff fixes the two endpoints of a target identity exchange.
type InvocationHandoff = executor.InvocationHandoff

const HandoffPeer = executor.HandoffPeer

type ReconciliationProfile = executor.ReconciliationProfile
type ReceiptGrant = executor.ReceiptGrant
