package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider/hetzner"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

// pricingHCloudClient is a fake Hetzner API for the A-07 plan tests. It
// embeds the Client interface so any call outside the plan path (including
// every create call) panics or is counted instead of silently succeeding.
type pricingHCloudClient struct {
	hetzner.Client
	serverTypes map[string]*hcloud.ServerType
	pricing     hcloud.Pricing
	creates     int
}

func (c *pricingHCloudClient) GetLocation(_ context.Context, name string) (*hcloud.Location, error) {
	return &hcloud.Location{Name: name}, nil
}

func (c *pricingHCloudClient) GetServerType(_ context.Context, name string) (*hcloud.ServerType, error) {
	return c.serverTypes[name], nil
}

func (c *pricingHCloudClient) GetImage(_ context.Context, name string) (*hcloud.Image, error) {
	return &hcloud.Image{Name: name}, nil
}

func (c *pricingHCloudClient) GetPricing(context.Context) (hcloud.Pricing, error) {
	return c.pricing, nil
}

func (c *pricingHCloudClient) CreateSSHKey(context.Context, hcloud.SSHKeyCreateOpts) (*hcloud.SSHKey, error) {
	c.creates++
	return nil, nil
}

func (c *pricingHCloudClient) CreateFirewall(context.Context, hcloud.FirewallCreateOpts) (*hcloud.Firewall, error) {
	c.creates++
	return nil, nil
}

func (c *pricingHCloudClient) CreateServer(context.Context, hcloud.ServerCreateOpts) (*hcloud.Server, *hcloud.Action, error) {
	c.creates++
	return nil, nil, nil
}

// newPricingHCloudClient returns fixture prices that differ per (type,
// location), standing in for the Hetzner API response.
func newPricingHCloudClient() *pricingHCloudClient {
	serverPrice := func(location, hNet, hGross, mNet, mGross string) hcloud.ServerTypeLocationPricing {
		return hcloud.ServerTypeLocationPricing{
			Location: &hcloud.Location{Name: location},
			Hourly:   hcloud.Price{Net: hNet, Gross: hGross},
			Monthly:  hcloud.Price{Net: mNet, Gross: mGross},
		}
	}
	ipv4Price := func(location, hNet, hGross, mNet, mGross string) hcloud.PrimaryIPTypePricing {
		return hcloud.PrimaryIPTypePricing{
			Location: location,
			Hourly:   hcloud.PrimaryIPPrice{Net: hNet, Gross: hGross},
			Monthly:  hcloud.PrimaryIPPrice{Net: mNet, Gross: mGross},
		}
	}
	return &pricingHCloudClient{
		serverTypes: map[string]*hcloud.ServerType{
			"cpx22": {Name: "cpx22", Pricings: []hcloud.ServerTypeLocationPricing{serverPrice("nbg1", "0.0112000000000000", "0.0133280000000000", "6.9900000000000000", "8.3181000000000000")}},
			"ccx63": {Name: "ccx63", Pricings: []hcloud.ServerTypeLocationPricing{serverPrice("sin", "0.5520000000000000", "0.6568800000000000", "344.4900000000000000", "409.9431000000000000")}},
		},
		pricing: hcloud.Pricing{
			Image: hcloud.ImagePricing{PerGBMonth: hcloud.Price{Currency: "EUR"}},
			PrimaryIPs: []hcloud.PrimaryIPPricing{{Type: "ipv4", Pricings: []hcloud.PrimaryIPTypePricing{
				ipv4Price("nbg1", "0.0008000000000000", "0.0009520000000000", "0.5000000000000000", "0.5950000000000000"),
				ipv4Price("sin", "0.0012000000000000", "0.0014280000000000", "0.8000000000000000", "0.9520000000000000"),
			}}},
		},
	}
}

func pricingCloudDeps(t *testing.T, client *pricingHCloudClient, out, errOut *bytes.Buffer) Dependencies {
	t.Helper()
	cloud := hetzner.NewProvider(map[string]string{hetzner.EnvHCLOUDToken: "fake-token"}, hetzner.WithClient(client))
	cloud.Now = func() time.Time { return time.Date(2026, 9, 26, 10, 30, 0, 0, time.UTC) }
	return Dependencies{
		Version:        "test-version",
		Out:            out,
		Err:            errOut,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Providers:      provider.NewRegistry(cloud),
		StateBaseDir:   t.TempDir(),
		Sleep:          noSleep,
	}
}

// TestCloudPlan_PriceFromAPI (A-07): each plan shows the figure the Hetzner
// API reports for its own (server type, location); on the old code both
// printed the same hard-coded constant.
func TestCloudPlan_PriceFromAPI(t *testing.T) {
	cases := []struct {
		profile, region    string
		wantHuman          []string
		wantMonthlyGross   string
		wantMonthlyNet     string
		otherMonthlyFigure string
	}{
		{
			profile: "cpx22", region: "nbg1",
			wantHuman:          []string{"Estimated cost: EUR 0.01428/hour gross (0.012 net), EUR 8.9131/month gross (7.49 net)", "server:cpx22 6.99 net / 8.3181 gross; primary_ipv4 0.50 net / 0.595 gross"},
			wantMonthlyGross:   "8.9131000000000000",
			wantMonthlyNet:     "7.4900000000000000",
			otherMonthlyFigure: "410.8951",
		},
		{
			profile: "ccx63", region: "sin",
			wantHuman:          []string{"Estimated cost: EUR 0.658308/hour gross (0.5532 net), EUR 410.8951/month gross (345.29 net)", "server:ccx63 344.49 net / 409.9431 gross; primary_ipv4 0.80 net / 0.952 gross"},
			wantMonthlyGross:   "410.8951000000000000",
			wantMonthlyNet:     "345.2900000000000000",
			otherMonthlyFigure: "8.9131",
		},
	}
	for _, tc := range cases {
		t.Run(tc.profile+"/"+tc.region, func(t *testing.T) {
			// Human plan.
			client := newPricingHCloudClient()
			var out, errOut bytes.Buffer
			cmd := NewRootCommand(pricingCloudDeps(t, client, &out, &errOut))
			cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--cloud-region", tc.region, "--cloud-profile", tc.profile, "--yes", "--dry-run", "--no-color"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
			}
			stdout := strings.Join(strings.Fields(out.String()), " ")
			for _, want := range append(tc.wantHuman, "Prices reported by the Hetzner API at 2026-09-26T10:30:00Z; excludes traffic overage") {
				if !strings.Contains(stdout, want) {
					t.Fatalf("plan missing %q:\n%s", want, stdout)
				}
			}
			if strings.Contains(stdout, tc.otherMonthlyFigure) || strings.Contains(stdout, "4.90") {
				t.Fatalf("plan shows a figure that is not this plan's API price:\n%s", stdout)
			}

			// JSON plan.
			out.Reset()
			errOut.Reset()
			cmd = NewRootCommand(pricingCloudDeps(t, client, &out, &errOut))
			cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--cloud", "hetzner", "--cloud-region", tc.region, "--cloud-profile", tc.profile, "--yes", "--dry-run", "--no-color"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("json dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
			}
			var payload struct {
				Monthly   map[string]any `json:"estimated_monthly_cost"`
				CloudPlan struct {
					Monthly map[string]any `json:"estimated_monthly_cost"`
					Hourly  map[string]any `json:"estimated_hourly_cost"`
				} `json:"cloud_plan"`
			}
			if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
				t.Fatalf("invalid json: %v\n%s", err, out.String())
			}
			for _, monthly := range []map[string]any{payload.Monthly, payload.CloudPlan.Monthly} {
				if monthly["amount"] != tc.wantMonthlyGross || monthly["gross"] != tc.wantMonthlyGross || monthly["net"] != tc.wantMonthlyNet ||
					monthly["currency"] != "EUR" || monthly["source"] != "hetzner_api" || monthly["fetched_at"] != "2026-09-26T10:30:00Z" {
					t.Fatalf("estimated_monthly_cost = %#v\n%s", monthly, out.String())
				}
			}
			if payload.CloudPlan.Hourly["source"] != "hetzner_api" {
				t.Fatalf("estimated_hourly_cost = %#v", payload.CloudPlan.Hourly)
			}
			if client.creates != 0 {
				t.Fatalf("dry-run created %d resources", client.creates)
			}
		})
	}
}

// TestCloudPlan_UnpricedLocationRefuses (A-07): no price for the chosen
// type in the chosen location refuses with cloud_location_unpriced (exit
// 2) before any create call, even with --yes.
func TestCloudPlan_UnpricedLocationRefuses(t *testing.T) {
	client := newPricingHCloudClient()
	var out, errOut bytes.Buffer
	deps := pricingCloudDeps(t, client, &out, &errOut)
	cmd := NewRootCommand(deps)
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--cloud", "hetzner", "--cloud-region", "sin", "--cloud-profile", "cpx22", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unpriced location refusal\nstdout=%s\nstderr=%s", out.String(), errOut.String())
	}
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInvalidInput)
	}
	combined := out.String() + errOut.String()
	for _, want := range []string{`"code":"cloud_location_unpriced"`, "Hetzner reports no price for cpx22 in sin; choose another --cloud-region"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("refusal missing %q:\n%s", want, combined)
		}
	}
	if client.creates != 0 {
		t.Fatalf("unpriced location made %d create calls, want 0", client.creates)
	}
	if _, err := os.Stat(state.NewStore(deps.StateBaseDir).Path()); !os.IsNotExist(err) {
		t.Fatalf("unpriced refusal wrote state or stat failed unexpectedly: %v", err)
	}
}

// A provider plan without a price must never be shown or confirmed.
func TestCloudPlan_MissingPriceRefuses(t *testing.T) {
	fake := &provider.FakeProvider{Prices: map[string]provider.FakePrice{}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: newFakePermittedGitHubService(), RemoteExecutor: newFakeRemoteExecutor(), Providers: provider.NewRegistry(fake), StateBaseDir: t.TempDir(), Sleep: noSleep})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--cloud", "hetzner", "--cloud-region", "nbg1", "--yes", "--no-color"})
	err := cmd.Execute()
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Fatalf("ExitCode() = %d (err %v), want %d\n%s%s", got, err, ExitInvalidInput, out.String(), errOut.String())
	}
	if fake.ProvisionCalls != 0 {
		t.Fatalf("ProvisionCalls = %d, want 0", fake.ProvisionCalls)
	}
	if !strings.Contains(out.String()+errOut.String(), "cloud_location_unpriced") {
		t.Fatalf("missing cloud_location_unpriced:\n%s%s", out.String(), errOut.String())
	}
}
