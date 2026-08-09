package lsp

import (
	"fmt"
	"strings"
	"unicode"
)

type systemVariableDocumentation struct {
	Type    string
	Access  string
	English string
	German  string
	Manual  string
}

func systemVariableHover(identifier, locale string) (string, bool) {
	key := canonicalSystemVariable(identifier)
	if !strings.HasPrefix(key, "up.$") {
		return "", false
	}
	documentation, ok := systemVariableDocumentationByName[key]
	if !ok {
		documentation, ok = patternedSystemVariableDocumentation(key)
	}
	if !ok && strings.HasPrefix(key, "up.$dialog.") {
		leaf := key[strings.LastIndex(key, ".")+1:]
		documentation, ok = dialogPropertyDocumentation[leaf]
	}
	if !ok {
		for _, root := range []string{"up.$pack", "up.$dialog", "up.$step", "up.$env", "up.$script", "up.$accesslevelpwdconfig", "up.$basicsecsettings"} {
			if key == root || strings.HasPrefix(key, root+".") || strings.HasPrefix(key, root+"[") {
				documentation, ok = systemVariableDocumentationByName[root]
				break
			}
		}
	}
	if !ok {
		return "", false
	}

	var result strings.Builder
	fmt.Fprintf(&result, "### `%s`", identifier)
	var metadata []string
	if documentation.Type != "" {
		metadata = append(metadata, "Type: `"+documentation.Type+"`")
	}
	if documentation.Access != "" {
		access := documentation.Access
		if isGermanLocale(locale) && access == "Read-only" {
			access = "Schreibgesch\u00fctzt"
		}
		metadata = append(metadata, access)
	}
	if len(metadata) > 0 {
		result.WriteString("\n\n")
		result.WriteString(strings.Join(metadata, " · "))
	}
	description := documentation.English
	if strings.HasPrefix(strings.ToLower(locale), "de") && documentation.German != "" {
		description = documentation.German
	}
	if description != "" {
		result.WriteString("\n\n")
		result.WriteString(description)
	}
	if documentation.Manual != "" {
		result.WriteString("\n\n_Source: Siemens Create MyConfig manual, ")
		result.WriteString(documentation.Manual)
		result.WriteString("._")
	}
	return result.String(), true
}

func systemVariableAt(text string, target position) string {
	lines := strings.Split(text, "\n")
	if target.Line < 0 || target.Line >= len(lines) {
		return ""
	}
	runes := []rune(lines[target.Line])
	column := utf16ToRuneColumn(runes, target.Character)
	if column == len(runes) && column > 0 {
		column--
	}
	allowed := func(value rune) bool {
		return unicode.IsLetter(value) || unicode.IsNumber(value) || strings.ContainsRune("_$.[()]", value)
	}
	start, end := column, column
	for start > 0 && allowed(runes[start-1]) {
		start--
	}
	for end < len(runes) && allowed(runes[end]) {
		end++
	}
	candidate := strings.Trim(string(runes[start:end]), ".")
	if !strings.HasPrefix(strings.ToLower(candidate), "up.$") {
		return ""
	}
	return candidate
}

func canonicalSystemVariable(identifier string) string {
	key := strings.ToLower(strings.TrimSpace(identifier))
	var result strings.Builder
	for start := 0; start < len(key); {
		open := strings.IndexByte(key[start:], '[')
		if open < 0 {
			result.WriteString(key[start:])
			break
		}
		open += start
		result.WriteString(key[start : open+1])
		close := strings.IndexByte(key[open+1:], ']')
		if close < 0 {
			result.WriteString(key[open+1:])
			break
		}
		close += open + 1
		result.WriteByte('?')
		result.WriteByte(']')
		start = close + 1
	}
	return result.String()
}

func patternedSystemVariableDocumentation(key string) (systemVariableDocumentation, bool) {
	if !strings.HasPrefix(key, "up.$dialog.") {
		return systemVariableDocumentation{}, false
	}
	step := strings.Index(key, ".step[?]")
	if step < 0 {
		return systemVariableDocumentation{}, false
	}
	pattern := "up.$dialog.?.step[?]" + key[step+len(".step[?]"):]
	documentation, ok := systemVariableDocumentationByName[pattern]
	return documentation, ok
}

var legacySystemVariableDocumentationByName = map[string]systemVariableDocumentation{
	"up.$pack": {
		English: "Package configuration and runtime metadata. Fields describe the deployed package, selected target and enabled data areas.",
		German:  "Paketkonfiguration und Laufzeitmetadaten. Die Felder beschreiben das bereitgestellte Paket, das Zielsystem und die aktivierten Datenbereiche.",
		Manual:  "section 7.11.2",
	},
	"up.$pack.userversion":   {Type: "STRING", Access: "Read-only", English: "User-defined package version configured in Expert or supplied by an `.upcfg` file.", German: "Benutzerdefinierte Paketversion aus Expert oder einer `.upcfg`-Datei.", Manual: "table 7-44"},
	"up.$pack.deployname":    {Type: "STRING", Access: "Read-only", English: "Name used when the package is deployed.", German: "Name, unter dem das Paket bereitgestellt wird.", Manual: "table 7-44"},
	"up.$pack.deploydir":     {Type: "STRING", Access: "Read-only", English: "Destination directory used when the package is deployed.", German: "Zielverzeichnis, in dem das Paket bereitgestellt wird.", Manual: "table 7-44"},
	"up.$pack.deploytarget":  {Type: "DeployTarget", Access: "Read-only", English: "Runtime system for which the package is deployed.", German: "Laufzeitsystem, für das das Paket bereitgestellt wird.", Manual: "table 7-44"},
	"up.$pack.deployexclude": {Type: "BOOL", Access: "Read-only", English: "Whether configured data areas are excluded during deployment.", German: "Gibt an, ob konfigurierte Datenbereiche bei der Bereitstellung ausgeschlossen werden.", Manual: "table 7-44"},
	"up.$pack.language":      {Type: "Language", Access: "Read-only", English: "Language selected for dialog pages.", German: "Für Dialogseiten ausgewählte Sprache.", Manual: "table 7-44"},
	"up.$pack.prodversion":   {Type: "VERSION", Access: "Read-only", English: "Version of the CMC software executing the package.", German: "Version der CMC-Software, die das Paket ausführt.", Manual: "table 7-44"},
	"up.$pack.arc":           {Type: "BOOL", Access: "Read-only", English: "Whether the Archive data area is used.", German: "Gibt an, ob der Archiv-Datenbereich verwendet wird.", Manual: "table 7-44"},
	"up.$pack.pcu":           {Type: "BOOL", English: "Whether the PCU data area is used. It may only be deselected through a configuration file.", German: "Gibt an, ob der PCU-Datenbereich verwendet wird. Er kann nur über eine Konfigurationsdatei abgewählt werden.", Manual: "table 7-44"},
	"up.$pack.ncu":           {Type: "BOOL", English: "Whether the NCU/PPU data area is used. It may only be deselected through a configuration file.", German: "Gibt an, ob der NCU/PPU-Datenbereich verwendet wird. Er kann nur über eine Konfigurationsdatei abgewählt werden.", Manual: "table 7-44"},
	"up.$pack.name":          {Type: "STRING", Access: "Read-only", English: "File-system name of the package that was started.", German: "Dateisystemname des gestarteten Pakets.", Manual: "table 7-44"},
	"up.$pack.dir":           {Type: "STRING", Access: "Read-only", English: "Directory containing the package that was started.", German: "Verzeichnis des gestarteten Pakets.", Manual: "table 7-44"},

	"up.$dialog": {
		English: "Configuration and state for CMC dialog pages. The next path segment is the dialog ID; its properties control processing and dialog-specific inputs.",
		German:  "Konfiguration und Zustand der CMC-Dialogseiten. Das nächste Pfadsegment ist die Dialog-ID; deren Eigenschaften steuern die Verarbeitung und dialogspezifische Eingaben.",
		Manual:  "section 7.11.3",
	},
	"up.$dialog.processmodes.automatic": {Type: "ProcessMode", Access: "Read-only", English: "Automatic dialog processing mode.", German: "Automatischer Dialog-Verarbeitungsmodus.", Manual: "section 7.11.3.19"},
	"up.$dialog.processmodes.manual":    {Type: "ProcessMode", Access: "Read-only", English: "Manual dialog processing mode.", German: "Manueller Dialog-Verarbeitungsmodus.", Manual: "section 7.11.3.19"},
	"up.$dialog.processmodes.progress":  {Type: "ProcessMode", Access: "Read-only", English: "Progress dialog processing mode.", German: "Dialog-Verarbeitungsmodus für Fortschrittsanzeigen.", Manual: "section 7.11.3.19"},
	"up.$dialog.interactlevels.all":     {Type: "InteractLevel", Access: "Read-only", English: "Display all messages.", German: "Alle Meldungen anzeigen.", Manual: "section 7.11.3.19"},
	"up.$dialog.interactlevels.error":   {Type: "InteractLevel", Access: "Read-only", English: "Display error messages only.", German: "Nur Fehlermeldungen anzeigen.", Manual: "section 7.11.3.19"},
	"up.$dialog.interactlevels.warning": {Type: "InteractLevel", Access: "Read-only", English: "Display errors and warnings.", German: "Fehler und Warnungen anzeigen.", Manual: "section 7.11.3.19"},

	"up.$step": {
		English: "Accesses a step-tree item by ID. Query the object against `null`, or use its properties to inspect and control activation and presentation.",
		German:  "Greift über die ID auf einen Schrittbaumeintrag zu. Das Objekt kann gegen `null` geprüft werden; seine Eigenschaften steuern Aktivierung und Darstellung.",
		Manual:  "section 7.11.4",
	},
	"up.$step[?]":            {Type: "OBJECT", Access: "Read-only", English: "Returns the step with the specified ID, or `null` when that step does not exist.", German: "Liefert den Schritt mit der angegebenen ID oder `null`, wenn er nicht existiert.", Manual: "table 7-63"},
	"up.$step[?].activated":  {Type: "BOOL", English: "Activation state of the step. `true` means its checkbox or option button is selected.", German: "Aktivierungszustand des Schritts. `true` bedeutet, dass Kontrollkästchen oder Optionsfeld ausgewählt sind.", Manual: "table 7-63"},
	"up.$step[?].collapsed":  {Type: "BOOL", English: "Whether substeps are collapsed and hidden from the operator.", German: "Gibt an, ob Unterschritte eingeklappt und für den Bediener verborgen sind.", Manual: "table 7-63"},
	"up.$step[?].locked":     {Type: "BOOL", English: "Whether activation of the step is disabled.", German: "Gibt an, ob die Aktivierung des Schritts gesperrt ist.", Manual: "table 7-63"},
	"up.$step[?].processing": {Type: "BOOL", Access: "Read-only", English: "Execution state of the step. `true` means the activated step is on the green execution track.", German: "Ausführungszustand des Schritts. `true` bedeutet, dass der aktivierte Schritt auf der grünen Ausführungsspur liegt.", Manual: "table 7-63"},

	"up.$env": {
		Access:  "Read-only",
		English: "Information supplied by the environment of the running package.",
		German:  "Informationen aus der Umgebung des laufenden Pakets.",
		Manual:  "section 7.11.5",
	},
	"up.$env.runtime":          {Type: "RunTimes", Access: "Read-only", English: "Runtime environment in which the package is executing.", German: "Laufzeitumgebung, in der das Paket ausgeführt wird.", Manual: "section 7.11.5"},
	"up.$env.ncu":              {Type: "STRING", Access: "Read-only", English: "NCU/PPU hardware type read from `hwversion.xml`.", German: "NCU/PPU-Hardwaretyp aus `hwversion.xml`.", Manual: "section 7.11.5"},
	"up.$env.plc":              {Type: "STRING", Access: "Read-only", English: "PLC hardware type read from `hwversion.xml`.", German: "PLC-Hardwaretyp aus `hwversion.xml`.", Manual: "section 7.11.5"},
	"up.$env.cfid":             {Type: "STRING", Access: "Read-only", English: "CompactFlash card identifier read from `hwversion.xml`.", German: "CompactFlash-Kartenkennung aus `hwversion.xml`.", Manual: "section 7.11.5"},
	"up.$env.batchmode":        {Type: "BOOL", Access: "Read-only", English: "Whether package or script execution was started in command-line batch mode.", German: "Gibt an, ob Paket oder Skript im Kommandozeilen-Batchmodus gestartet wurden.", Manual: "section 7.11.5"},
	"up.$env.runtimes.linux":   {Type: "RunTimes", Access: "Read-only", English: "Package execution under Linux.", German: "Paketausführung unter Linux.", Manual: "section 7.11.5.1"},
	"up.$env.runtimes.windows": {Type: "RunTimes", Access: "Read-only", English: "Package execution under Windows.", German: "Paketausführung unter Windows.", Manual: "section 7.11.5.1"},

	"up.$script": {
		English: "Execution context for a CMC Diff script, including its input archive and result files.",
		German:  "Ausführungskontext eines CMC-Diff-Skripts einschließlich Eingangsarchiv und Ergebnisdateien.",
		Manual:  "sections 6.3.8.5-6.3.8.6",
	},
	"up.$script.scriptfile": {Type: "STRING", Access: "Read-only", English: "Absolute path of the CMC script file being executed.", German: "Absoluter Pfad der ausgeführten CMC-Skriptdatei.", Manual: "section 6.3.8.6"},
	"up.$script.arcfile":    {Type: "STRING", Access: "Read-only", English: "Absolute path of the archive being processed when Diff is started from the command line.", German: "Absoluter Pfad des verarbeiteten Archivs beim Kommandozeilenstart von Diff.", Manual: "section 6.3.8.6"},
	"up.$script.result":     {Type: "STRING", English: "Semicolon-separated relative paths of files that Diff should return to its caller after script execution.", German: "Durch Semikolon getrennte relative Pfade der Dateien, die Diff nach der Skriptausführung zurückgeben soll.", Manual: "section 6.3.8.6"},
	"up.$script.editmode":   {Type: "BOOL", Access: "Read-only", English: "Whether CMC Diff is running in edit mode.", German: "Gibt an, ob CMC Diff im Bearbeitungsmodus läuft.", Manual: "section 6.3.8.6"},
}

var dialogPropertyDocumentation = map[string]systemVariableDocumentation{
	"activated":     {Type: "BOOL", English: "Whether processing of this dialog page is enabled.", German: "Gibt an, ob die Verarbeitung dieser Dialogseite aktiviert ist.", Manual: "section 7.11.3"},
	"processmode":   {Type: "ProcessMode", English: "Processing mode used for this dialog page.", German: "Verarbeitungsmodus dieser Dialogseite.", Manual: "section 7.11.3"},
	"interactlevel": {Type: "InteractLevel", English: "Message level used while processing this dialog page.", German: "Meldungsstufe bei der Verarbeitung dieser Dialogseite.", Manual: "section 7.11.3"},
	"cfgfile":       {Type: "STRING", English: "Configuration file selected for this dialog or step tree.", German: "Für diesen Dialog oder Schrittbaum ausgewählte Konfigurationsdatei.", Manual: "sections 7.11.3.1 and 7.11.3.16"},
	"address":       {Type: "STRING", English: "Network name, address or IP address of the target system.", German: "Netzwerkname, Adresse oder IP-Adresse des Zielsystems.", Manual: "sections 7.11.3.4 and 7.11.3.6"},
	"username":      {Type: "STRING", English: "User name used to log in to the target system.", German: "Benutzername für die Anmeldung am Zielsystem.", Manual: "sections 7.11.3.4 and 7.11.3.6"},
	"password":      {Type: "STRING", English: "Password used to log in to the target system.", German: "Passwort für die Anmeldung am Zielsystem.", Manual: "sections 7.11.3.4 and 7.11.3.6"},
	"archivein":     {Type: "STRING", English: "Path of the input archive.", German: "Pfad des Eingangsarchivs.", Manual: "section 7.11.3.5"},
	"archiveout":    {Type: "STRING", English: "Path of the output archive.", German: "Pfad des Ausgangsarchivs.", Manual: "section 7.11.3.5"},
	"ncfile":        {Type: "STRING", English: "Path of the NC data file.", German: "Pfad der NC-Datendatei.", Manual: "sections 7.11.3.5 and 7.11.3.11"},
	"plcfile":       {Type: "STRING", English: "Path of the PLC data file.", German: "Pfad der PLC-Datendatei.", Manual: "sections 7.11.3.5 and 7.11.3.11"},
	"sdbfile":       {Type: "STRING", English: "Path of the SDB data file.", German: "Pfad der SDB-Datendatei.", Manual: "section 7.11.3.11"},
	"drvfile":       {Type: "STRING", English: "Path of the drive data file.", German: "Pfad der Antriebsdatendatei.", Manual: "sections 7.11.3.5 and 7.11.3.11"},
	"ustfile":       {Type: "STRING", English: "Path of the comparison-topology `.ust` file.", German: "Pfad der `.ust`-Datei für die Vergleichstopologie.", Manual: "section 7.11.3.13"},
	"utzfile":       {Type: "STRING", English: "Path of the user-specified-topology `.utz` file.", German: "Pfad der `.utz`-Datei für die benutzerdefinierte Topologie.", Manual: "section 7.11.3.13"},
	"backup":        {Type: "BOOL", English: "Whether a complete TGZ data backup is created.", German: "Gibt an, ob eine vollständige TGZ-Datensicherung erstellt wird.", Manual: "sections 7.11.3.7 and 7.11.3.17"},
	"archive":       {Type: "BOOL", English: "Whether an NC/PLC/drive archive is created.", German: "Gibt an, ob ein NC/PLC/Antriebsarchiv erstellt wird.", Manual: "sections 7.11.3.7 and 7.11.3.17"},
	"logdir":        {Type: "STRING", English: "Directory in which the package logbook is stored.", German: "Verzeichnis, in dem das Paketlogbuch gespeichert wird.", Manual: "section 7.11.3.18"},
	"logname":       {Type: "STRING", English: "File name of the package logbook.", German: "Dateiname des Paketlogbuchs.", Manual: "section 7.11.3.18"},
}
