// Package database loads Siemens CMC machine-data, setting-data, system
// variable, and SINAMICS parameter descriptions.
package database

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/textencoding"
)

type Parameter struct {
	Identifier  string
	Number      string
	Type        string
	Dimension   string
	ReadOnly    bool
	Name        string
	Brief       string
	Description string
	Note        string
	Attention   string
	Sources     []string
}

type Catalog struct {
	parameters map[string][]Parameter
	count      int
}

// Load parses a replaceable Siemens DataBase directory. German locales use
// the root descriptions; other locales prefer their subdirectory and then en.
func Load(directory, locale string) (*Catalog, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	dataDirectory := localizedDirectory(directory, locale)
	entries, err := os.ReadDir(dataDirectory)
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{parameters: make(map[string][]Parameter)}
	loadedFiles := 0
	for _, entry := range entries {
		if entry.IsDir() || !isDescriptionFile(entry.Name()) {
			continue
		}
		if err := catalog.loadFile(filepath.Join(dataDirectory, entry.Name())); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		loadedFiles++
	}
	if loadedFiles == 0 {
		return nil, fmt.Errorf("no Siemens parameter description files found in %s", dataDirectory)
	}
	return catalog, nil
}

// Locate searches an explicit location, workspace ancestors, the executable
// directory, and the current directory for a replaceable DataBase directory.
func Locate(roots []string, configured string) string {
	var candidates []string
	if configured != "" {
		candidates = append(candidates, configured)
	}
	for _, root := range roots {
		candidates = append(candidates, databaseCandidatesInAncestors(root)...)
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, databaseCandidatesInAncestors(filepath.Dir(executable))...)
	}
	if current, err := os.Getwd(); err == nil {
		candidates = append(candidates, databaseCandidatesInAncestors(current)...)
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if containsDescriptionFiles(candidate) {
			return candidate
		}
	}
	return ""
}

func databaseCandidatesInAncestors(path string) []string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil
	}
	var candidates []string
	for current := absolute; ; current = filepath.Dir(current) {
		candidates = append(candidates, filepath.Join(current, "DataBase"))
		parent := filepath.Dir(current)
		if parent == current {
			return candidates
		}
	}
}

func (catalog *Catalog) Count() int {
	if catalog == nil {
		return 0
	}
	return catalog.count
}

func (catalog *Catalog) Lookup(identifier string) []Parameter {
	if catalog == nil {
		return nil
	}
	key := canonicalIdentifier(identifier)
	if exact := catalog.parameters[key]; len(exact) > 0 {
		return slices.Clone(exact)
	}
	// Dynamic CMC identifiers commonly end in _$(...), for example
	// $MC_TRAFO_TYPE_$(Up.channel). In that case the source token ends in an
	// underscore and represents a numbered family in the Siemens database.
	if !strings.HasSuffix(key, "_") {
		return nil
	}
	var matches []Parameter
	for candidate, parameters := range catalog.parameters {
		if strings.HasPrefix(candidate, key) {
			matches = append(matches, parameters...)
		}
	}
	slices.SortFunc(matches, func(left, right Parameter) int {
		return strings.Compare(left.Identifier, right.Identifier)
	})
	return matches
}

func (catalog *Catalog) Hover(identifier string) (string, bool) {
	parameters := catalog.Lookup(identifier)
	if len(parameters) == 0 {
		return "", false
	}
	const maxVariants = 4
	var result strings.Builder
	for index, parameter := range parameters {
		if index == maxVariants {
			fmt.Fprintf(&result, "\n\n_%d additional drive-object variants omitted._", len(parameters)-maxVariants)
			break
		}
		if index > 0 {
			result.WriteString("\n\n---\n\n")
		}
		fmt.Fprintf(&result, "### `%s`", parameter.Identifier)
		if parameter.Brief != "" {
			result.WriteString(" — ")
			result.WriteString(parameter.Brief)
		}
		var metadata []string
		if parameter.Number != "" {
			metadata = append(metadata, "Number: `"+parameter.Number+"`")
		}
		if parameter.Type != "" {
			metadata = append(metadata, "Type: `"+parameter.Type+"`")
		}
		if parameter.Dimension != "" {
			metadata = append(metadata, "Dimension: `"+parameter.Dimension+"`")
		}
		if parameter.ReadOnly {
			metadata = append(metadata, "Read-only")
		}
		if len(metadata) > 0 {
			result.WriteString("\n\n")
			result.WriteString(strings.Join(metadata, " · "))
		}
		if parameter.Description != "" {
			result.WriteString("\n\n")
			result.WriteString(limit(parameter.Description, 2400))
		}
		if parameter.Attention != "" {
			result.WriteString("\n\n**Attention:** ")
			result.WriteString(limit(parameter.Attention, 700))
		}
		if parameter.Note != "" {
			result.WriteString("\n\n**Note:** ")
			result.WriteString(limit(parameter.Note, 700))
		}
		if len(parameter.Sources) > 0 {
			result.WriteString("\n\n_Source: ")
			result.WriteString(strings.Join(parameter.Sources, ", "))
			result.WriteString("_")
		}
	}
	return result.String(), true
}

type xmlParameter struct {
	Number      string `xml:"number,attr"`
	Type        string `xml:"type,attr"`
	Dimension   string `xml:"dim,attr"`
	ReadOnly    string `xml:"readonly,attr"`
	Name        string `xml:"name"`
	Brief       string `xml:"brief"`
	Description string `xml:"description"`
	Note        string `xml:"note"`
	Attention   string `xml:"attention"`
}

func (catalog *Catalog) loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	decoder.CharsetReader = charsetReader
	fileName := strings.ToLower(filepath.Base(path))
	source := filepath.Base(path)
	infoComment := ""
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "info" {
			infoComment = strings.TrimSpace(attribute(start.Attr, "comment"))
			continue
		}
		if start.Name.Local != "parameter" {
			continue
		}
		var raw xmlParameter
		if err := decoder.DecodeElement(&raw, &start); err != nil {
			return err
		}
		parameter, ok := makeParameter(fileName, source, infoComment, raw)
		if ok {
			catalog.add(parameter)
		}
	}
}

func makeParameter(fileName, source, infoComment string, raw xmlParameter) (Parameter, bool) {
	parameter := Parameter{
		Number: raw.Number, Type: cleanText(raw.Type), Dimension: cleanText(raw.Dimension),
		ReadOnly: strings.EqualFold(raw.ReadOnly, "true"), Name: cleanText(raw.Name),
		Brief: cleanText(raw.Brief), Description: cleanText(raw.Description),
		Note: cleanText(raw.Note), Attention: cleanText(raw.Attention), Sources: []string{source},
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	switch extension {
	case ".svar":
		parameter.Identifier = parameter.Name
	case ".mdat":
		prefix := machineDataPrefix(strings.TrimSuffix(fileName, extension))
		if prefix == "" || parameter.Name == "" {
			return Parameter{}, false
		}
		parameter.Identifier = prefix + "_" + parameter.Name
	case ".para":
		if parameter.Number == "" {
			return Parameter{}, false
		}
		kind := "p"
		if parameter.ReadOnly {
			kind = "r"
		}
		parameter.Identifier = kind + normalizedNumber(parameter.Number)
		if infoComment != "" {
			parameter.Sources = []string{infoComment + " (" + source + ")"}
		}
	default:
		return Parameter{}, false
	}
	return parameter, parameter.Identifier != ""
}

func (catalog *Catalog) add(parameter Parameter) {
	key := canonicalIdentifier(parameter.Identifier)
	existing := catalog.parameters[key]
	for index := range existing {
		if sameDocumentation(existing[index], parameter) {
			existing[index].Sources = append(existing[index].Sources, parameter.Sources...)
			catalog.parameters[key] = existing
			return
		}
	}
	catalog.parameters[key] = append(existing, parameter)
	catalog.count++
}

func sameDocumentation(left, right Parameter) bool {
	return left.Identifier == right.Identifier && left.Number == right.Number &&
		left.Type == right.Type && left.Dimension == right.Dimension &&
		left.ReadOnly == right.ReadOnly && left.Name == right.Name &&
		left.Brief == right.Brief && left.Description == right.Description &&
		left.Note == right.Note && left.Attention == right.Attention
}

func machineDataPrefix(base string) string {
	switch base {
	case "mdnck", "sfnck":
		return "$MN"
	case "mdchan":
		return "$MC"
	case "mdaxis", "sfaxis":
		return "$MA"
	case "cmdnck":
		return "$MNS"
	case "cmdchan":
		return "$MCS"
	case "cmdaxis":
		return "$MAS"
	case "mdhmi":
		return "$MM"
	case "sdnck":
		return "$SN"
	case "sdchan":
		return "$SC"
	case "sdaxis":
		return "$SA"
	case "csdnck":
		return "$SNS"
	case "csdchan":
		return "$SCS"
	case "odnck", "odchan":
		return "$ON"
	default:
		return ""
	}
}

func canonicalIdentifier(identifier string) string {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	if len(identifier) > 1 && (identifier[0] == 'p' || identifier[0] == 'r') {
		if _, err := strconv.ParseUint(identifier[1:], 10, 64); err == nil {
			return identifier[:1] + normalizedNumber(identifier[1:])
		}
	}
	return identifier
}

func normalizedNumber(number string) string {
	number = strings.TrimLeft(number, "0")
	if number == "" {
		return "0"
	}
	return number
}

func cleanText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for index := range lines {
		lines[index] = strings.TrimSpace(lines[index])
	}
	value = strings.TrimSpace(strings.Join(lines, "\n"))
	for strings.Contains(value, "\n\n\n") {
		value = strings.ReplaceAll(value, "\n\n\n", "\n\n")
	}
	return value
}

func limit(value string, maximum int) string {
	if len([]rune(value)) <= maximum {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:maximum])) + "…"
}

func localizedDirectory(directory, locale string) string {
	language := strings.ToLower(locale)
	if cut, _, found := strings.Cut(language, "-"); found {
		language = cut
	}
	if cut, _, found := strings.Cut(language, "_"); found {
		language = cut
	}
	if language == "de" && containsDescriptionFilesDirect(directory) {
		return directory
	}
	if language != "" {
		localized := filepath.Join(directory, language)
		if containsDescriptionFilesDirect(localized) {
			return localized
		}
	}
	english := filepath.Join(directory, "en")
	if containsDescriptionFilesDirect(english) {
		return english
	}
	return directory
}

func containsDescriptionFiles(directory string) bool {
	return containsDescriptionFilesDirect(directory) || containsDescriptionFilesDirect(filepath.Join(directory, "en"))
}

func containsDescriptionFilesDirect(directory string) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && isDescriptionFile(entry.Name()) {
			return true
		}
	}
	return false
}

func isDescriptionFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mdat", ".svar", ".para":
		return true
	default:
		return false
	}
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	charset = strings.ToLower(strings.TrimSpace(charset))
	if charset != "windows-1252" && charset != "cp1252" && charset != "iso-8859-1" {
		return nil, fmt.Errorf("unsupported XML encoding %q", charset)
	}
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	if charset == "iso-8859-1" {
		text, _ := textencoding.Decode(data)
		return strings.NewReader(text), nil
	}
	return strings.NewReader(textencoding.DecodeWindows1252(data)), nil
}

func attribute(attributes []xml.Attr, name string) string {
	for _, attribute := range attributes {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}
