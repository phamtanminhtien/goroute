package codex

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

const (
	defaultMachineIDSalt        = "endpoint-proxy-salt"
	codexSessionTTL             = time.Hour
	codexSessionCleanupInterval = 10 * time.Minute
)

var (
	cachedMachineID = getConsistentMachineID()

	sessionStore = codexConversationSessionStore{
		entries: make(map[string]codexConversationSessionEntry),
		now:     time.Now,
	}
	sessionCleanupOnce sync.Once
)

type codexConversationSessionEntry struct {
	sessionID string
	expiresAt time.Time
}

type codexConversationSessionStore struct {
	mu      sync.Mutex
	entries map[string]codexConversationSessionEntry
	now     func() time.Time
}

func getConsistentMachineID() string {
	id, err := machineid.ID()
	if err != nil || strings.TrimSpace(id) == "" {
		if hostname, hostErr := os.Hostname(); hostErr == nil && strings.TrimSpace(hostname) != "" {
			id = hostname
		} else {
			id = "unknown-machine"
		}
	}
	return shortHash(strings.TrimSpace(id) + machineIDSalt())
}

func machineIDSalt() string {
	if salt := strings.TrimSpace(os.Getenv("MACHINE_ID_SALT")); salt != "" {
		return salt
	}
	return defaultMachineIDSalt
}

func resolveConversationSessionID(input []openaiwire.ResponseInputItem, machineID string) string {
	startConversationSessionCleanup()

	firstAssistantText := firstAssistantText(input)
	if strings.TrimSpace(firstAssistantText) == "" {
		return "sess_" + shortHash(machineID)
	}

	key := shortHash(machineID + firstAssistantText)
	return sessionStore.resolve(key)
}

func startConversationSessionCleanup() {
	sessionCleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(codexSessionCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				sessionStore.cleanupExpired()
			}
		}()
	})
}

func (s *codexConversationSessionStore) resolve(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if entry, ok := s.entries[key]; ok && now.Before(entry.expiresAt) {
		return entry.sessionID
	}

	sessionID := generatedSessionID(now)
	s.entries[key] = codexConversationSessionEntry{
		sessionID: sessionID,
		expiresAt: now.Add(codexSessionTTL),
	}
	return sessionID
}

func (s *codexConversationSessionStore) cleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	for key, entry := range s.entries {
		if !now.Before(entry.expiresAt) {
			delete(s.entries, key)
		}
	}
}

func generatedSessionID(now time.Time) string {
	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return fmt.Sprintf("sess_%d_%s", now.UnixMilli(), shortHash(now.String()))
	}
	return fmt.Sprintf("sess_%d_%s", now.UnixMilli(), hex.EncodeToString(randomBytes[:]))
}

func firstAssistantText(input []openaiwire.ResponseInputItem) string {
	for _, item := range input {
		if item.Type != codexInputTypeMessage || item.Role != string(openaiwire.ChatRoleAssistant) {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "output_text" || part.Type == "input_text" || part.Type == "text" || part.Type == "" {
				if strings.TrimSpace(part.Text) != "" {
					return part.Text
				}
			}
		}
	}
	return ""
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}
