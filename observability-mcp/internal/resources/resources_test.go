package resources

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResourcesDiscoveryAndRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "observability-mcp",
			Version: "1.0.0",
		},
		nil,
	)
	RegisterResources(server)

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		nil,
	)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	go func() {
		_, _ = server.Connect(ctx, serverTransport, nil)
	}()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer clientSession.Close()

	// 1. Test ListResources
	listRes, err := clientSession.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	expectedURIs := map[string]bool{
		"service://orderflow":              false,
		"service://orderflow/dependencies": false,
		"service://orderflow/endpoints":    false,
		"service://orderflow/deployments":  false,
		"runbook://orderflow":              false,
		"runbook://redis":                  false,
		"runbook://postgresql":             false,
		"runbook://payment":                false,
	}

	for _, r := range listRes.Resources {
		if _, ok := expectedURIs[r.URI]; ok {
			expectedURIs[r.URI] = true
		}
	}

	for uri, found := range expectedURIs {
		if !found {
			t.Errorf("expected resource %s was not discovered in ListResources", uri)
		}
	}

	// 2. Test Reading Service Resources (JSON)
	jsonURIs := []string{
		"service://orderflow",
		"service://orderflow/dependencies",
		"service://orderflow/endpoints",
		"service://orderflow/deployments",
	}

	for _, uri := range jsonURIs {
		t.Run("read_"+uri, func(t *testing.T) {
			res, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
			if err != nil {
				t.Fatalf("ReadResource(%s) failed: %v", uri, err)
			}
			if len(res.Contents) == 0 {
				t.Fatalf("ReadResource(%s) returned empty contents", uri)
			}
			content := res.Contents[0]
			if content.MIMEType != "application/json" {
				t.Errorf("expected MIMEType application/json, got %s", content.MIMEType)
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(content.Text), &parsed); err != nil {
				t.Fatalf("ReadResource(%s) content is not valid JSON: %v", uri, err)
			}
		})
	}

	// 3. Test Reading Runbook Resources (Markdown)
	runbookURIs := []string{
		"runbook://orderflow",
		"runbook://redis",
		"runbook://postgresql",
		"runbook://payment",
	}

	for _, uri := range runbookURIs {
		t.Run("read_"+uri, func(t *testing.T) {
			res, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
			if err != nil {
				t.Fatalf("ReadResource(%s) failed: %v", uri, err)
			}
			if len(res.Contents) == 0 {
				t.Fatalf("ReadResource(%s) returned empty contents", uri)
			}
			content := res.Contents[0]
			if content.MIMEType != "text/markdown" {
				t.Errorf("expected MIMEType text/markdown, got %s", content.MIMEType)
			}
			if !strings.Contains(content.Text, "Runbook") {
				t.Errorf("expected runbook heading in content for %s, got: %s", uri, content.Text)
			}
		})
	}
}
