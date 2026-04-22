package data

import (
	"strconv"
	"strings"
	"sync"

	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"gorm.io/gorm/clause"
)

// Simple in-process cache. Settings are read frequently (on every order
// create, every scanner tick, etc.); the DB round-trip isn't free. Cache
// is invalidated explicitly via SetString/Delete and expires on process
// restart — that's enough for admin-edit semantics without TTL complexity.
var (
	settingsCache   = map[string]string{}
	settingsCacheMu sync.RWMutex
	settingsLoadMu  sync.Mutex // serializes the lazy bootstrap load
	settingsLoaded  bool
)

func loadAllSettings() error {
	var rows []mdb.Setting
	if err := dao.Mdb.Find(&rows).Error; err != nil {
		return err
	}
	settingsCacheMu.Lock()
	defer settingsCacheMu.Unlock()
	settingsCache = make(map[string]string, len(rows))
	for _, r := range rows {
		settingsCache[r.Key] = r.Value
	}
	settingsLoaded = true
	return nil
}

// ensureLoaded performs a thread-safe lazy bootstrap of the cache. Uses
// a dedicated mutex + double-checked flag so two concurrent callers
// don't both hit the DB. Can't use sync.Once because ReloadSettings
// needs to be able to rebuild after a failed initial load.
func ensureLoaded() {
	settingsCacheMu.RLock()
	loaded := settingsLoaded
	settingsCacheMu.RUnlock()
	if loaded {
		return
	}
	settingsLoadMu.Lock()
	defer settingsLoadMu.Unlock()
	// Recheck under the load mutex — another goroutine may have loaded
	// while we were blocked.
	settingsCacheMu.RLock()
	loaded = settingsLoaded
	settingsCacheMu.RUnlock()
	if loaded {
		return
	}
	_ = loadAllSettings()
}

// ReloadSettings forces a refresh from DB. Call after external writes.
func ReloadSettings() error {
	settingsLoadMu.Lock()
	defer settingsLoadMu.Unlock()
	return loadAllSettings()
}

// GetSettingString returns the string value for key, or fallback if unset.
func GetSettingString(key, fallback string) string {
	ensureLoaded()
	settingsCacheMu.RLock()
	defer settingsCacheMu.RUnlock()
	v, ok := settingsCache[key]
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// GetSettingInt returns the int value for key, or fallback on miss/parse fail.
func GetSettingInt(key string, fallback int) int {
	v := GetSettingString(key, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// GetSettingFloat returns the float value for key, or fallback on miss/parse fail.
func GetSettingFloat(key string, fallback float64) float64 {
	v := GetSettingString(key, "")
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

// GetSettingBool returns the bool value for key, or fallback on miss/parse fail.
func GetSettingBool(key string, fallback bool) bool {
	v := GetSettingString(key, "")
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

// SetSetting upserts a setting row and refreshes the cache entry.
func SetSetting(group, key, value, valueType string) error {
	if valueType == "" {
		valueType = mdb.SettingTypeString
	}
	row := mdb.Setting{Group: group, Key: key, Value: value, Type: valueType}
	err := dao.Mdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"group", "value", "type", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return err
	}
	settingsCacheMu.Lock()
	settingsCache[key] = value
	settingsCacheMu.Unlock()
	return nil
}

// DeleteSetting removes a setting row and drops the cache entry.
func DeleteSetting(key string) error {
	if err := dao.Mdb.Where("key = ?", key).Delete(&mdb.Setting{}).Error; err != nil {
		return err
	}
	settingsCacheMu.Lock()
	delete(settingsCache, key)
	settingsCacheMu.Unlock()
	return nil
}

// sensitiveSettingKeys lists keys that must never be returned to API callers.
var sensitiveSettingKeys = []string{
	mdb.SettingKeyJwtSecret,
	mdb.SettingKeyInitAdminPasswordPlain,
	mdb.SettingKeyInitAdminPasswordHash,
	mdb.SettingKeyInitAdminPasswordFetched,
	mdb.SettingKeyInitAdminPasswordChanged,
}

// ListSettingsByGroup returns all rows for a given group (empty group = all),
// excluding any keys in sensitiveSettingKeys.
func ListSettingsByGroup(group string) ([]mdb.Setting, error) {
	var rows []mdb.Setting
	tx := dao.Mdb.Model(&mdb.Setting{}).Not("`key`", sensitiveSettingKeys)
	if group != "" {
		tx = tx.Where("`group` = ?", group)
	}
	err := tx.Order("`key` ASC").Find(&rows).Error
	return rows, err
}
