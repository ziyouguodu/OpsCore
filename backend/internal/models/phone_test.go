package models

import (
	"encoding/json"
	"testing"
)

func TestUserMutationPhoneDistinguishesOmittedAndCleared(t *testing.T) {
	for _, test := range []struct {
		body     string
		nilPhone bool
		phone    string
	}{
		{body: `{}`, nilPhone: true},
		{body: `{"phone":null}`, nilPhone: true},
		{body: `{"phone":""}`},
		{body: `{"phone":"+86 138-0013-8000"}`, phone: "+86 138-0013-8000"},
	} {
		var mutation UserMutation
		if err := json.Unmarshal([]byte(test.body), &mutation); err != nil {
			t.Fatal(err)
		}
		if (mutation.Phone == nil) != test.nilPhone || mutation.Phone != nil && *mutation.Phone != test.phone {
			t.Fatalf("unexpected phone for %s: %+v", test.body, mutation.Phone)
		}
	}
}

func TestPhoneOnlyAppearsInManagementAndDutyModels(t *testing.T) {
	for _, test := range []struct {
		name      string
		value     any
		wantPhone bool
	}{
		{name: "authentication", value: User{ID: 1}},
		{name: "directory", value: UserDirectoryItem{ID: 1}},
		{name: "management", value: UserListItem{ID: 1}, wantPhone: true},
		{name: "duty member", value: DutyMember{UserID: 1}, wantPhone: true},
		{name: "current person", value: DutyCurrentPerson{}, wantPhone: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			var object map[string]any
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			if _, ok := object["phone"]; ok != test.wantPhone {
				t.Fatalf("unexpected phone visibility: %s", raw)
			}
		})
	}
}
