package testutil_test

import "testing"

// TestPackage ensures Go coverage tooling works for this package.
// The testutil package contains shared mocks and has no behavioral tests,
// but Go's covdata tool requires at least one test file to exist.
func TestPackage(t *testing.T) {
	t.Log("testutil: shared mock library — no behavioral tests")
}
