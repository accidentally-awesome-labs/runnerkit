package hetzner

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

// Prices below are test fixtures standing in for the Hetzner API response;
// they deliberately differ per (type, location) so a plan can only show the
// right figure if it read the API.

func fakeServerPrice(location, hourlyNet, hourlyGross, monthlyNet, monthlyGross string) hcloud.ServerTypeLocationPricing {
	return hcloud.ServerTypeLocationPricing{
		Location: &hcloud.Location{Name: location},
		Hourly:   hcloud.Price{Net: hourlyNet, Gross: hourlyGross},
		Monthly:  hcloud.Price{Net: monthlyNet, Gross: monthlyGross},
	}
}

func fakeServerType(name string, prices ...hcloud.ServerTypeLocationPricing) *hcloud.ServerType {
	return &hcloud.ServerType{Name: name, Pricings: prices}
}

func fakeIPv4Price(location, hourlyNet, hourlyGross, monthlyNet, monthlyGross string) hcloud.PrimaryIPTypePricing {
	return hcloud.PrimaryIPTypePricing{
		Location: location,
		Hourly:   hcloud.PrimaryIPPrice{Net: hourlyNet, Gross: hourlyGross},
		Monthly:  hcloud.PrimaryIPPrice{Net: monthlyNet, Gross: monthlyGross},
	}
}

// fakePricing mirrors hcloud.PricingFromSchema, which copies the response's
// top-level currency into every Price it builds.
func fakePricing(currency string, ipv4 ...hcloud.PrimaryIPTypePricing) hcloud.Pricing {
	return hcloud.Pricing{
		Image:      hcloud.ImagePricing{PerGBMonth: hcloud.Price{Currency: currency, Net: "0.0100", Gross: "0.0119"}},
		PrimaryIPs: []hcloud.PrimaryIPPricing{{Type: "ipv4", Pricings: ipv4}},
	}
}

// pricingFakeClient returns server types by name so one client can price
// several (type, location) plans.
type pricingFakeClient struct {
	*fakeClient
	serverTypes map[string]*hcloud.ServerType
}

func (f *pricingFakeClient) GetServerType(_ context.Context, name string) (*hcloud.ServerType, error) {
	f.calls = append(f.calls, "lookup:server_type")
	return f.serverTypes[name], nil
}

func newPricingFakeClient() *pricingFakeClient {
	base := newFakeClient()
	base.pricing = fakePricing("EUR",
		fakeIPv4Price("nbg1", "0.0008000000000000", "0.0009520000000000", "0.5000000000000000", "0.5950000000000000"),
		fakeIPv4Price("sin", "0.0012000000000000", "0.0014280000000000", "0.8000000000000000", "0.9520000000000000"),
	)
	return &pricingFakeClient{
		fakeClient: base,
		serverTypes: map[string]*hcloud.ServerType{
			"cpx22": fakeServerType("cpx22", fakeServerPrice("nbg1", "0.0112000000000000", "0.0133280000000000", "6.9900000000000000", "8.3181000000000000")),
			"ccx63": fakeServerType("ccx63", fakeServerPrice("sin", "0.5520000000000000", "0.6568800000000000", "344.4900000000000000", "409.9431000000000000")),
		},
	}
}

func pricedInput(serverType, region string) provider.ProvisionInput {
	input := provisionInput()
	input.Profile.ServerType = serverType
	input.Profile.Region = region
	return input
}

func TestCloudPlan_PriceFromAPI(t *testing.T) {
	fetchedAt := time.Date(2026, 9, 26, 10, 30, 0, 0, time.UTC)
	client := newPricingFakeClient()
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	p.Now = func() time.Time { return fetchedAt }

	cases := []struct {
		serverType, region                               string
		monthlyNet, monthlyGross, hourlyNet, hourlyGross string
		components                                       []provider.CostComponent
	}{
		{
			serverType: "cpx22", region: "nbg1",
			monthlyNet: "7.4900000000000000", monthlyGross: "8.9131000000000000",
			hourlyNet: "0.0120000000000000", hourlyGross: "0.0142800000000000",
			components: []provider.CostComponent{
				{Resource: "server:cpx22", Net: "6.9900000000000000", Gross: "8.3181000000000000"},
				{Resource: "primary_ipv4", Net: "0.5000000000000000", Gross: "0.5950000000000000"},
			},
		},
		{
			serverType: "ccx63", region: "sin",
			monthlyNet: "345.2900000000000000", monthlyGross: "410.8951000000000000",
			hourlyNet: "0.5532000000000000", hourlyGross: "0.6583080000000000",
			components: []provider.CostComponent{
				{Resource: "server:ccx63", Net: "344.4900000000000000", Gross: "409.9431000000000000"},
				{Resource: "primary_ipv4", Net: "0.8000000000000000", Gross: "0.9520000000000000"},
			},
		},
	}
	labels := map[string]string{}
	for _, tc := range cases {
		plan, err := p.Plan(context.Background(), pricedInput(tc.serverType, tc.region))
		if err != nil {
			t.Fatalf("Plan(%s, %s) returned error: %v", tc.serverType, tc.region, err)
		}
		monthly, hourly := plan.EstimatedMonthlyCost, plan.EstimatedHourlyCost
		if monthly == nil || hourly == nil {
			t.Fatalf("Plan(%s, %s) carries no price: %#v", tc.serverType, tc.region, plan)
		}
		if monthly.Net != tc.monthlyNet || monthly.Gross != tc.monthlyGross || monthly.Amount != tc.monthlyGross {
			t.Fatalf("%s/%s monthly = %#v, want net %s gross %s", tc.serverType, tc.region, monthly, tc.monthlyNet, tc.monthlyGross)
		}
		if hourly.Net != tc.hourlyNet || hourly.Gross != tc.hourlyGross {
			t.Fatalf("%s/%s hourly = %#v, want net %s gross %s", tc.serverType, tc.region, hourly, tc.hourlyNet, tc.hourlyGross)
		}
		for _, c := range []*provider.CostEstimate{monthly, hourly} {
			if c.Currency != "EUR" || c.Source != provider.CostSourceHetznerAPI || !c.FetchedAt.Equal(fetchedAt) {
				t.Fatalf("%s/%s estimate provenance = %#v", tc.serverType, tc.region, c)
			}
		}
		if !reflect.DeepEqual(monthly.Components, tc.components) {
			t.Fatalf("%s/%s monthly components = %#v, want the raw API strings %#v", tc.serverType, tc.region, monthly.Components, tc.components)
		}
		encoded, err := json.Marshal(plan)
		if err != nil {
			t.Fatalf("marshal plan: %v", err)
		}
		for _, want := range []string{`"estimated_monthly_cost":{"amount":"` + tc.monthlyGross + `"`, `"source":"hetzner_api"`, `"fetched_at":"2026-09-26T10:30:00Z"`, `"currency":"EUR"`} {
			if !strings.Contains(string(encoded), want) {
				t.Fatalf("%s/%s plan JSON missing %s:\n%s", tc.serverType, tc.region, want, encoded)
			}
		}
		labels[tc.serverType] = monthly.Label()
	}
	if labels["cpx22"] != "EUR 8.9131/month gross (7.49 net)" || labels["ccx63"] != "EUR 410.8951/month gross (345.29 net)" {
		t.Fatalf("each plan must show its own API figure; got %#v", labels)
	}
	if gotCreates := createCalls(client.calls); len(gotCreates) != 0 {
		t.Fatalf("Plan created resources: %#v", gotCreates)
	}
}

func TestCloudPlan_UnpricedLocationRefuses(t *testing.T) {
	cases := []struct {
		name         string
		serverType   string
		region       string
		wantResource string
	}{
		{name: "server type unpriced in location", serverType: "cpx22", region: "sin", wantResource: "server"},
		{name: "primary IPv4 unpriced in location", serverType: "cpx22", region: "nbg1", wantResource: "primary_ipv4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newPricingFakeClient()
			if tc.wantResource == "primary_ipv4" {
				client.pricing = fakePricing("EUR", fakeIPv4Price("sin", "0.0012", "0.0014", "0.80", "0.95"))
			}
			// Provision must also resolve the location/image before pricing.
			client.location = &hcloud.Location{Name: tc.region}
			p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
			input := pricedInput(tc.serverType, tc.region)

			_, err := p.Plan(context.Background(), input)
			var unpriced *provider.UnpricedLocationError
			if !errors.As(err, &unpriced) {
				t.Fatalf("Plan error = %T %v, want *provider.UnpricedLocationError", err, err)
			}
			if unpriced.Resource != tc.wantResource || unpriced.Location != tc.region || unpriced.ServerType != tc.serverType {
				t.Fatalf("unpriced error = %#v", unpriced)
			}
			if !strings.Contains(err.Error(), "Hetzner reports no") || !strings.Contains(err.Error(), "choose another --cloud-region") {
				t.Fatalf("unpriced message = %q", err.Error())
			}

			_, err = p.Provision(context.Background(), input)
			if !errors.As(err, &unpriced) {
				t.Fatalf("Provision error = %T %v, want *provider.UnpricedLocationError", err, err)
			}
			if gotCreates := createCalls(client.calls); len(gotCreates) != 0 {
				t.Fatalf("unpriced location must refuse before any create call; creates = %#v", gotCreates)
			}
		})
	}
}

func TestQuoteCostFallsBackToPricingServerTypeList(t *testing.T) {
	pricing := fakePricing("EUR", fakeIPv4Price("hel1", "0.0008", "0.0010", "0.50", "0.60"))
	pricing.ServerTypes = []hcloud.ServerTypePricing{{
		ServerType: &hcloud.ServerType{Name: "cx32"},
		Pricings: []hcloud.ServerTypeLocationPricing{{
			Location: &hcloud.Location{Name: "hel1"},
			Hourly:   hcloud.Price{Currency: "EUR", Net: "0.0100", Gross: "0.0120"},
			Monthly:  hcloud.Price{Currency: "EUR", Net: "6.80", Gross: "8.16"},
		}},
	}}
	_, monthly, err := quoteCost(&hcloud.ServerType{Name: "cx32"}, pricing, "hel1", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("quoteCost: %v", err)
	}
	if monthly.Net != "7.30" || monthly.Gross != "8.76" || monthly.Currency != "EUR" {
		t.Fatalf("monthly = %#v", monthly)
	}
}

func TestQuoteCostRejectsUnparseablePrice(t *testing.T) {
	pricing := fakePricing("EUR", fakeIPv4Price("hel1", "0.0008", "0.0010", "n/a", "0.60"))
	st := fakeServerType("cx32", fakeServerPrice("hel1", "0.0100", "0.0120", "6.80", "8.16"))
	if _, _, err := quoteCost(st, pricing, "hel1", time.Unix(0, 0)); err == nil || !strings.Contains(err.Error(), "unparseable") {
		t.Fatalf("quoteCost error = %v, want unparseable price error", err)
	}
}
