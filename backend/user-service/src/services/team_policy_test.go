package services

import (
	"testing"

	"github.com/pos/user-service/src/models"
)

func TestCanManageTeamMember(t *testing.T) {
	tests := []struct {
		name      string
		actorID   string
		actorRole string
		target    *models.User
		want      bool
	}{
		{
			name:      "owner can manage manager",
			actorID:   "owner-1",
			actorRole: string(models.RoleOwner),
			target:    &models.User{ID: "manager-1", Role: string(models.RoleManager)},
			want:      true,
		},
		{
			name:      "owner can manage cashier",
			actorID:   "owner-1",
			actorRole: string(models.RoleOwner),
			target:    &models.User{ID: "cashier-1", Role: string(models.RoleCashier)},
			want:      true,
		},
		{
			name:      "manager can manage cashier",
			actorID:   "manager-1",
			actorRole: string(models.RoleManager),
			target:    &models.User{ID: "cashier-1", Role: string(models.RoleCashier)},
			want:      true,
		},
		{
			name:      "manager cannot manage manager",
			actorID:   "manager-1",
			actorRole: string(models.RoleManager),
			target:    &models.User{ID: "manager-2", Role: string(models.RoleManager)},
			want:      false,
		},
		{
			name:      "manager cannot manage owner",
			actorID:   "manager-1",
			actorRole: string(models.RoleManager),
			target:    &models.User{ID: "owner-1", Role: string(models.RoleOwner)},
			want:      false,
		},
		{
			name:      "self management is denied",
			actorID:   "cashier-1",
			actorRole: string(models.RoleManager),
			target:    &models.User{ID: "cashier-1", Role: string(models.RoleCashier)},
			want:      false,
		},
		{
			name:      "cashier cannot manage cashier",
			actorID:   "cashier-1",
			actorRole: string(models.RoleCashier),
			target:    &models.User{ID: "cashier-2", Role: string(models.RoleCashier)},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canManageTeamMember(tt.actorID, tt.actorRole, tt.target); got != tt.want {
				t.Fatalf("canManageTeamMember() = %v, want %v", got, tt.want)
			}
		})
	}
}
