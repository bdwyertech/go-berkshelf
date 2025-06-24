package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

func TestSupermarketSource_ListVersions(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/cookbooks/nginx" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		response := cookbookResponse{
			Name:          "nginx",
			LatestVersion: "2.7.6",
			Versions: []string{
				"2.7.6",
				"2.7.4",
				"2.7.2",
				"2.6.0",
				"2.5.0",
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	source := NewSupermarketSource(server.URL)
	versions, err := source.ListVersions(context.Background(), "nginx")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	if len(versions) != 5 {
		t.Errorf("ListVersions() returned %d versions, want 5", len(versions))
	}

	// Check first version
	if versions[0].String() != "2.7.6" {
		t.Errorf("First version = %s, want 2.7.6", versions[0].String())
	}
}

func TestSupermarketSource_ListVersions_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	source := NewSupermarketSource(server.URL)
	_, err := source.ListVersions(context.Background(), "nonexistent")
	if err == nil {
		t.Error("ListVersions() should return error for non-existent cookbook")
	}

	if _, ok := err.(*ErrCookbookNotFound); !ok {
		t.Errorf("ListVersions() error = %v, want ErrCookbookNotFound", err)
	}
}

func TestSupermarketSource_FetchMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/cookbooks/nginx/versions/2.7.6" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		response := cookbookVersionResponse{
			Version:    "2.7.6",
			TarballURL: "https://example.com/nginx-2.7.6.tar.gz",
			Dependencies: map[string]string{
				"apt":             "~> 2.2",
				"build-essential": "~> 2.0",
				"yum-epel":        "~> 0.3",
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	source := NewSupermarketSource(server.URL)
	version, _ := berkshelf.NewVersion("2.7.6")
	metadata, err := source.FetchMetadata(context.Background(), "nginx", version)
	if err != nil {
		t.Fatalf("FetchMetadata() error = %v", err)
	}

	if metadata.Name != "nginx" {
		t.Errorf("FetchMetadata() Name = %s, want nginx", metadata.Name)
	}

	if metadata.Version.String() != "2.7.6" {
		t.Errorf("FetchMetadata() Version = %s, want 2.7.6", metadata.Version.String())
	}

	if len(metadata.Dependencies) != 3 {
		t.Errorf("FetchMetadata() Dependencies = %d, want 3", len(metadata.Dependencies))
	}

	// Check a dependency
	if aptDep, ok := metadata.Dependencies["apt"]; ok {
		if aptDep.String() != "~> 2.2" {
			t.Errorf("apt dependency = %s, want ~> 2.2", aptDep.String())
		}
	} else {
		t.Error("apt dependency not found")
	}
}

func TestSupermarketSource_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query().Get("q")
		if query != "nginx" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := struct {
			Items []cookbookResponse `json:"items"`
			Total int                `json:"total"`
		}{
			Items: []cookbookResponse{
				{
					Name:          "nginx",
					LatestVersion: "2.7.6",
					Description:   "Installs and configures nginx",
				},
				{
					Name:          "nginx-proxy",
					LatestVersion: "1.0.0",
					Description:   "Nginx proxy configuration",
				},
			},
			Total: 2,
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	source := NewSupermarketSource(server.URL)
	results, err := source.Search(context.Background(), "nginx")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Search() returned %d results, want 2", len(results))
	}

	if results[0].Name != "nginx" {
		t.Errorf("First result name = %s, want nginx", results[0].Name)
	}
}

func TestSupermarketSource_Priority(t *testing.T) {
	source := NewSupermarketSource("https://supermarket.chef.io")

	if source.Priority() != 100 {
		t.Errorf("Priority() = %d, want 100", source.Priority())
	}

	source.SetPriority(50)
	if source.Priority() != 50 {
		t.Errorf("Priority() after SetPriority = %d, want 50", source.Priority())
	}
}

func TestSupermarketSource_Name(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://supermarket.chef.io", "supermarket (https://supermarket.chef.io)"},
		{"https://internal.example.com", "supermarket (https://internal.example.com)"},
		{"", "supermarket (https://supermarket.chef.io)"},
	}

	for _, tt := range tests {
		source := NewSupermarketSource(tt.url)
		if name := source.Name(); name != tt.expected {
			t.Errorf("Name() = %s, want %s", name, tt.expected)
		}
	}
}
