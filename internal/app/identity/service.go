package identity

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

type Service struct {
	Users    core.UserStore
	Groups   core.GroupStore
	Sessions core.SessionStore
	Settings core.SettingStore
	APIKeys  core.APIKeyStore
}

func (s Service) SetupCompleted() (bool, error) {
	value, ok, err := s.Settings.Get("setup_completed")
	if err != nil {
		return false, err
	}
	return ok && value == "true", nil
}

func (s Service) CompleteSetup(input core.SetupRequest) (core.User, error) {
	done, err := s.SetupCompleted()
	if err != nil {
		return core.User{}, err
	}
	if done {
		return core.User{}, fmt.Errorf("setup is already completed")
	}
	email := strings.ToLower(strings.TrimSpace(input.AdminEmail))
	name := strings.TrimSpace(input.AdminName)
	if email == "" || name == "" {
		return core.User{}, fmt.Errorf("admin name and email are required")
	}
	hash, err := core.HashPassword(input.AdminPassword)
	if err != nil {
		return core.User{}, err
	}
	now := time.Now().UTC()
	user := core.User{
		ID: core.NewID("usr"), Email: email, Name: name, Locale: localeOrDefault(input.Language),
		Theme: "dark", IsActive: true, PasswordHash: hash, CreatedAt: now,
	}
	if err := s.Users.Create(user); err != nil {
		return core.User{}, err
	}
	allUsers := core.Group{ID: "grp_all_users", Name: "所有人", Kind: "all_users"}
	admins := core.Group{ID: "grp_admins", Name: "管理员", Kind: "admins"}
	if err := s.Groups.Create(allUsers); err != nil {
		return core.User{}, err
	}
	if err := s.Groups.Create(admins); err != nil {
		return core.User{}, err
	}
	if err := s.Groups.AddMember(allUsers.ID, user.ID); err != nil {
		return core.User{}, err
	}
	if err := s.Groups.AddMember(admins.ID, user.ID); err != nil {
		return core.User{}, err
	}
	siteName := strings.TrimSpace(input.SiteName)
	if siteName == "" {
		siteName = "Topbase"
	}
	if err := s.Settings.Set("site_name", siteName); err != nil {
		return core.User{}, err
	}
	if err := s.Settings.Set("setup_completed", "true"); err != nil {
		return core.User{}, err
	}
	return user, nil
}

func (s Service) Login(email, password string) (core.Session, core.User, error) {
	settings, err := s.AuthSettings()
	if err != nil {
		return core.Session{}, core.User{}, err
	}
	if !settings.PasswordLoginEnabled {
		return core.Session{}, core.User{}, fmt.Errorf("password login is disabled by administrator")
	}
	user, err := s.Users.ByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return core.Session{}, core.User{}, fmt.Errorf("invalid email or password")
	}
	if !user.IsActive || !core.VerifyPassword(user.PasswordHash, password) {
		return core.Session{}, core.User{}, fmt.Errorf("invalid email or password")
	}
	session := core.Session{ID: core.NewID("ses"), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour)}
	if err := s.Sessions.Create(session); err != nil {
		return core.Session{}, core.User{}, err
	}
	return session, user, nil
}

func (s Service) LoginExternal(email string) (core.Session, core.User, error) {
	user, err := s.Users.ByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil || !user.IsActive {
		return core.Session{}, core.User{}, fmt.Errorf("this third-party account is not linked to an active member")
	}
	session := core.Session{ID: core.NewID("ses"), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour)}
	if err := s.Sessions.Create(session); err != nil {
		return core.Session{}, core.User{}, err
	}
	return session, user, nil
}

func (s Service) LoginExternalIdentity(providerID, subject, email string) (core.Session, core.User, error) {
	links, err := s.ExternalIdentityLinks()
	if err != nil {
		return core.Session{}, core.User{}, err
	}
	for _, link := range links {
		if link.ProviderID == providerID && link.Subject == subject {
			user, err := s.Users.ByID(link.UserID)
			if err != nil || !user.IsActive {
				return core.Session{}, core.User{}, fmt.Errorf("this third-party account is not linked to an active member")
			}
			session := core.Session{ID: core.NewID("ses"), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour)}
			if err := s.Sessions.Create(session); err != nil {
				return core.Session{}, core.User{}, err
			}
			return session, user, nil
		}
	}
	if email != "" {
		return s.LoginExternal(email)
	}
	return core.Session{}, core.User{}, fmt.Errorf("this third-party account has not been bound by an administrator")
}

func (s Service) UserForSession(sessionID string) (core.User, error) {
	session, err := s.Sessions.ByID(sessionID)
	if err != nil {
		return core.User{}, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.Sessions.Delete(sessionID)
		return core.User{}, fmt.Errorf("session expired")
	}
	return s.Users.ByID(session.UserID)
}

func (s Service) Logout(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.Sessions.Delete(sessionID)
}

func localeOrDefault(language string) string {
	if strings.HasPrefix(strings.ToLower(language), "en") {
		return "en"
	}
	return "zh-CN"
}

func (s Service) ListUsers() ([]core.User, error) {
	return s.Users.List()
}

func (s Service) CreateGroup(name string) (core.Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return core.Group{}, fmt.Errorf("group name is required")
	}
	group := core.Group{ID: core.NewID("grp"), Name: name, Kind: "manual"}
	if err := s.Groups.Create(group); err != nil {
		return core.Group{}, err
	}
	return group, nil
}

func (s Service) ReplaceGroupMembers(id string, memberIDs []string) (core.Group, error) {
	groups, err := s.Groups.List()
	if err != nil {
		return core.Group{}, err
	}
	var group core.Group
	found := false
	for _, item := range groups {
		if item.ID == id {
			group = item
			found = true
			break
		}
	}
	if !found {
		return core.Group{}, core.ErrNotFound
	}
	if group.Kind != "manual" {
		return core.Group{}, fmt.Errorf("system or synchronized groups cannot be edited manually")
	}
	seen := map[string]bool{}
	valid := []string{}
	for _, id := range memberIDs {
		if id == "" || seen[id] {
			continue
		}
		if _, err := s.Users.ByID(id); err != nil {
			return core.Group{}, fmt.Errorf("member not found")
		}
		seen[id] = true
		valid = append(valid, id)
	}
	if err := s.Groups.ReplaceMembers(group.ID, valid); err != nil {
		return core.Group{}, err
	}
	group.MemberIDs = valid
	return group, nil
}

func (s Service) InviteUser(name, email, password string) (core.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if email == "" || name == "" {
		return core.User{}, fmt.Errorf("name and email are required")
	}
	if password == "" {
		password = core.NewID("tmp") + "A1"
	}
	hash, err := core.HashPassword(password)
	if err != nil {
		return core.User{}, err
	}
	user := core.User{
		ID: core.NewID("usr"), Email: email, Name: name, Locale: "zh-CN",
		Theme: "dark", IsActive: true, PasswordHash: hash, CreatedAt: time.Now().UTC(),
	}
	if err := s.Users.Create(user); err != nil {
		return core.User{}, err
	}
	if s.Groups != nil {
		_ = s.Groups.AddMember("grp_all_users", user.ID)
	}
	return user, nil
}

func (s Service) SetUserActive(id string, active bool) error {
	return s.Users.SetActive(id, active)
}

func (s Service) ResetUserPassword(id, password string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password is required")
	}
	hash, err := core.HashPassword(password)
	if err != nil {
		return err
	}
	return s.Users.SetPassword(id, hash)
}

func (s Service) UpdateProfile(userID, name, email, locale, theme, avatarURL, currentPassword string) (core.User, error) {
	user, err := s.Users.ByID(userID)
	if err != nil {
		return core.User{}, err
	}
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	locale = strings.TrimSpace(locale)
	theme = strings.TrimSpace(theme)
	if name == "" || email == "" {
		return core.User{}, fmt.Errorf("name and email are required")
	}
	if locale != "zh-CN" && locale != "en" {
		return core.User{}, fmt.Errorf("unsupported language")
	}
	if theme != "system" && theme != "light" && theme != "dark" {
		return core.User{}, fmt.Errorf("unsupported theme")
	}
	if err := validateAvatar(avatarURL); err != nil {
		return core.User{}, err
	}
	if email != user.Email && (user.PasswordHash == "" || !core.VerifyPassword(user.PasswordHash, currentPassword)) {
		return core.User{}, fmt.Errorf("current password is required to change email")
	}
	if existing, lookupErr := s.Users.ByEmail(email); lookupErr == nil && existing.ID != userID {
		return core.User{}, fmt.Errorf("email is already in use")
	} else if lookupErr != nil && !errors.Is(lookupErr, core.ErrNotFound) {
		return core.User{}, lookupErr
	}
	if err := s.Users.UpdateProfile(userID, name, email, locale, theme, avatarURL); err != nil {
		return core.User{}, err
	}
	user.Name, user.Email, user.Locale, user.Theme, user.AvatarURL = name, email, locale, theme, avatarURL
	return s.WithRole(user), nil
}

func validateAvatar(value string) error {
	if value == "" {
		return nil
	}
	if len(value) > 350_000 {
		return fmt.Errorf("avatar image is too large")
	}
	allowed := []string{"data:image/png;base64,", "data:image/jpeg;base64,", "data:image/webp;base64,"}
	raw := ""
	for _, prefix := range allowed {
		if strings.HasPrefix(value, prefix) {
			raw = strings.TrimPrefix(value, prefix)
			break
		}
	}
	if raw == "" {
		return fmt.Errorf("avatar must be a PNG, JPEG or WebP image")
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(decoded) == 0 || len(decoded) > 256*1024 {
		return fmt.Errorf("avatar image is invalid or too large")
	}
	isPNG := len(decoded) >= 8 && string(decoded[:8]) == "\x89PNG\r\n\x1a\n"
	isJPEG := len(decoded) >= 3 && decoded[0] == 0xff && decoded[1] == 0xd8 && decoded[2] == 0xff
	isWebP := len(decoded) >= 12 && string(decoded[:4]) == "RIFF" && string(decoded[8:12]) == "WEBP"
	if !isPNG && !isJPEG && !isWebP {
		return fmt.Errorf("avatar image is invalid or too large")
	}
	return nil
}

func (s Service) ChangePassword(userID, currentPassword, newPassword string) error {
	user, err := s.Users.ByID(userID)
	if err != nil {
		return err
	}
	if user.PasswordHash == "" || !core.VerifyPassword(user.PasswordHash, currentPassword) {
		return fmt.Errorf("current password is incorrect")
	}
	if currentPassword == newPassword {
		return fmt.Errorf("new password must be different")
	}
	hash, err := core.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.Users.SetPassword(userID, hash)
}

// ReplaceUserManualGroups updates only manually managed groups. System and
// synchronized memberships remain owned by their respective sources.
func (s Service) ReplaceUserManualGroups(userID string, groupIDs []string) error {
	if _, err := s.Users.ByID(userID); err != nil {
		return err
	}
	groups, err := s.ListGroups()
	if err != nil {
		return err
	}
	manualGroups := map[string]bool{}
	for _, group := range groups {
		if group.Kind == "manual" {
			manualGroups[group.ID] = true
		}
	}
	desired := map[string]bool{}
	for _, groupID := range groupIDs {
		if !manualGroups[groupID] {
			return fmt.Errorf("group not found or cannot be edited manually")
		}
		desired[groupID] = true
	}
	for _, group := range groups {
		if group.Kind != "manual" {
			continue
		}
		members := make([]string, 0, len(group.MemberIDs)+1)
		for _, memberID := range group.MemberIDs {
			if memberID != userID {
				members = append(members, memberID)
			}
		}
		if desired[group.ID] {
			members = append(members, userID)
		}
		if err := s.Groups.ReplaceMembers(group.ID, members); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) CreateAPIKey(userID, name string) (core.APIKey, error) {
	if s.APIKeys == nil {
		return core.APIKey{}, fmt.Errorf("api keys are not configured")
	}
	if strings.TrimSpace(name) == "" {
		return core.APIKey{}, fmt.Errorf("name is required")
	}
	if len([]rune(strings.TrimSpace(name))) > 80 {
		return core.APIKey{}, fmt.Errorf("name must be at most 80 characters")
	}
	settings, err := s.DeveloperSettings()
	if err != nil {
		return core.APIKey{}, err
	}
	if !settings.Enabled {
		return core.APIKey{}, fmt.Errorf("developer mode is disabled")
	}
	raw, prefix, hash, err := core.NewAPIKeySecret()
	if err != nil {
		return core.APIKey{}, err
	}
	now := time.Now().UTC()
	key := core.APIKey{ID: core.NewID("key"), Name: strings.TrimSpace(name), Prefix: prefix, Hash: hash, UserID: userID, Key: raw, CreatedAt: now}
	if settings.DefaultKeyTTLDays > 0 {
		expiresAt := now.Add(time.Duration(settings.DefaultKeyTTLDays) * 24 * time.Hour)
		key.ExpiresAt = &expiresAt
	}
	if err := s.APIKeys.Create(key); err != nil {
		return core.APIKey{}, err
	}
	return key, nil
}

func (s Service) ListAPIKeys(userID string) ([]core.APIKey, error) {
	if s.APIKeys == nil {
		return nil, nil
	}
	return s.APIKeys.ListByUser(userID)
}

func (s Service) ListAllAPIKeys() ([]core.APIKey, error) {
	if s.APIKeys == nil {
		return nil, nil
	}
	return s.APIKeys.List()
}

func (s Service) DeleteAPIKey(id string) error {
	if s.APIKeys == nil {
		return nil
	}
	return s.APIKeys.Delete(id)
}

func (s Service) IsAdmin(userID string) bool {
	if s.Groups == nil || userID == "" {
		return false
	}
	ok, err := s.Groups.HasMember("grp_admins", userID)
	if err == nil && ok {
		return true
	}
	if s.Settings == nil {
		return false
	}
	graph, err := s.PermissionGraph()
	if err != nil {
		return false
	}
	groups, err := s.Groups.GroupsForUser(userID)
	if err != nil {
		return false
	}
	for _, group := range groups {
		raw, ok := graph.DataGraph[group.ID]
		if !ok {
			continue
		}
		values, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if role, _ := values["admin"].(string); role == "admin" {
			return true
		}
	}
	return false
}

// HasCapability enforces the instance-wide group permission matrix. Project
// roles still decide which saved analyses and dashboards a user can access;
// these capabilities decide whether the user may operate on source data.
func (s Service) HasCapability(userID, capability, required string) bool {
	if s.IsAdmin(userID) {
		return true
	}
	ranks := map[string]map[string]int{
		"data": {"none": 0, "view": 1, "curate": 2},
		"sql":  {"none": 0, "query": 1, "native": 2},
	}
	levels, ok := ranks[capability]
	if !ok || levels[required] == 0 {
		return false
	}
	graph, err := s.PermissionGraph()
	if err != nil {
		return false
	}
	groups, err := s.Groups.GroupsForUser(userID)
	if err != nil {
		return false
	}
	for _, group := range groups {
		values, ok := graph.DataGraph[group.ID].(map[string]any)
		if !ok {
			continue
		}
		granted, _ := values[capability].(string)
		if levels[granted] >= levels[required] {
			return true
		}
	}
	return false
}

func (s Service) WithRole(user core.User) core.User {
	user.IsAdmin = s.IsAdmin(user.ID)
	return user
}

func (s Service) SiteName() string {
	if s.Settings == nil {
		return "Topbase"
	}
	value, ok, err := s.Settings.Get("site_name")
	if err != nil || !ok || strings.TrimSpace(value) == "" {
		return "Topbase"
	}
	return value
}

func (s Service) SetSiteName(name string) error {
	if s.Settings == nil {
		return fmt.Errorf("settings are not configured")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("site_name is required")
	}
	return s.Settings.Set("site_name", name)
}

func (s Service) AdminSettings() (core.AdminSettings, error) {
	// Public distribution is opt-in for every new Topbase instance. Administrators
	// can enable it deliberately from the instance settings when needed.
	settings := core.AdminSettings{SiteName: s.SiteName(), Timezone: "Asia/Shanghai", PublicSharingEnabled: false, EmbeddingEnabled: false}
	if s.Settings == nil {
		return settings, nil
	}
	raw, ok, err := s.Settings.Get("admin_settings")
	if err != nil || !ok || raw == "" {
		return settings, err
	}
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return core.AdminSettings{}, err
	}
	if settings.SiteName == "" {
		settings.SiteName = s.SiteName()
	}
	if settings.Timezone == "" {
		settings.Timezone = "Asia/Shanghai"
	}
	return settings, nil
}

func (s Service) SaveAdminSettings(settings core.AdminSettings) (core.AdminSettings, error) {
	settings.SiteName = strings.TrimSpace(settings.SiteName)
	settings.Timezone = strings.TrimSpace(settings.Timezone)
	if settings.SiteName == "" {
		return core.AdminSettings{}, fmt.Errorf("site_name is required")
	}
	if settings.Timezone == "" {
		settings.Timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(settings.Timezone); err != nil {
		return core.AdminSettings{}, fmt.Errorf("invalid timezone")
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return core.AdminSettings{}, err
	}
	if s.Settings == nil {
		return core.AdminSettings{}, fmt.Errorf("settings are not configured")
	}
	if err := s.Settings.Set("admin_settings", string(raw)); err != nil {
		return core.AdminSettings{}, err
	}
	if err := s.Settings.Set("site_name", settings.SiteName); err != nil {
		return core.AdminSettings{}, err
	}
	return settings, nil
}

func (s Service) IdentityProviders() ([]core.IdentityProvider, error) {
	return readSettingsJSON[core.IdentityProvider](s.Settings, "identity_providers")
}

func (s Service) AuthSettings() (core.AuthSettings, error) {
	value, ok, err := s.Settings.Get("auth_settings")
	if err != nil || !ok || strings.TrimSpace(value) == "" {
		return core.AuthSettings{PasswordLoginEnabled: true}, err
	}
	var settings core.AuthSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return core.AuthSettings{}, err
	}
	return settings, nil
}

func (s Service) SaveAuthSettings(settings core.AuthSettings) error {
	if !settings.PasswordLoginEnabled {
		providers, err := s.IdentityProviders()
		if err != nil {
			return err
		}
		enabled := false
		for _, provider := range providers {
			if supportsBrowserLogin(provider.Type) && provider.Enabled && strings.TrimSpace(provider.ClientID) != "" && strings.TrimSpace(provider.ClientSecret) != "" {
				enabled = true
				break
			}
		}
		if !enabled {
			return fmt.Errorf("enable and configure at least one third-party login before disabling password login")
		}
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return s.Settings.Set("auth_settings", string(raw))
}

func (s Service) ExternalIdentityLinks() ([]core.ExternalIdentityLink, error) {
	return readSettingsJSON[core.ExternalIdentityLink](s.Settings, "external_identity_links")
}

func (s Service) BindExternalIdentity(link core.ExternalIdentityLink) error {
	link.ProviderID, link.Subject, link.UserID = strings.TrimSpace(link.ProviderID), strings.TrimSpace(link.Subject), strings.TrimSpace(link.UserID)
	if link.ProviderID == "" || link.Subject == "" || link.UserID == "" {
		return fmt.Errorf("provider, account ID and member are required")
	}
	if _, err := s.Users.ByID(link.UserID); err != nil {
		return err
	}
	links, err := s.ExternalIdentityLinks()
	if err != nil {
		return err
	}
	for i := range links {
		if links[i].ProviderID == link.ProviderID && links[i].Subject == link.Subject {
			if links[i].UserID != link.UserID {
				return fmt.Errorf("this third-party account is already linked to another member")
			}
			return nil
		}
	}
	return writeSettingsJSON(s.Settings, "external_identity_links", append(links, link))
}

func (s Service) ExternalIdentityLinksForUser(userID string) ([]core.ExternalIdentityLink, error) {
	links, err := s.ExternalIdentityLinks()
	if err != nil {
		return nil, err
	}
	out := []core.ExternalIdentityLink{}
	for _, link := range links {
		if link.UserID == userID {
			out = append(out, link)
		}
	}
	return out, nil
}

func (s Service) UnbindExternalIdentity(userID, providerID string) error {
	links, err := s.ExternalIdentityLinks()
	if err != nil {
		return err
	}
	found := false
	kept := make([]core.ExternalIdentityLink, 0, len(links))
	for _, link := range links {
		if link.UserID == userID && link.ProviderID == providerID {
			found = true
			continue
		}
		kept = append(kept, link)
	}
	if !found {
		return core.ErrNotFound
	}
	return writeSettingsJSON(s.Settings, "external_identity_links", kept)
}

func (s Service) SaveIdentityProviders(items []core.IdentityProvider) error {
	for i := range items {
		items[i].ID = strings.TrimSpace(items[i].ID)
		items[i].Type = strings.TrimSpace(items[i].Type)
		items[i].Name = strings.TrimSpace(items[i].Name)
		if items[i].ID == "" || items[i].Type == "" || items[i].Name == "" {
			return fmt.Errorf("provider id, type and name are required")
		}
		if items[i].GroupMappings == nil {
			items[i].GroupMappings = map[string]string{}
		}
	}
	settings, err := s.AuthSettings()
	if err != nil {
		return err
	}
	if !settings.PasswordLoginEnabled {
		hasLogin := false
		for _, item := range items {
			if supportsBrowserLogin(item.Type) && item.Enabled && item.ClientID != "" && item.ClientSecret != "" {
				hasLogin = true
				break
			}
		}
		if !hasLogin {
			return fmt.Errorf("cannot remove the last configured third-party login while password login is disabled")
		}
	}
	return writeSettingsJSON(s.Settings, "identity_providers", items)
}

func supportsBrowserLogin(providerType string) bool {
	return providerType == "google" || providerType == "wechat"
}

func (s Service) Webhooks() ([]core.WebhookEndpoint, error) {
	return readSettingsJSON[core.WebhookEndpoint](s.Settings, "webhooks")
}
func (s Service) SaveWebhooks(items []core.WebhookEndpoint) error {
	for i := range items {
		items[i].ID, items[i].Name, items[i].Provider = strings.TrimSpace(items[i].ID), strings.TrimSpace(items[i].Name), strings.TrimSpace(items[i].Provider)
		if items[i].ID == "" || items[i].Name == "" || items[i].Provider == "" || strings.TrimSpace(items[i].URL) == "" {
			return fmt.Errorf("webhook id, name, provider and url are required")
		}
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now().UTC()
		}
	}
	return writeSettingsJSON(s.Settings, "webhooks", items)
}

func (s Service) ProjectAccessRules() ([]core.ProjectAccessRule, error) {
	return readSettingsJSON[core.ProjectAccessRule](s.Settings, "project_access_rules")
}
func (s Service) SaveProjectAccessRules(items []core.ProjectAccessRule) error {
	for _, item := range items {
		if item.ProjectID == "" || item.GroupID == "" || (item.Role != "view" && item.Role != "edit" && item.Role != "manage") {
			return fmt.Errorf("invalid project access rule")
		}
	}
	return writeSettingsJSON(s.Settings, "project_access_rules", items)
}

func (s Service) CanAccessProject(user core.User, project core.Collection, required string) bool {
	if user.ID == "" {
		return false
	}
	if s.IsAdmin(user.ID) {
		return true
	}
	if project.Kind == "personal_project" {
		return project.PersonalOwnerUserID == user.ID
	}
	roles := map[string]int{"view": 1, "edit": 2, "manage": 3}
	need := roles[required]
	if project.OwnerGroupID != "" {
		if ok, _ := s.Groups.HasMember(project.OwnerGroupID, user.ID); ok {
			return true
		}
	}
	rules, err := s.ProjectAccessRules()
	if err != nil {
		return false
	}
	for _, rule := range rules {
		if rule.ProjectID == project.ID && roles[rule.Role] >= need {
			if ok, _ := s.Groups.HasMember(rule.GroupID, user.ID); ok {
				return true
			}
		}
	}
	return false
}

func readSettingsJSON[T any](settings core.SettingStore, key string) ([]T, error) {
	if settings == nil {
		return []T{}, nil
	}
	raw, ok, err := settings.Get(key)
	if err != nil || !ok || raw == "" {
		return []T{}, err
	}
	var items []T
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	return items, nil
}
func writeSettingsJSON[T any](settings core.SettingStore, key string, items []T) error {
	if settings == nil {
		return fmt.Errorf("settings are not configured")
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return settings.Set(key, string(raw))
}

func (s Service) UserForAPIKey(raw string) (core.User, error) {
	if s.APIKeys == nil {
		return core.User{}, fmt.Errorf("not signed in")
	}
	settings, err := s.DeveloperSettings()
	if err != nil {
		return core.User{}, fmt.Errorf("read developer settings: %w", err)
	}
	if !settings.Enabled {
		return core.User{}, fmt.Errorf("developer mode is disabled")
	}
	key, err := s.APIKeys.ByHash(core.HashAPIKey(raw))
	if err != nil {
		return core.User{}, fmt.Errorf("invalid api key")
	}
	if key.ExpiresAt != nil && !time.Now().UTC().Before(*key.ExpiresAt) {
		return core.User{}, fmt.Errorf("api key expired")
	}
	user, err := s.Users.ByID(key.UserID)
	if err != nil || !user.IsActive {
		return core.User{}, fmt.Errorf("api key owner is inactive")
	}
	return user, nil
}

func (s Service) DeveloperSettings() (core.DeveloperSettings, error) {
	defaults := core.DeveloperSettings{
		Enabled: false, AllowPersonalKeys: true, AllowAnalysisWrite: false,
		DefaultKeyTTLDays: 90, MaxQueryRows: 200,
	}
	if s.Settings == nil {
		return defaults, nil
	}
	raw, ok, err := s.Settings.Get("developer_settings")
	if err != nil || !ok || strings.TrimSpace(raw) == "" {
		return defaults, err
	}
	if err := json.Unmarshal([]byte(raw), &defaults); err != nil {
		return core.DeveloperSettings{}, err
	}
	if defaults.MaxQueryRows == 0 {
		defaults.MaxQueryRows = 200
	}
	return defaults, nil
}

func (s Service) SaveDeveloperSettings(settings core.DeveloperSettings) (core.DeveloperSettings, error) {
	settings.PublicBaseURL = strings.TrimRight(strings.TrimSpace(settings.PublicBaseURL), "/")
	if settings.DefaultKeyTTLDays < 0 || settings.DefaultKeyTTLDays > 3650 {
		return core.DeveloperSettings{}, fmt.Errorf("default_key_ttl_days must be between 0 and 3650")
	}
	if settings.MaxQueryRows < 1 || settings.MaxQueryRows > 2000 {
		return core.DeveloperSettings{}, fmt.Errorf("max_query_rows must be between 1 and 2000")
	}
	if settings.PublicBaseURL != "" {
		parsed, err := url.Parse(settings.PublicBaseURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return core.DeveloperSettings{}, fmt.Errorf("public_base_url must be an http or https URL")
		}
		if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return core.DeveloperSettings{}, fmt.Errorf("public_base_url cannot contain credentials, query parameters or fragments")
		}
	}
	if s.Settings == nil {
		return core.DeveloperSettings{}, fmt.Errorf("settings are not configured")
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return core.DeveloperSettings{}, err
	}
	if err := s.Settings.Set("developer_settings", string(raw)); err != nil {
		return core.DeveloperSettings{}, err
	}
	return settings, nil
}

func (s Service) PermissionGraph() (core.PermissionGraph, error) {
	raw, ok, err := s.Settings.Get("permission_graph")
	if err != nil || !ok || raw == "" {
		return core.PermissionGraph{Revision: 1, DataGraph: map[string]any{}, CollectionGraph: map[string]any{}}, err
	}
	var graph core.PermissionGraph
	if err := json.Unmarshal([]byte(raw), &graph); err != nil {
		return core.PermissionGraph{}, err
	}
	return graph, nil
}

func (s Service) SavePermissionGraph(graph core.PermissionGraph) (core.PermissionGraph, error) {
	graph.Revision++
	if graph.DataGraph == nil {
		graph.DataGraph = map[string]any{}
	}
	if graph.CollectionGraph == nil {
		graph.CollectionGraph = map[string]any{}
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		return core.PermissionGraph{}, err
	}
	if err := s.Settings.Set("permission_graph", string(raw)); err != nil {
		return core.PermissionGraph{}, err
	}
	return graph, nil
}
