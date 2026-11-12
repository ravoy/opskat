package desktop

import (
	"fmt"
	"net/url"
	"strconv"
)

// PageWindowRequest identifies a dedicated tool window and optional navigation.
type PageWindowRequest struct {
	Page           string `json:"page"`
	Section        string `json:"section"`
	ConversationID int64  `json:"conversationID"`
	ExtensionName  string `json:"extensionName"`
}

type PageWindowSpec struct {
	Name  string
	Query url.Values
}

var pageWindowNames = map[string]string{
	"settings": "page-settings", "sshkeys": "page-sshkeys", "audit": "page-audit",
	"forward": "page-forward", "snippets": "page-snippets", "ai": "page-ai",
}

var settingsSections = map[string]bool{
	"ai": true, "import": true, "backup": true, "shortcuts": true,
	"terminal": true, "network": true, "external-edit": true, "appearance": true,
	"custom-types": true, "status": true, "extensions": true, "about": true, "debug": true,
}

func ResolvePageWindow(request PageWindowRequest) (PageWindowSpec, error) {
	name, ok := pageWindowNames[request.Page]
	if !ok {
		return PageWindowSpec{}, fmt.Errorf("unsupported window page %q", request.Page)
	}
	if request.ConversationID < 0 || (request.ConversationID != 0 && request.Page != "ai") {
		return PageWindowSpec{}, fmt.Errorf("conversation ID is only valid for AI windows and must not be negative")
	}
	if request.Section != "" && (request.Page != "settings" || !settingsSections[request.Section]) {
		return PageWindowSpec{}, fmt.Errorf("invalid settings section %q", request.Section)
	}
	if request.ExtensionName != "" && (request.Page != "settings" || request.Section != "extensions") {
		return PageWindowSpec{}, fmt.Errorf("extension navigation requires the extensions settings section")
	}
	query := url.Values{"page": {request.Page}}
	if request.Page == "ai" {
		id := strconv.FormatInt(request.ConversationID, 10)
		name += "-" + id
		query.Set("conversation", id)
	}
	query.Set("workspace", name)
	if request.Section != "" {
		query.Set("section", request.Section)
	}
	if request.ExtensionName != "" {
		query.Set("extension", request.ExtensionName)
	}
	return PageWindowSpec{Name: name, Query: query}, nil
}
