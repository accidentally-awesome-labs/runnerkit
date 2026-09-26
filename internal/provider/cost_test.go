package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestNoHardcodedCost guards A-07: RunnerKit must never ship a price. The
// old constants ("approx €4.90/month", "approx €0.0081/hour") were applied
// to every server type and region; any amount shown now comes from the
// Hetzner pricing API.
func TestNoHardcodedCost(t *testing.T) {
	root := moduleRoot(t)
	pattern := regexp.MustCompile(`4\.90|0\.0081|approx €`)
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			if pattern.MatchString(line) {
				rel, _ := filepath.Rel(root, path)
				offenders = append(offenders, rel+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(offenders) > 0 {
		t.Fatalf("hard-coded cost found in non-test Go (prices must come from the Hetzner API):\n%s", strings.Join(offenders, "\n"))
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test directory")
		}
		dir = parent
	}
}

func TestDefaultHetznerPlanCarriesNoPrice(t *testing.T) {
	plan := HetznerProvisionPlan(ProvisionInput{RepoFullName: "owner/name", Profile: DefaultHetznerProfile()})
	if plan.EstimatedHourlyCost != nil || plan.EstimatedMonthlyCost != nil {
		t.Fatalf("static plan must not invent a price; got hourly=%#v monthly=%#v", plan.EstimatedHourlyCost, plan.EstimatedMonthlyCost)
	}
	var nilCost *CostEstimate
	if nilCost.Label() != "" || nilCost.Summary() != "" {
		t.Fatal("nil estimate must render empty, never a made-up figure")
	}
}

func TestCostEstimateLabels(t *testing.T) {
	fetchedAt := time.Date(2026, 9, 26, 10, 30, 0, 0, time.FixedZone("CEST", 2*3600))
	cost := &CostEstimate{Amount: "8.9131000000000000", Net: "7.4900000000000000", Gross: "8.9131000000000000", Currency: "EUR", Period: "month", Source: CostSourceHetznerAPI, FetchedAt: fetchedAt}
	if got, want := cost.Label(), "EUR 8.9131/month gross (7.49 net)"; got != want {
		t.Fatalf("Label() = %q, want %q", got, want)
	}
	if got, want := cost.Summary(), "EUR 8.9131/month gross (7.49 net), reported by the Hetzner API at 2026-09-26T08:30:00Z; excludes traffic overage"; got != want {
		t.Fatalf("Summary() = %q, want %q", got, want)
	}
	for in, want := range map[string]string{"5.0000": "5.00", "0.5": "0.5", "12": "12", "0.0072000": "0.0072", "100.00": "100.00"} {
		if got := trimDecimal(in); got != want {
			t.Fatalf("trimDecimal(%q) = %q, want %q", in, got, want)
		}
	}
	if got := (CostComponent{Resource: "primary_ipv4", Net: "0.5000000000", Gross: "0.5950000000"}).Label(); got != "primary_ipv4 0.50 net / 0.595 gross" {
		t.Fatalf("component label = %q", got)
	}
}

func TestFakeProviderPricesRefuseUnpricedLocation(t *testing.T) {
	fake := &FakeProvider{Prices: map[string]FakePrice{"cpx22/nbg1": {Monthly: CostEstimate{Amount: "1"}, Hourly: CostEstimate{Amount: "0.1"}}}}
	if _, err := fake.Plan(context.Background(), ProvisionInput{Profile: Profile{Provider: HetznerProvider, Region: "nbg1", ServerType: "cpx22"}}); err != nil {
		t.Fatalf("priced plan: %v", err)
	}
	_, err := fake.Plan(context.Background(), ProvisionInput{Profile: Profile{Provider: HetznerProvider, Region: "sin", ServerType: "cpx22"}})
	var unpriced *UnpricedLocationError
	if !errors.As(err, &unpriced) {
		t.Fatalf("unpriced plan error = %v", err)
	}
}
