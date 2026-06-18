package auth

import (
	"testing"
	"time"
)

func TestPasswordHashVerifiesOriginalPasswordOnly(t *testing.T) {
	hash, err := HashPassword("ChangeMe123!")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !VerifyPassword(hash, "ChangeMe123!") {
		t.Fatalf("expected original password to verify")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatalf("expected wrong password to fail")
	}
}

func TestSignerRejectsTamperedToken(t *testing.T) {
	signer := NewSigner("secret", time.Hour)
	token, err := signer.Issue(1, "admin", []string{RoleSuperAdmin})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	if _, err := signer.Verify(token + "x"); err == nil {
		t.Fatalf("expected tampered token to fail")
	}
}

func TestOpsEngineerPermissionsAreLimitedToPhaseOneOperations(t *testing.T) {
	roles := []string{RoleOpsEngineer}
	allowed := []string{
		PermissionAssetRead,
		PermissionAssetWrite,
		PermissionOnCallRead,
		PermissionTaskRead,
		PermissionTaskWrite,
		PermissionIncidentRead,
		PermissionIncidentFollowup,
	}
	for _, permission := range allowed {
		if !HasPermission(roles, permission) {
			t.Fatalf("expected ops engineer to have %s", permission)
		}
	}
	if HasPermission(roles, PermissionAssetCredential) {
		t.Fatalf("ops engineer must not see asset credentials by default")
	}
	if HasPermission(roles, PermissionAssetCredentialWrite) {
		t.Fatalf("ops engineer must not edit asset credentials by default")
	}
	if HasPermission(roles, PermissionOnCallWrite) {
		t.Fatalf("ops engineer must not modify oncall schedules in phase one")
	}
	if HasPermission(roles, PermissionUserManage) {
		t.Fatalf("ops engineer must not manage users")
	}
}
