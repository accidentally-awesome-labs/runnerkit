package hetzner

import (
	"context"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

// primaryIPv4Type is the Primary IP type name in GET /pricing.
const primaryIPv4Type = "ipv4"

// apiDecimal is the shape of a Hetzner API price string ("4.4900000000").
// big.Rat.SetString alone would also accept fractions ("1/3"), exponents and
// signs, which the API never sends and which could not be shown exactly.
var apiDecimal = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// quote fetches the live price list and prices the planned server plus its
// auto-allocated primary IPv4 in the profile's location. serverType may be
// nil, in which case it is looked up by name.
func (p *Provider) quote(ctx context.Context, client Client, serverType *hcloud.ServerType, profile provider.Profile) (hourly, monthly *provider.CostEstimate, err error) {
	if serverType == nil {
		serverType, err = client.GetServerType(ctx, profile.ServerType)
		if err != nil {
			return nil, nil, fmt.Errorf("read Hetzner server type %s pricing: %w", profile.ServerType, err)
		}
		if serverType == nil {
			return nil, nil, fmt.Errorf("Hetzner server type %s is unavailable; choose --cloud-profile cpx22 or another supported profile", profile.ServerType)
		}
	}
	pricing, err := client.GetPricing(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("read Hetzner pricing: %w", err)
	}
	return quoteCost(serverType, pricing, profile.Region, p.now())
}

func (p *Provider) now() time.Time {
	if p.Now != nil {
		return p.Now().UTC()
	}
	return time.Now().UTC()
}

// quoteCost computes the hourly and monthly estimate for serverType in
// location from Hetzner API data only: the server type's per-location price
// plus the location's primary IPv4 price. Net and gross are summed exactly
// (decimal, not float) from the API strings. A missing price for either
// resource returns *provider.UnpricedLocationError; there is no fallback.
func quoteCost(serverType *hcloud.ServerType, pricing hcloud.Pricing, location string, fetchedAt time.Time) (hourly, monthly *provider.CostEstimate, err error) {
	typeName := ""
	if serverType != nil {
		typeName = serverType.Name
	}
	server, ok := serverTypeLocationPrice(serverType, pricing, location)
	if !ok {
		return nil, nil, &provider.UnpricedLocationError{ServerType: typeName, Location: location, Resource: "server"}
	}
	ipv4, ok := primaryIPv4LocationPrice(pricing, location)
	if !ok {
		return nil, nil, &provider.UnpricedLocationError{ServerType: typeName, Location: location, Resource: "primary_ipv4"}
	}
	currency := strings.TrimSpace(server.Monthly.Currency)
	if currency == "" {
		currency = pricingCurrency(pricing)
	}
	if currency == "" {
		return nil, nil, fmt.Errorf("Hetzner pricing response carried no currency")
	}
	serverResource := "server:" + typeName
	hourly, err = sumCost("hour", currency, fetchedAt,
		provider.CostComponent{Resource: serverResource, Net: server.Hourly.Net, Gross: server.Hourly.Gross},
		provider.CostComponent{Resource: "primary_ipv4", Net: ipv4.Hourly.Net, Gross: ipv4.Hourly.Gross},
	)
	if err != nil {
		return nil, nil, err
	}
	monthly, err = sumCost("month", currency, fetchedAt,
		provider.CostComponent{Resource: serverResource, Net: server.Monthly.Net, Gross: server.Monthly.Gross},
		provider.CostComponent{Resource: "primary_ipv4", Net: ipv4.Monthly.Net, Gross: ipv4.Monthly.Gross},
	)
	if err != nil {
		return nil, nil, err
	}
	return hourly, monthly, nil
}

// serverTypeLocationPrice returns the server type's price in location. The
// GET /server_types entry is authoritative; the GET /pricing server type
// list is consulted when that entry carries no price for the location.
func serverTypeLocationPrice(serverType *hcloud.ServerType, pricing hcloud.Pricing, location string) (hcloud.ServerTypeLocationPricing, bool) {
	if serverType == nil {
		return hcloud.ServerTypeLocationPricing{}, false
	}
	for _, entry := range serverType.Pricings {
		if entry.Location != nil && entry.Location.Name == location && complete(entry.Hourly.Net, entry.Hourly.Gross, entry.Monthly.Net, entry.Monthly.Gross) {
			return entry, true
		}
	}
	for _, st := range pricing.ServerTypes {
		if st.ServerType == nil || st.ServerType.Name != serverType.Name {
			continue
		}
		for _, entry := range st.Pricings {
			if entry.Location != nil && entry.Location.Name == location && complete(entry.Hourly.Net, entry.Hourly.Gross, entry.Monthly.Net, entry.Monthly.Gross) {
				return entry, true
			}
		}
	}
	return hcloud.ServerTypeLocationPricing{}, false
}

func primaryIPv4LocationPrice(pricing hcloud.Pricing, location string) (hcloud.PrimaryIPTypePricing, bool) {
	for _, ipType := range pricing.PrimaryIPs {
		if ipType.Type != primaryIPv4Type {
			continue
		}
		for _, entry := range ipType.Pricings {
			if entry.Location == location && complete(entry.Hourly.Net, entry.Hourly.Gross, entry.Monthly.Net, entry.Monthly.Gross) {
				return entry, true
			}
		}
	}
	return hcloud.PrimaryIPTypePricing{}, false
}

// pricingCurrency returns the price list's currency. hcloud-go copies the
// response's top-level currency into every Price it builds from GET /pricing.
func pricingCurrency(pricing hcloud.Pricing) string {
	for _, st := range pricing.ServerTypes {
		for _, entry := range st.Pricings {
			if c := strings.TrimSpace(entry.Monthly.Currency); c != "" {
				return c
			}
		}
	}
	return strings.TrimSpace(pricing.Image.PerGBMonth.Currency)
}

func complete(values ...string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}
	return true
}

// sumCost adds the components' net and gross prices exactly, keeping the
// API's decimal precision.
func sumCost(period, currency string, fetchedAt time.Time, components ...provider.CostComponent) (*provider.CostEstimate, error) {
	net, gross := new(big.Rat), new(big.Rat)
	digits := 0
	for _, c := range components {
		for _, field := range []struct {
			kind  string
			value string
			sum   *big.Rat
		}{{"net", c.Net, net}, {"gross", c.Gross, gross}} {
			value := strings.TrimSpace(field.value)
			parsed, ok := new(big.Rat).SetString(value)
			if !ok || !apiDecimal.MatchString(value) {
				return nil, fmt.Errorf("Hetzner pricing API returned an unparseable %s %s price %q for %s", period, field.kind, field.value, c.Resource)
			}
			field.sum.Add(field.sum, parsed)
			if dot := strings.IndexByte(value, '.'); dot >= 0 && len(value)-dot-1 > digits {
				digits = len(value) - dot - 1
			}
		}
	}
	return &provider.CostEstimate{
		Amount:     gross.FloatString(digits),
		Net:        net.FloatString(digits),
		Gross:      gross.FloatString(digits),
		Currency:   currency,
		Period:     period,
		Source:     provider.CostSourceHetznerAPI,
		FetchedAt:  fetchedAt,
		Components: components,
	}, nil
}
