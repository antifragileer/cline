// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
)

// StaticRegion represents a region of static content that has been "logged" and won't change.
// This is the Go equivalent of Ink's <Static> component.
type StaticRegion struct {
	mu sync.RWMutex

	// renderedContent contains all messages that have been rendered to the static region
	renderedContent map[string]string // key -> content hash

	// contentOrder maintains the order of keys for consistent rendering
	contentOrder []string

	// renderedOutput is the cached concatenated output
	renderedOutput string

	// dirty flag indicates if the rendered output needs regeneration
	dirty bool
}

// NewStaticRegion creates a new static region.
func NewStaticRegion() *StaticRegion {
	return &StaticRegion{
		renderedContent: make(map[string]string),
		contentOrder:    make([]string, 0),
		dirty:           true,
	}
}

// ContentItem represents an item to be rendered in the static region.
type ContentItem struct {
	Key     string
	Content string
}

// Add adds or updates content in the static region.
// Returns true if the content was new or changed, false if it was already present with the same content.
func (sr *StaticRegion) Add(key, content string) bool {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Calculate hash of content
	hash := hashContent(content)

	// Check if this key already exists with the same content
	if existingHash, exists := sr.renderedContent[key]; exists && existingHash == hash {
		// Content unchanged, no need to update
		return false
	}

	// New or changed content
	sr.renderedContent[key] = hash
	sr.dirty = true

	// If this is a new key, add it to the order
	if _, exists := sr.renderedContent[key]; !exists || !stringSliceContains(sr.contentOrder, key) {
		sr.contentOrder = append(sr.contentOrder, key)
	}

	return true
}

// Remove removes content from the static region.
func (sr *StaticRegion) Remove(key string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.renderedContent[key]; exists {
		delete(sr.renderedContent, key)
		sr.contentOrder = removeFromSlice(sr.contentOrder, key)
		sr.dirty = true
	}
}

// Has returns true if the key exists in the static region.
func (sr *StaticRegion) Has(key string) bool {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	_, exists := sr.renderedContent[key]
	return exists
}

// GetRenderedOutput returns the concatenated rendered output of all static content.
// This should be called once per render cycle.
func (sr *StaticRegion) GetRenderedOutput(items []ContentItem) string {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Check if any items are new or changed
	changed := false
	for _, item := range items {
		hash := hashContent(item.Content)
		if existingHash, exists := sr.renderedContent[item.Key]; !exists || existingHash != hash {
			sr.renderedContent[item.Key] = hash
			if !stringSliceContains(sr.contentOrder, item.Key) {
				sr.contentOrder = append(sr.contentOrder, item.Key)
			}
			changed = true
		}
	}

	if changed {
		sr.dirty = true
	}

	// Regenerate output if dirty
	if sr.dirty {
		sr.regenerateOutput(items)
		sr.dirty = false
	}

	return sr.renderedOutput
}

// regenerateOutput rebuilds the rendered output from the current items.
func (sr *StaticRegion) regenerateOutput(items []ContentItem) {
	var builder strings.Builder

	// Build output in order
	for _, key := range sr.contentOrder {
		// Find the item with this key
		for _, item := range items {
			if item.Key == key {
				if builder.Len() > 0 {
					builder.WriteString("\n")
				}
				builder.WriteString(item.Content)
				break
			}
		}
	}

	sr.renderedOutput = builder.String()
}

// Clear removes all content from the static region.
func (sr *StaticRegion) Clear() {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.renderedContent = make(map[string]string)
	sr.contentOrder = make([]string, 0)
	sr.renderedOutput = ""
	sr.dirty = true
}

// Size returns the number of items in the static region.
func (sr *StaticRegion) Size() int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	return len(sr.contentOrder)
}

// hashContent creates a hash of the content for change detection.
func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for efficiency
}

// contains checks if a string slice contains a specific string.
func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// removeFromSlice removes an item from a string slice.
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// StaticRenderer provides high-level static/dynamic rendering capabilities.
type StaticRenderer struct {
	staticRegion *StaticRegion

	// Message tracking for deduplication
	messageKeys map[string]bool

	// Content that should skip dynamic rendering
	skipDynamicTypes map[string]bool
}

// NewStaticRenderer creates a new static renderer.
func NewStaticRenderer() *StaticRenderer {
	return &StaticRenderer{
		staticRegion: NewStaticRegion(),
		messageKeys:  make(map[string]bool),
		skipDynamicTypes: map[string]bool{
			"completion_result": true,
			"plan_mode_respond": true,
		},
	}
}

// ShouldSkipDynamic returns true if a message type should skip dynamic rendering.
func (sr *StaticRenderer) ShouldSkipDynamic(msgType, subType string) bool {
	if sr.skipDynamicTypes[msgType] {
		return true
	}
	if sr.skipDynamicTypes[subType] {
		return true
	}
	return false
}

// IsFileEditToolMessage returns true if the message is a file edit tool.
func (sr *StaticRenderer) IsFileEditToolMessage(toolName string) bool {
	// File edit tools that should skip dynamic rendering
	fileEditTools := map[string]bool{
		"apply_diff":      true,
		"write_to_file":   true,
		"replace_in_file": true,
	}
	return fileEditTools[toolName]
}

// ShouldCommandStayInDynamic returns true if a command message should stay in dynamic region.
func (sr *StaticRenderer) ShouldCommandStayInDynamic(isCommand, isCompleted, hasOutput, isLast bool) bool {
	if !isCommand {
		return false
	}

	// If not completed, definitely stay in dynamic
	if !isCompleted {
		return true
	}

	// If completed but no output yet AND still the last message,
	// stay in dynamic to allow output to be combined
	if !hasOutput && isLast {
		return true
	}

	return false
}

// IsUnselectedFollowup returns true if this is a followup with options but no selection.
func (sr *StaticRenderer) IsUnselectedFollowup(msgType, askType, text string) bool {
	if msgType == string(MessageTypeAsk) && askType == "followup" && text != "" {
		// Check if it has options but no selected field
		// This is a simplified check - in full implementation would parse JSON
		return strings.Contains(text, "\"options\"") && !strings.Contains(text, "\"selected\"")
	}
	return false
}

// AddToStatic adds a message to the static region.
// Returns true if the message was new and added.
func (sr *StaticRenderer) AddToStatic(key, content string) bool {
	return sr.staticRegion.Add(key, content)
}

// GetStaticOutput returns the current static region output.
func (sr *StaticRenderer) GetStaticOutput(items []ContentItem) string {
	return sr.staticRegion.GetRenderedOutput(items)
}

// Clear clears all static content.
func (sr *StaticRenderer) Clear() {
	sr.staticRegion.Clear()
	sr.messageKeys = make(map[string]bool)
}

// StaticRegionSize returns the number of items in the static region.
func (sr *StaticRenderer) StaticRegionSize() int {
	return sr.staticRegion.Size()
}

// ContentPartition represents the partition of messages into static and dynamic regions.
type ContentPartition struct {
	StaticItems []ContentItem
	DynamicItem *ContentItem
	HasDynamic  bool
}

// PartitionMessages partitions messages into static and dynamic regions.
// This implements the logic from ChatView.tsx lines 688-771.
func (sr *StaticRenderer) PartitionMessages(messages []Message, hasUserScrolled bool) ContentPartition {
	partition := ContentPartition{
		StaticItems: make([]ContentItem, 0),
	}

	// Add header as static if messages exist or user has scrolled
	if len(messages) > 0 || hasUserScrolled {
		// Header will be rendered by the view, we just track that it should be static
	}

	for i, msg := range messages {
		isLast := i == len(messages)-1

		// Get message key
		key := msg.GetKey()
		if key == "" {
			key = string(rune(i))
		}

		// Determine message type info
		msgType := string(msg.Type)
		subType := ""
		if msg.Type == MessageTypeSay {
			subType = msg.SayType
		} else if msg.Type == MessageTypeAsk {
			subType = msg.AskType
		}

		// Check if this message type should skip dynamic rendering
		shouldSkipDynamic := sr.ShouldSkipDynamic(subType, msgType) ||
			(msg.Type == MessageTypeToolUse && sr.IsFileEditToolMessage(msg.ToolName))

		// Check if this is an unselected followup
		isUnselectedFollowup := sr.IsUnselectedFollowup(msgType, subType, msg.Content)

		// Check if command should stay in dynamic
		isCommand := subType == "command"
		hasOutput := msg.HasOutput
		shouldCommandStayInDynamic := sr.ShouldCommandStayInDynamic(isCommand, msg.CommandCompleted, hasOutput, isLast)

		// Determine where this message goes
		if msg.Partial {
			// Message is still streaming
			if isLast && !shouldSkipDynamic {
				// Show in dynamic region
				partition.DynamicItem = &ContentItem{
					Key:     key,
					Content: "", // Content will be rendered dynamically
				}
				partition.HasDynamic = true
			}
			// If shouldSkipDynamic and partial: don't show anywhere, wait for complete
		} else if isLast && isUnselectedFollowup {
			// Keep unselected followup in dynamic region
			partition.DynamicItem = &ContentItem{
				Key:     key,
				Content: "",
			}
			partition.HasDynamic = true
		} else if shouldCommandStayInDynamic && isLast {
			// Command needs to stay in dynamic to allow output to be combined
			partition.DynamicItem = &ContentItem{
				Key:     key,
				Content: "",
			}
			partition.HasDynamic = true
		} else {
			// Message is complete, add to static
			partition.StaticItems = append(partition.StaticItems, ContentItem{
				Key:     key,
				Content: "", // Content will be rendered separately
			})
		}
	}

	return partition
}

// MessageKey generates a unique key for a message.
func (sr *StaticRenderer) MessageKey(msg Message, index int) string {
	if msg.ID != "" {
		return msg.ID
	}
	if msg.Timestamp.UnixNano() > 0 {
		return string(rune(msg.Timestamp.UnixNano()))
	}
	return string(rune(index))
}
