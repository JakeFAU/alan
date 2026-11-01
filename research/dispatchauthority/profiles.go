// Package dispatchauthority specifies a finite projection of one immutable
// reservation and the distinct opportunities to act upon it. It has no runtime,
// database, network, fixture, test capture, or scheduler dependency.
package dispatchauthority

// Profile fixes environmental choices before exploration. Empty Restart means
// that only the original owner is present. Every operator action is optional.
type Profile struct {
	Name, Route, Member, Closure, Target, Reply, Restart string
	Duplicate, Peer                                      bool
}

func Profiles() []Profile {
	ps := []Profile{}
	add := func(name, route, member, closure, target, reply, restart string, duplicate, peer bool) {
		ps = append(ps, Profile{Name: name, Route: route, Member: member, Closure: closure, Target: target, Reply: reply, Restart: restart, Duplicate: duplicate, Peer: peer})
	}
	for _, route := range []string{"raw-keyed", "adapter"} {
		add(route+"-cancel", route, "none", "cancel", "source", "known", "", false, true)
		add(route+"-root", route, "root", "root-close", "scope", "known", "", false, false)
	}
	add("raw-abandon", "raw-plain", "none", "abandon", "source", "known", "", false, false)
	add("raw-retire", "raw-keyed", "none", "retire", "source", "known", "", false, false)
	add("raw-retry", "raw-keyed", "none", "retry-after-failure", "source", "known", "", false, false)
	add("raw-end", "raw-keyed", "root", "end", "source", "known", "", false, false)
	add("adapter-failure", "adapter", "nested", "failure", "source", "known", "", false, false)
	add("alias-source-ended-scope-open", "adapter", "root", "member-end", "scope", "known", "", false, false)
	for _, closure := range []string{"nested-fail", "nested-retire", "nested-publish"} {
		add(closure, "adapter", "nested", closure, "scope", "known", "", false, false)
	}
	add("two-prepared-direct", "raw-keyed", "none", "cancel", "source", "known", "", true, false)
	add("owner-close-open", "raw-keyed", "none", "none", "source", "known", "close", false, false)
	add("no-receipt", "adapter", "root", "cancel", "source", "loss", "", false, false)
	add("account-error", "adapter", "root", "root-close", "scope", "account-error", "", false, false)
	for _, route := range []string{"raw-keyed", "raw-plain", "adapter"} {
		add(route+"-recover-open", route, "none", "none", "source", "known", "kill", false, false)
	}
	add("raw-keyed-recover-revoked", "raw-keyed", "none", "cancel", "source", "known", "kill", false, false)
	return ps
}
