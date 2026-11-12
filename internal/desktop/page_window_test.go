package desktop

import "testing"

func TestPageWindowKeepsIdentityAcrossSettingsNavigation(t *testing.T) {
	first, err := ResolvePageWindow(PageWindowRequest{Page: "settings", Section: "appearance"})
	if err != nil {
		t.Fatal(err)
	}
	next, err := ResolvePageWindow(PageWindowRequest{Page: "settings", Section: "shortcuts"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != next.Name || next.Query.Get("section") != "shortcuts" {
		t.Fatalf("settings navigation must reuse its window and retain the requested section: %+v", next)
	}
}

func TestConversationWindowsHaveIndependentIdentity(t *testing.T) {
	first, err := ResolvePageWindow(PageWindowRequest{Page: "ai", ConversationID: 12})
	if err != nil {
		t.Fatal(err)
	}
	next, err := ResolvePageWindow(PageWindowRequest{Page: "ai", ConversationID: 13})
	if err != nil {
		t.Fatal(err)
	}
	if first.Name == next.Name || first.Query.Get("conversation") != "12" {
		t.Fatalf("different conversations must not share a window: %+v / %+v", first, next)
	}
}

func TestPageWindowRejectsInvalidNavigation(t *testing.T) {
	for _, request := range []PageWindowRequest{
		{Page: "unknown"}, {Page: "settings", Section: "unknown"},
		{Page: "ai", ConversationID: -1}, {Page: "audit", ConversationID: 2},
		{Page: "audit", Section: "appearance"}, {Page: "ai", ExtensionName: "driver"},
	} {
		if _, err := ResolvePageWindow(request); err == nil {
			t.Errorf("accepted invalid navigation: %+v", request)
		}
	}
}
