package runtime

import (
	"context"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

type Asset = effects.Asset
type AdapterAttempt = executor.AdapterAttempt
type AdapterResult = executor.AdapterResult

// Store is the exclusive execution owner of a single local SQLite database.
// Obtain an initialized Store from Create or Open. Share its pointer between
// goroutines. At most one Run or RunProgress may be active on it. Call Close
// to release ownership before reopening the database in another process.
type Store struct{ owner *executor.Store }

// Limits bounds encoded program, projection, observation, and reduction output
// sizes in bytes. Use DefaultLimits for ordinary stores. Zero does not select
// defaults. A nonpositive bound cannot accommodate a value of that kind, so
// Create and Open accept one without complaint and the refusal arrives at the
// first value it cannot admit. Limits belong to the Create, Open, Inspect or
// Replay call and are never retained, so reopening with different bounds is
// supported and a store is never permanently narrowed by them.
type Limits struct{ MaxProgramBytes, MaxDataBytes, MaxObservationBytes, MaxOutputBytes int }
type Artifact = executor.Artifact

// Provision holds the original programs, subjects, runs, permissions and retained
// assets. Obtain it from author.Application.Provision and review its permissions
// before Create. The zero value creates an empty store with no runnable agents.
// Omitted permissions grant no authority. Nil Composition and Reconciliation
// select ordinary execution.
type Provision struct {
	Reconciliation *ReconciliationProfile `json:",omitempty"`
	Composition    *CompositionProvision  `json:",omitempty"`
	// AssetScopes maps capability names to the exact IDs in Assets that their
	// adapters can receive. Omitted capabilities receive no assets in new stores.
	AssetScopes map[string][]string `json:",omitempty"`
	Assets      []Asset
	Programs    []Artifact
	Subjects    []Subject
	Runs        []InitialRun
	Permissions []Permission
}

type Subject = executor.Subject
type InitialRun = executor.InitialRun
type Permission = executor.Permission
type RunID = executor.RunID
type Projection = executor.Projection
type RunState = executor.RunState
type Authority = executor.Authority
type DispatchAuthority = executor.DispatchAuthority

// Input is a local application event. Its zero value is invalid. Supply
// nonempty TransportID, LogicalID, and Target, the port's Obligation, and a
// non-nil Payload (an empty map for an empty record).
type Input struct {
	TransportID, LogicalID, Target string
	// Generation zero binds to the target's current generation at first admission.
	Generation            int64
	Obligation, Operation string
	Payload               map[string]Value
}

// AdmissionResult records a durable admission decision. Check Accepted and
// Reason even when AdmitInput succeeds. "schema_invalid" is a known refusal.
type AdmissionResult = executor.AdmissionResult

// RunOptions bounds one call to Run or RunProgress. The zero value returns
// ErrLimit. Supply positive MaxSteps and MaxDispatch even for a computation
// that only performs local work.
type RunOptions struct {
	// Adapters maps retained capability names to trusted implementations for
	// this call. Nil registers none. Use the committed capability's exact name.
	Adapters map[string]effects.Adapter
	// MaxSteps caps committed reductions. MaxDispatch caps charged work,
	// including deliveries and settlements as well as external dispatch.
	MaxSteps, MaxDispatch int
	// MaxInFlight bounds concurrent calls per runner. Zero selects two.
	MaxInFlight int
	// ExternalTimeout bounds each external call. Zero selects two seconds.
	// A negative timeout or concurrency bound returns ErrLimit. Two seconds
	// suits a local adapter, not a model provider: a call cancelled by this
	// bound yields disposition "unknown", which leaves a possibly-charged
	// operation unresolved across restart. Set it from the capability's own
	// latency. The bundled model and forecast examples use 30 to 60 seconds.
	ExternalTimeout time.Duration
}

// RunReport combines counts for this call with retained state at its final cut.
// Quiescent means no eligible work remained and accounting succeeded. Check
// OpenRuns and UnknownAdapters or UnknownExternal for unresolved work.
type RunReport struct {
	Steps, Dispatches  int
	Quiescent          bool
	OpenRuns           []RunID
	UnresolvedIntents  []Intent
	PendingDeliveries  []string
	UnknownExternal    []string
	UnknownAdapters    []string
	Roots              []RootCompletion
	RejectedAdmissions int
}

// RunProgress contains the per-call execution counts and quiescence flag.
type RunProgress struct {
	Steps, Dispatches int
	Quiescent         bool
}

// Inspection is a detached raw table snapshot; Tables maps table names to rows.
type Inspection = executor.Inspection

// ReplayReport gives reconstructed Runs, Roots, HistoryCount, and IntentCount
// together with the store namespace, SQLite identity, and any conflicting receipts.
type ReplayReport = executor.ReplayReport
type ReceiptEvidence = executor.ReceiptEvidence
type Value = schema.Value

// ErrAdapterOutput identifies an invalid response from an application adapter.
// The paid operation may already have taken effect. Inspect its original ID.
var ErrAdapterOutput = executor.ErrAdapterOutput

// ErrCapabilityUnavailable means a committed request needs an adapter absent
// from the current registry. Register it on a subsequent Run to continue.
var ErrCapabilityUnavailable = executor.ErrCapabilityUnavailable

// ErrIneligible means the selected observation or continuation prerequisite
// is currently ineligible.
var ErrIneligible = executor.ErrIneligible

// ErrClosed means the addressed run or scope has closed.
var ErrClosed = executor.ErrClosed

// ErrSourceDenied means source authorization for an explicit lifecycle decision failed.
var ErrSourceDenied = executor.ErrSourceDenied

// ErrLimit identifies a resource bound, including an invalid or exhausted run budget.
var ErrLimit = executor.ErrLimit

// ErrVersion identifies an unsupported format, semantics, or optional protocol.
var ErrVersion = executor.ErrVersion

// ErrCorrupt indicates an integrity violation in retained evidence.
var ErrCorrupt = executor.ErrCorrupt

// ErrConflict indicates that a supplied identity or decision conflicts with existing evidence.
var ErrConflict = executor.ErrConflict

// ErrStale means the authority or evidence supplied by the caller is out of date.
var ErrStale = executor.ErrStale

// ErrLocked means another executor holds ownership of this database.
var ErrLocked = executor.ErrLocked

// ErrRunActive means a runner is already active on this Store.
var ErrRunActive = executor.ErrRunActive

// ErrDispatchMode means this store requires Run to dispatch an external effect.
var ErrDispatchMode = executor.ErrDispatchMode

// ErrDispatchAuthorityOpen means a dispatch drain was requested for authority that remains open.
var ErrDispatchAuthorityOpen = executor.ErrDispatchAuthorityOpen

// DefaultLimits allows one MiB each for encoded program, data, observation, and output.
func DefaultLimits() Limits { return Limits(executor.DefaultLimits()) }

// Create provisions a new database at path and returns its exclusive owner.
// Supply a local filesystem path, a validated deployment, and explicit limits.
// Deterministic provision validation precedes path creation. Invalid provisioning
// data can match fs.ErrInvalid, ErrLimit or ErrVersion. A pre-existing database
// is preserved. An interruption or storage failure after exclusive reservation
// can leave a database at path. Preserve its files and use Open to recover a
// committed ledger. Always close a successful Store.
func Create(ctx context.Context, path string, p Provision, l Limits) (*Store, error) {
	owner, err := executor.Create(ctx, path, executor.Provision(p), executor.Limits(l))
	if err != nil {
		return nil, err
	}
	return &Store{owner: owner}, nil
}

// Open acquires exclusive ownership of an existing store, validates its ledger,
// and renews open run epochs. Registered adapters belong to each Run call.
func Open(ctx context.Context, path string, l Limits) (*Store, error) {
	owner, err := executor.Open(ctx, path, executor.Limits(l))
	if err != nil {
		return nil, err
	}
	return &Store{owner: owner}, nil
}

// Inspect reads a coherent raw table snapshot without taking ownership.
// Use InspectExecution for typed execution, effect, and pending-work views.
func Inspect(ctx context.Context, path string, l Limits) (Inspection, error) {
	return executor.Inspect(ctx, path, executor.Limits(l))
}

// Replay verifies retained decisions and reconstructs execution in a read-only
// transaction. It uses the recorded evidence and makes no adapter calls.
func Replay(ctx context.Context, path string, l Limits) (ReplayReport, error) {
	return executor.Replay(ctx, path, executor.Limits(l))
}

// CompositionProvision configures the bounded composition protocol. A nil pointer
// in Provision selects the ordinary store profile. For a composition, supply
// its resource root, member bindings and limits as required by that deployment.
type CompositionProvision = executor.CompositionProvision
type ResourceRoot = executor.ResourceRoot

// ResourceLimits bounds a finite composition root. MaxInFlight must be positive.
// Zero MaxInvocations, MaxCreatedRuns, MaxDepth or MaxEffects grants no capacity
// in that dimension. Negative limits are invalid.
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

// RequestAuthorityVersion selects the bounded request-bound authority protocol.
const RequestAuthorityVersion = executor.RequestAuthorityVersion

// RequestAuthorityBindingVersion opts into exact historical request selection.
const RequestAuthorityBindingVersion = executor.RequestAuthorityBindingVersion

type AuthorityRoot = executor.AuthorityRoot
type DelegationClaim = executor.DelegationClaim
type RevocationClaim = executor.RevocationClaim
type AuthorityUse = executor.AuthorityUse
type AuthorityDecision = executor.AuthorityDecision
type EffectAuthority = executor.EffectAuthority

type AuthorityAdmission = executor.AuthorityAdmission
