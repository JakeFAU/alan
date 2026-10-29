//go:build integration

package substitutiontest

import "testing"

func TestSubstitutionRuntime(t *testing.T)  { runMatrix(t) }
func TestSubstitutionProbes(t *testing.T)   { runProbes(t) }
func TestSubstitutionRecovery(t *testing.T) { runRecovery(t) }
func TestSubstitutionClosure(t *testing.T)  { runClosure(t) }
func TestSubstitutionEffects(t *testing.T)  { runEffects(t) }
