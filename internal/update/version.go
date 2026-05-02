// Package update implements the lazy CLI update notice (D-06).
//
// The package is designed for embeddable use from any user-relevant
// command (`up`, `status`, `doctor`). Its public surface is intentionally
// small:
//
//   - IsNewer compares two semver-ish version strings.
//   - MaybePrint emits a single non-blocking notice line if a newer
//     release exists; silent on every failure path.
package update

import gv "github.com/hashicorp/go-version"

// IsNewer returns true if `latest` is strictly greater than `current`.
// Both arguments accept "v"-prefixed or unprefixed semver strings. On
// parse failure of either argument, IsNewer returns false (silent).
func IsNewer(current, latest string) bool {
	cur, err := gv.NewVersion(current)
	if err != nil {
		return false
	}
	lat, err := gv.NewVersion(latest)
	if err != nil {
		return false
	}
	return lat.GreaterThan(cur)
}
