package core

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Locale       string    `json:"locale"`
	Theme        string    `json:"theme"`
	IsActive     bool      `json:"is_active"`
	PasswordHash string    `json:"-"`
	FeishuOpenID string    `json:"feishu_open_id,omitempty"`
	IsAdmin      bool      `json:"is_admin,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Group struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	MemberIDs []string `json:"member_ids,omitempty"`
}

// IdentityProvider is deliberately provider-neutral.  Each office suite can
// supply its OAuth endpoints and organisation-sync adapter without changing
// the user, group, or permission model.
type IdentityProvider struct {
	ID               string            `json:"id"`
	Type             string            `json:"type"` // feishu | dingtalk | wecom | oidc
	Name             string            `json:"name"`
	Enabled          bool              `json:"enabled"`
	ClientID         string            `json:"client_id"`
	ClientSecret     string            `json:"client_secret,omitempty"`
	AuthorizationURL string            `json:"authorization_url"`
	TokenURL         string            `json:"token_url"`
	Scopes           []string          `json:"scopes,omitempty"`
	GroupMappings    map[string]string `json:"group_mappings,omitempty"` // external group ID -> Topbase group ID
}

// AuthSettings governs which sign-in methods are available to end users.
// Password sign-in is on by default so an administrator cannot be locked out
// before a third-party provider has been configured.
type AuthSettings struct {
	PasswordLoginEnabled bool `json:"password_login_enabled"`
}

type ExternalIdentityLink struct {
	ProviderID string `json:"provider_id"`
	Subject    string `json:"subject"`
	UserID     string `json:"user_id"`
}

type WebhookEndpoint struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"` // feishu | dingtalk | wecom | generic
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminSettings contains instance-wide switches that must be enforced by the
// HTTP layer, rather than merely hidden in the UI.
type AdminSettings struct {
	SiteName             string `json:"site_name"`
	Timezone             string `json:"timezone"`
	PublicSharingEnabled bool   `json:"public_sharing_enabled"`
	EmbeddingEnabled     bool   `json:"embedding_enabled"`
}

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

type SetupRequest struct {
	Language      string `json:"language"`
	AdminName     string `json:"admin_name"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	SiteName      string `json:"site_name"`
}
