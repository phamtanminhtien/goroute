package rtk

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

const (
	minLines                 = 80
	smartTruncateMinLines    = 160
	headLines                = 120
	tailLines                = 60
	dedupLineMax             = 2000
	treeMaxLines             = 200
	searchListMaxDirs        = 20
	searchListMaxFilesPerDir = 10
)

type Hint struct {
	RequestType string
	Role        string
	FieldPath   string
}

type Result struct {
	Text        string
	Applied     bool
	BytesBefore int
	BytesAfter  int
	Hits        int
	Filters     []string
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (s Service) CompressChatCompletions(req openaiwire.ChatCompletionsRequest) (openaiwire.ChatCompletionsRequest, chatcompletion.RTKSummary) {
	out := req
	summary := chatcompletion.RTKSummary{}

	for i, msg := range out.Messages {
		role := string(msg.Role)
		if !shouldCompressRole(role) {
			continue
		}

		if !msg.Content.IsParts() {
			result := s.CompressText(msg.Content.Text(), Hint{
				RequestType: "chat_completions",
				Role:        role,
				FieldPath:   fmt.Sprintf("messages[%d].content", i),
			})
			if result.Applied {
				out.Messages[i].Content = openaiwire.TextContent(result.Text)
				accumulate(&summary, result)
			}
			continue
		}

		parts := msg.Content.Parts()
		changed := false
		for partIndex, part := range parts {
			if part.Type != "text" {
				continue
			}
			result := s.CompressText(part.Text, Hint{
				RequestType: "chat_completions",
				Role:        role,
				FieldPath:   fmt.Sprintf("messages[%d].content[%d].text", i, partIndex),
			})
			if !result.Applied {
				continue
			}
			parts[partIndex].Text = result.Text
			changed = true
			accumulate(&summary, result)
		}
		if changed {
			out.Messages[i].Content = openaiwire.PartsContent(parts...)
		}
	}

	return out, summary
}

func (s Service) CompressResponses(req openaiwire.ResponsesRequest) (openaiwire.ResponsesRequest, chatcompletion.RTKSummary) {
	out := req
	summary := chatcompletion.RTKSummary{}

	if strings.TrimSpace(out.InputText) != "" {
		result := s.CompressText(out.InputText, Hint{
			RequestType: "responses",
			Role:        string(openaiwire.ChatRoleUser),
			FieldPath:   "input_text",
		})
		if result.Applied {
			out.InputText = result.Text
			accumulate(&summary, result)
		}
	}

	for itemIndex, item := range out.Input {
		switch item.Type {
		case "message":
			if !shouldCompressRole(item.Role) {
				continue
			}
			for partIndex, part := range item.Content {
				if part.Type != "input_text" {
					continue
				}
				result := s.CompressText(part.Text, Hint{
					RequestType: "responses",
					Role:        item.Role,
					FieldPath:   fmt.Sprintf("input[%d].content[%d].text", itemIndex, partIndex),
				})
				if !result.Applied {
					continue
				}
				out.Input[itemIndex].Content[partIndex].Text = result.Text
				accumulate(&summary, result)
			}
		case "function_call_output":
			result := s.CompressText(item.Output, Hint{
				RequestType: "responses",
				Role:        "tool",
				FieldPath:   fmt.Sprintf("input[%d].output", itemIndex),
			})
			if result.Applied {
				out.Input[itemIndex].Output = result.Text
				accumulate(&summary, result)
			}
		}
	}

	if len(out.RawBody) > 0 {
		rawBody, err := s.compressResponsesRawBody(out.RawBody)
		if err == nil {
			out.RawBody = rawBody
		}
	}

	return out, summary
}

func (s Service) compressResponsesRawBody(raw json.RawMessage) (json.RawMessage, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	if input, ok := payload["input"]; ok {
		next := s.compressResponsesRawInput(input)
		payload["input"] = next
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return encoded, nil
}

func (s Service) compressResponsesRawInput(input any) any {
	switch typed := input.(type) {
	case string:
		result := s.CompressText(typed, Hint{
			RequestType: "responses",
			Role:        string(openaiwire.ChatRoleUser),
			FieldPath:   "input",
		})
		if result.Applied {
			return result.Text
		}
		return input
	case []any:
		for itemIndex, rawItem := range typed {
			itemMap, ok := rawItem.(map[string]any)
			if !ok {
				continue
			}
			itemType, _ := itemMap["type"].(string)
			switch itemType {
			case "message":
				role, _ := itemMap["role"].(string)
				if !shouldCompressRole(role) {
					continue
				}
				content, ok := itemMap["content"].([]any)
				if !ok {
					continue
				}
				for partIndex, rawPart := range content {
					partMap, ok := rawPart.(map[string]any)
					if !ok {
						continue
					}
					partType, _ := partMap["type"].(string)
					if partType != "input_text" {
						continue
					}
					text, _ := partMap["text"].(string)
					result := s.CompressText(text, Hint{
						RequestType: "responses",
						Role:        role,
						FieldPath:   fmt.Sprintf("input[%d].content[%d].text", itemIndex, partIndex),
					})
					if !result.Applied {
						continue
					}
					partMap["text"] = result.Text
				}
			case "function_call_output":
				output, _ := itemMap["output"].(string)
				result := s.CompressText(output, Hint{
					RequestType: "responses",
					Role:        "tool",
					FieldPath:   fmt.Sprintf("input[%d].output", itemIndex),
				})
				if !result.Applied {
					continue
				}
				itemMap["output"] = result.Text
			}
		}
	}

	return input
}

func (s Service) CompressText(input string, hint Hint) Result {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	if !shouldCompressRole(hint.Role) || shouldSkipText(input) {
		return Result{Text: input}
	}

	lines := splitLines(input)
	if len(lines) < minLines && !hasConsecutiveDuplicates(lines) {
		return Result{Text: input}
	}

	text := input
	filters := make([]string, 0, 2)

	switch {
	case isLSOutput(lines):
		text = filterLS(lines)
		filters = append(filters, "ls")
	case isTreeOutput(lines):
		text = filterTree(lines)
		filters = append(filters, "tree")
	case isReadNumbered(lines):
		text = filterReadNumbered(lines)
		filters = append(filters, "read-numbered")
	case isSearchList(lines):
		text = filterSearchList(lines)
		filters = append(filters, "search-list")
	case hasConsecutiveDuplicates(lines) || len(lines) >= minLines:
		text = filterDedupLog(lines)
		if text != input {
			filters = append(filters, "dedup-log")
		}
	}

	if len(filters) == 0 {
		return Result{Text: input}
	}

	if len(splitLines(text)) >= smartTruncateMinLines && !contains(filters, "tree") && !contains(filters, "read-numbered") && !contains(filters, "ls") && !contains(filters, "search-list") {
		truncated := filterSmartTruncate(splitLines(text), "... +%d lines truncated")
		if truncated != text {
			text = truncated
			filters = append(filters, "smart-truncate")
		}
	}

	if text == input {
		return Result{Text: input}
	}

	return Result{
		Text:        text,
		Applied:     true,
		BytesBefore: len(input),
		BytesAfter:  len(text),
		Hits:        len(filters),
		Filters:     filters,
	}
}

func shouldCompressRole(role string) bool {
	switch role {
	case string(openaiwire.ChatRoleUser), string(openaiwire.ChatRoleTool):
		return true
	default:
		return false
	}
}

func shouldSkipText(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return true
	}
	if strings.Contains(trimmed, "data:") || strings.Contains(trimmed, "base64,") {
		return true
	}
	if len(trimmed) > 2048 && !strings.Contains(trimmed, "\n") {
		return true
	}
	return false
}

func splitLines(input string) []string {
	return strings.Split(input, "\n")
}

func hasConsecutiveDuplicates(lines []string) bool {
	for i := 1; i < len(lines); i++ {
		if lines[i] != "" && lines[i] == lines[i-1] {
			return true
		}
	}
	return false
}

func isLSOutput(lines []string) bool {
	if len(lines) < minLines {
		return false
	}
	matches := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 8 && strings.HasPrefix(fields[0], "-") || len(fields) >= 8 && strings.HasPrefix(fields[0], "d") {
			matches++
		}
	}
	return matches >= minLines/2
}

func isTreeOutput(lines []string) bool {
	if len(lines) < minLines {
		return false
	}
	matches := 0
	for _, line := range lines {
		if strings.Contains(line, "├──") || strings.Contains(line, "└──") || strings.Contains(line, "│") {
			matches++
		}
	}
	return matches >= minLines/3
}

func isReadNumbered(lines []string) bool {
	if len(lines) < minLines {
		return false
	}
	matches := 0
	for _, line := range lines {
		if isNumberedLine(line) {
			matches++
		}
	}
	return matches >= minLines/2
}

func isNumberedLine(line string) bool {
	line = strings.TrimLeft(line, " ")
	if line == "" {
		return false
	}
	index := strings.IndexByte(line, '|')
	if index <= 0 {
		return false
	}
	for _, r := range line[:index] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isSearchList(lines []string) bool {
	if len(lines) < minLines {
		return false
	}
	matches := 0
	for _, line := range lines {
		if looksLikePath(line) {
			matches++
		}
	}
	return matches >= (len(lines) * 3 / 4)
}

func looksLikePath(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || strings.Contains(line, " ") {
		return false
	}
	return strings.Contains(line, "/") || strings.HasSuffix(line, ".go") || strings.HasSuffix(line, ".ts") || strings.HasSuffix(line, ".tsx") || strings.HasSuffix(line, ".md")
}

func filterDedupLog(lines []string) string {
	out := make([]string, 0, min(len(lines), dedupLineMax))
	blankRun := false
	for i := 0; i < len(lines); {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			if !blankRun {
				out = append(out, "")
				blankRun = true
			}
			i++
			continue
		}

		blankRun = false
		run := 1
		for i+run < len(lines) && lines[i+run] == line {
			run++
		}
		out = append(out, line)
		if run > 1 {
			out = append(out, fmt.Sprintf("... (%d duplicate lines)", run-1))
		}
		i += run
		if len(out) >= dedupLineMax {
			break
		}
	}

	if len(out) > dedupLineMax {
		out = out[:dedupLineMax]
	}

	return strings.Join(out, "\n")
}

func filterLS(lines []string) string {
	type extCount struct {
		ext   string
		count int
	}

	out := make([]string, 0, len(lines))
	files := 0
	dirs := 0
	extensions := map[string]int{}

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		if fields[0] == "total" {
			continue
		}
		name := strings.Join(fields[8:], " ")
		if name == "." || name == ".." || name == "" {
			continue
		}

		isDir := strings.HasPrefix(fields[0], "d")
		if isDir {
			dirs++
			out = append(out, name+"/")
			continue
		}

		files++
		size := fields[4]
		out = append(out, strings.TrimSpace(name)+"  "+size)
		ext := filepath.Ext(name)
		if ext != "" {
			extensions[ext]++
		}
	}

	exts := make([]extCount, 0, len(extensions))
	for ext, count := range extensions {
		exts = append(exts, extCount{ext: ext, count: count})
	}
	sort.Slice(exts, func(i, j int) bool {
		if exts[i].count == exts[j].count {
			return exts[i].ext < exts[j].ext
		}
		return exts[i].count > exts[j].count
	})

	summary := make([]string, 0, min(3, len(exts)))
	for _, ext := range exts[:min(3, len(exts))] {
		summary = append(summary, fmt.Sprintf("%s=%d", ext.ext, ext.count))
	}
	out = append(out, fmt.Sprintf("summary: %d files, %d dirs", files, dirs))
	if len(summary) > 0 {
		out = append(out, "top extensions: "+strings.Join(summary, ", "))
	}

	return strings.Join(out, "\n")
}

func filterTree(lines []string) string {
	trimmed := trimBlankLines(lines)
	out := make([]string, 0, len(trimmed))
	for _, line := range trimmed {
		if treeFooter(line) {
			continue
		}
		out = append(out, line)
	}
	if len(out) > treeMaxLines {
		remaining := len(out) - treeMaxLines
		out = append(out[:treeMaxLines], fmt.Sprintf("... +%d more lines", remaining))
	}
	return strings.Join(out, "\n")
}

func treeFooter(line string) bool {
	line = strings.TrimSpace(line)
	return strings.Contains(line, "directories") && strings.Contains(line, "files")
}

func filterReadNumbered(lines []string) string {
	return filterSmartTruncate(lines, "... +%d lines truncated (file continues)")
}

func filterSmartTruncate(lines []string, pattern string) string {
	if len(lines) < smartTruncateMinLines {
		return strings.Join(lines, "\n")
	}
	if len(lines) <= headLines+tailLines {
		return strings.Join(lines, "\n")
	}
	truncated := len(lines) - headLines - tailLines
	out := make([]string, 0, headLines+tailLines+1)
	out = append(out, lines[:headLines]...)
	out = append(out, fmt.Sprintf(pattern, truncated))
	out = append(out, lines[len(lines)-tailLines:]...)
	return strings.Join(out, "\n")
}

func filterSearchList(lines []string) string {
	groups := map[string][]string{}
	dirs := make([]string, 0)

	for _, line := range lines {
		path := strings.TrimSpace(line)
		if !looksLikePath(path) {
			continue
		}
		dir := filepath.Dir(path)
		base := filepath.Base(path)
		if _, ok := groups[dir]; !ok {
			dirs = append(dirs, dir)
		}
		groups[dir] = append(groups[dir], base)
	}

	sort.Strings(dirs)
	totalDirs := len(dirs)
	if totalDirs > searchListMaxDirs {
		dirs = dirs[:searchListMaxDirs]
	}

	out := make([]string, 0, len(dirs)+1)
	for _, dir := range dirs {
		files := groups[dir]
		sort.Strings(files)
		display := files
		overflow := 0
		if len(display) > searchListMaxFilesPerDir {
			overflow = len(display) - searchListMaxFilesPerDir
			display = display[:searchListMaxFilesPerDir]
		}
		out = append(out, dir+": "+strings.Join(display, ", "))
		if overflow > 0 {
			out = append(out, fmt.Sprintf("%s: ... +%d more files", dir, overflow))
		}
	}
	if totalDirs > len(dirs) {
		out = append(out, fmt.Sprintf("... +%d more directories", totalDirs-len(dirs)))
	}
	return strings.Join(out, "\n")
}

func trimBlankLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return append([]string(nil), lines[start:end]...)
}

func accumulate(summary *chatcompletion.RTKSummary, result Result) {
	if summary == nil || !result.Applied {
		return
	}

	summary.Applied = true
	summary.BytesBefore += result.BytesBefore
	summary.BytesAfter += result.BytesAfter
	summary.SavedBytes += result.BytesBefore - result.BytesAfter
	summary.HitCount += result.Hits
	summary.FieldCount++
	for _, filter := range result.Filters {
		if !contains(summary.FilterChain, filter) {
			summary.FilterChain = append(summary.FilterChain, filter)
		}
	}
	if summary.BytesBefore > 0 {
		summary.SavedPercent = (summary.SavedBytes * 100) / summary.BytesBefore
	}
}

func mergeSummary(dst *chatcompletion.RTKSummary, src chatcompletion.RTKSummary) {
	if dst == nil || !src.Applied {
		return
	}

	if !dst.Applied {
		*dst = src
		return
	}

	dst.BytesBefore += src.BytesBefore
	dst.BytesAfter += src.BytesAfter
	dst.SavedBytes += src.SavedBytes
	dst.HitCount += src.HitCount
	dst.FieldCount += src.FieldCount
	for _, filter := range src.FilterChain {
		if !contains(dst.FilterChain, filter) {
			dst.FilterChain = append(dst.FilterChain, filter)
		}
	}
	if dst.BytesBefore > 0 {
		dst.Applied = true
		dst.SavedPercent = (dst.SavedBytes * 100) / dst.BytesBefore
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
