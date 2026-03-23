package authz_test

import (
	"net/http"
	"testing"

	"backend/internal/authz"
)

func TestMapEndpointToPermission_CRMDeals(t *testing.T) {
	base := "/api/v1/workspaces/ws-1"
	t.Run("GET list deals", func(t *testing.T) {
		obj, act := authz.MapEndpointToPermission(http.MethodGet, base+"/crm/deals")
		if obj != "crm:deal" || act != "read" {
			t.Fatalf("expected crm:deal read, got %q %q", obj, act)
		}
	})
	t.Run("POST create deal", func(t *testing.T) {
		obj, act := authz.MapEndpointToPermission(http.MethodPost, base+"/crm/deals")
		if obj != "crm:deal" || act != "create" {
			t.Fatalf("expected crm:deal create, got %q %q", obj, act)
		}
	})
	t.Run("PUT deal by id", func(t *testing.T) {
		obj, act := authz.MapEndpointToPermission(http.MethodPut, base+"/crm/deals/550e8400-e29b-41d4-a716-446655440000")
		if obj != "crm:deal" || act != "update" {
			t.Fatalf("expected crm:deal update, got %q %q", obj, act)
		}
	})
}

func TestMapEndpointToPermission_WorkspaceMembers(t *testing.T) {
	base := "/api/v1/workspaces/ws-1"
	obj, act := authz.MapEndpointToPermission(http.MethodGet, base+"/members")
	if obj != "workspace:member" || act != "invite" {
		t.Fatalf("expected workspace:member invite for GET members, got %q %q", obj, act)
	}
}

func TestMapEndpointToPermission_Tasks(t *testing.T) {
	base := "/api/v1/workspaces/ws-1"
	obj, act := authz.MapEndpointToPermission(http.MethodGet, base+"/tasks")
	if obj != "tasks:task" || act != "read" {
		t.Fatalf("expected tasks:task read, got %q %q", obj, act)
	}
}
