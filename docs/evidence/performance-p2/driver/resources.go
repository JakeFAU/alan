package main

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	goruntime "runtime"
	"runtime/metrics"
	"sort"
	"time"
)

var metricVocabulary = []string{
	"/cpu/classes/gc/mark/assist:cpu-seconds", "/cpu/classes/gc/mark/dedicated:cpu-seconds",
	"/cpu/classes/gc/mark/idle:cpu-seconds", "/cpu/classes/gc/pause:cpu-seconds", "/cpu/classes/gc/total:cpu-seconds",
	"/gc/cycles/automatic:gc-cycles", "/gc/cycles/forced:gc-cycles", "/gc/cycles/total:gc-cycles",
	"/sched/goroutines-created:goroutines",
}
var sampledMetricNames []string

func supportedMetrics() []string {
	available := map[string]bool{}
	for _, d := range metrics.All() {
		if d.Kind == metrics.KindFloat64 || d.Kind == metrics.KindUint64 {
			available[d.Name] = true
		}
	}
	names := make([]string, 0, len(metricVocabulary))
	for _, name := range metricVocabulary {
		if available[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	sampledMetricNames = names
	return append([]string{}, names...)
}

type resourceSnapshot struct {
	at                                time.Time
	user, system, rss                 int64
	alloc, mallocs, gc, forced, pause uint64
	counters                          map[string]uint64
	runtimeMetrics                    map[string]float64
}
type prefixConfiguration struct {
	Boundary        string `json:"boundary"`
	SnapshotMethod  string `json:"snapshot_method"`
	Preallocated    bool   `json:"preallocated"`
	ExtraInspection bool   `json:"extra_inspection"`
	ForcedGC        bool   `json:"forced_gc"`
	Serialization   string `json:"serialization"`
}

func frozenPrefixes() prefixConfiguration {
	return prefixConfiguration{Boundary: "contiguous-after-run-contact-and-returned-cut-proof-including-approval", SnapshotMethod: "ReadMemStats/Getrusage/runtime.metrics", Preallocated: true, Serialization: "after-Execute"}
}

type prefixRecord struct {
	Cut               cutProof `json:"cut"`
	Position          int      `json:"position"`
	DeclarationBound  int      `json:"declaration_bound"`
	RetainedPositions int      `json:"retained_positions"`
	Contacts          int      `json:"contacts"`
	ServerContacts    int      `json:"server_contacts"`
	Direct            stage    `json:"direct"`
	Cumulative        stage    `json:"cumulative"`
}

func takeSnapshot() (resourceSnapshot, error) {
	var mem goruntime.MemStats
	goruntime.ReadMemStats(&mem)
	u, err := usage()
	if err != nil {
		return resourceSnapshot{}, err
	}
	s := resourceSnapshot{user: cpu(u.Utime), system: cpu(u.Stime), rss: u.Maxrss,
		alloc: mem.TotalAlloc, mallocs: mem.Mallocs, gc: uint64(mem.NumGC), forced: uint64(mem.NumForcedGC), pause: mem.PauseTotalNs,
		runtimeMetrics: make(map[string]float64, len(sampledMetricNames))}
	samples := make([]metrics.Sample, len(sampledMetricNames))
	for i, name := range sampledMetricNames {
		samples[i].Name = name
	}
	metrics.Read(samples)
	for _, v := range samples {
		switch v.Value.Kind() {
		case metrics.KindFloat64:
			s.runtimeMetrics[v.Name] = v.Value.Float64()
		case metrics.KindUint64:
			value := v.Value.Uint64()
			if value > 1<<53 {
				return resourceSnapshot{}, fmt.Errorf("runtime metric %s exceeds exact float64 integer range", v.Name)
			}
			s.runtimeMetrics[v.Name] = float64(value)
		default:
			return resourceSnapshot{}, fmt.Errorf("unsupported metric %s", v.Name)
		}
	}
	for name, value := range s.runtimeMetrics {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return resourceSnapshot{}, fmt.Errorf("invalid runtime metric %s", name)
		}
	}
	if counterSnapshot != nil {
		s.counters = counterSnapshot()
	}
	s.at = time.Now()
	return s, nil
}
func checkedSub(a, b uint64) (uint64, error) {
	if b < a {
		return 0, errors.New("unsigned resource counter reset/underflow")
	}
	return b - a, nil
}
func subtractCounters(a, b map[string]uint64) (map[string]uint64, error) {
	for k := range a {
		if _, ok := b[k]; !ok {
			return nil, errors.New("counter disappeared/reset")
		}
	}
	out := make(map[string]uint64, len(b))
	for k, v := range b {
		d, err := checkedSub(a[k], v)
		if err != nil {
			return nil, fmt.Errorf("counter %s: %w", k, err)
		}
		out[k] = d
	}
	return out, nil
}
func snapshotDelta(a, b resourceSnapshot) (stage, error) {
	d := stage{WallNS: b.at.Sub(a.at).Nanoseconds(), UserCPUNS: b.user - a.user, SystemCPUNS: b.system - a.system, ProcessMaxRSS: b.rss, RuntimeMetrics: make(map[string]float64, len(a.runtimeMetrics))}
	if d.WallNS < 0 || d.UserCPUNS < 0 || d.SystemCPUNS < 0 || b.rss < a.rss {
		return stage{}, errors.New("nonmonotone process snapshot")
	}
	fields := []struct {
		a, b   uint64
		target *uint64
	}{
		{a.alloc, b.alloc, &d.TotalAlloc}, {a.mallocs, b.mallocs, &d.Mallocs}, {a.gc, b.gc, &d.GCCount},
		{a.forced, b.forced, &d.ForcedGCCount}, {a.pause, b.pause, &d.GCPauseNS}}
	for _, f := range fields {
		v, err := checkedSub(f.a, f.b)
		if err != nil {
			return stage{}, err
		}
		*f.target = v
	}
	if len(a.runtimeMetrics) != len(b.runtimeMetrics) {
		return stage{}, errors.New("runtime metric vocabulary changed")
	}
	for k, v := range a.runtimeMetrics {
		next, ok := b.runtimeMetrics[k]
		if !ok || math.IsNaN(next) || math.IsInf(next, 0) || next < v {
			return stage{}, fmt.Errorf("nonmonotone runtime metric %s", k)
		}
		d.RuntimeMetrics[k] = next - v
	}
	var err error
	d.Counters, err = subtractCounters(a.counters, b.counters)
	return d, err
}

// Treat stages as vectors of cumulative quantities. RSS remains a high-water
// gauge and is intentionally not in this arithmetic.
func stageSnapshot(s stage) resourceSnapshot {
	return resourceSnapshot{at: time.Unix(0, s.WallNS), user: s.UserCPUNS, system: s.SystemCPUNS, rss: 0,
		alloc: s.TotalAlloc, mallocs: s.Mallocs, gc: s.GCCount, forced: s.ForcedGCCount, pause: s.GCPauseNS,
		counters: s.Counters, runtimeMetrics: s.RuntimeMetrics}
}
func sameMeasured(a, b stage) bool {
	return a.WallNS == b.WallNS && a.UserCPUNS == b.UserCPUNS && a.SystemCPUNS == b.SystemCPUNS &&
		a.TotalAlloc == b.TotalAlloc && a.Mallocs == b.Mallocs && a.GCCount == b.GCCount &&
		a.ForcedGCCount == b.ForcedGCCount && a.GCPauseNS == b.GCPauseNS && reflect.DeepEqual(a.Counters, b.Counters) && sameMetrics(a.RuntimeMetrics, b.RuntimeMetrics)
}
func sameMetrics(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		w, ok := b[k]
		if !ok || math.IsNaN(v) || math.IsInf(v, 0) || math.IsNaN(w) || math.IsInf(w, 0) || math.Abs(v-w) > 1e-12+1e-12*math.Max(math.Abs(v), math.Abs(w)) {
			return false
		}
	}
	return true
}
func (r *report) verifyPrefixes() error {
	if !r.PrefixEnabled {
		if len(r.Prefixes) != 0 {
			return errors.New("prefix-off emitted records")
		}
		return nil
	}
	if len(r.Prefixes) != r.Bound {
		return errors.New("missing/extra prefixes")
	}
	previous := stage{RuntimeMetrics: map[string]float64{}, Counters: map[string]uint64{}}
	for i, p := range r.Prefixes {
		if i == 0 {
			for k := range p.Cumulative.RuntimeMetrics {
				previous.RuntimeMetrics[k] = 0
			}
			for k := range p.Cumulative.Counters {
				previous.Counters[k] = 0
			}
		}
		serverCount := 0
		if r.Transport.Mode == "http" {
			serverCount = i + 1
		}
		if p.Position != i+1 || p.DeclarationBound != r.Bound || p.RetainedPositions != i+1 || p.Contacts != i+1 || p.ServerContacts != serverCount {
			return fmt.Errorf("prefix %d count/order mismatch", i+1)
		}
		if err := verifyCutRecord(p.Cut, r.RootMembership, i+1); err != nil {
			return err
		}
		expected, err := snapshotDelta(stageSnapshot(previous), stageSnapshot(p.Cumulative))
		if err != nil {
			return err
		}
		if p.Cumulative.ProcessMaxRSS < previous.ProcessMaxRSS || p.Direct.ProcessMaxRSS != p.Cumulative.ProcessMaxRSS {
			return errors.New("prefix RSS gauge mismatch")
		}
		if !sameMeasured(expected, p.Direct) {
			return fmt.Errorf("prefix %d direct/cumulative arithmetic mismatch", i+1)
		}
		previous = p.Cumulative
	}
	for _, s := range r.Stages {
		if s.Name == "execute_window" || s.Name == "execution_window_including_reopen" {
			residual, err := snapshotDelta(stageSnapshot(previous), stageSnapshot(s))
			if err != nil {
				return fmt.Errorf("prefix exceeds enclosing Execute: %w", err)
			}
			residual.Name = "execute_window_minus_final_prefix"
			residual.ProcessMaxRSS = s.ProcessMaxRSS
			r.PrefixResidual = residual
			return nil
		}
	}
	return errors.New("no enclosing Execute window")
}
