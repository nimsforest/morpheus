// Package venture provides DNS template management for venture services.
// Ventures are services like "experiencenet" and "nimsforest" that customers can enable.
// Each venture has DNS record templates that define the required DNS records.
package venture

import (
	"fmt"

	"github.com/nimsforest/morpheus/pkg/dns"
)

// VentureTemplate defines DNS records needed for a venture
type VentureTemplate struct {
	Name        string           // e.g., "experiencenet", "nimsforest"
	Description string           // Human-readable description of the venture
	Records     []RecordTemplate // DNS records to create for this venture
}

// RecordTemplate defines a DNS record pattern
type RecordTemplate struct {
	Name  string         // e.g., "www", "@", "api"
	Type  dns.RecordType // A, AAAA, CNAME
	Value string         // Can use placeholders like {{.ServerIP}}
	TTL   int            // Time-to-live in seconds (0 = use default)
}

// experiencenetTemplate defines DNS records for the ExperienceNet venture
var experiencenetTemplate = VentureTemplate{
	Name:        "experiencenet",
	Description: "ExperienceNet VR streaming platform - provides immersive cloud VR experiences",
	Records: []RecordTemplate{
		{
			Name:  "@",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   300,
		},
		{
			Name:  "www",
			Type:  dns.RecordTypeCNAME,
			Value: "@",
			TTL:   300,
		},
		{
			Name:  "api",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   300,
		},
		{
			Name:  "stream",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   60,
		},
		{
			Name:  "ws",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   60,
		},
	},
}

// nimsforestTemplate defines DNS records for the NimsForest venture
var nimsforestTemplate = VentureTemplate{
	Name:        "nimsforest",
	Description: "NimsForest distributed computing platform - scalable forest infrastructure",
	Records: []RecordTemplate{
		{
			Name:  "@",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   300,
		},
		{
			Name:  "www",
			Type:  dns.RecordTypeCNAME,
			Value: "@",
			TTL:   300,
		},
		{
			Name:  "node",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   60,
		},
		{
			Name:  "registry",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   300,
		},
		{
			Name:  "metrics",
			Type:  dns.RecordTypeA,
			Value: "{{.ServerIP}}",
			TTL:   300,
		},
	},
}

// ventureTemplates holds all available venture templates
var ventureTemplates = map[string]VentureTemplate{
	"experiencenet": experiencenetTemplate,
	"nimsforest":    nimsforestTemplate,
}

// GetTemplate returns the template for a venture by name.
// Returns an error if the venture template is not found.
func GetTemplate(ventureName string) (*VentureTemplate, error) {
	template, ok := ventureTemplates[ventureName]
	if !ok {
		available := make([]string, 0, len(ventureTemplates))
		for name := range ventureTemplates {
			available = append(available, name)
		}
		return nil, fmt.Errorf("venture template %q not found, available ventures: %v", ventureName, available)
	}
	return &template, nil
}

// ListTemplates returns all available venture templates
func ListTemplates() []VentureTemplate {
	templates := make([]VentureTemplate, 0, len(ventureTemplates))
	for _, template := range ventureTemplates {
		templates = append(templates, template)
	}
	return templates
}

// ListVentureNames returns all available venture names
func ListVentureNames() []string {
	names := make([]string, 0, len(ventureTemplates))
	for name := range ventureTemplates {
		names = append(names, name)
	}
	return names
}
