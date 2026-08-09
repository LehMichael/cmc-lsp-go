package lsp

import (
	"fmt"
	"strings"
)

type builtinCallableDocumentation struct {
	Name      string
	Signature string
	Kind      string
	English   string
	German    string
	Manual    string
}

type builtinCallableAlias struct {
	Name        string
	Replacement string
}

func builtinCallableHover(identifier, locale string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(identifier))
	documentation, ok := builtinCallableDocumentationByName[key]
	displayName := documentation.Name
	alias, isAlias := builtinCallableAliases[key]
	if !ok && isAlias {
		documentation, ok = builtinCallableDocumentationByName[strings.ToLower(alias.Replacement)]
		displayName = alias.Name
	}
	if !ok {
		return "", false
	}

	signature := documentation.Signature
	if isAlias {
		signature = displayName + strings.TrimPrefix(signature, documentation.Name)
	}
	description := documentation.English
	kind := documentation.Kind
	if isGermanLocale(locale) {
		description = documentation.German
		kind = germanCallableKind(kind)
	}

	var result strings.Builder
	fmt.Fprintf(&result, "### `%s`\n\n%s", signature, kind)
	if description != "" {
		result.WriteString("\n\n")
		result.WriteString(description)
	}
	if isAlias {
		if isGermanLocale(locale) {
			fmt.Fprintf(&result, "\n\n_Veralteter Kompatibilit\u00e4tsname; stattdessen `%s` verwenden._", documentation.Name)
		} else {
			fmt.Fprintf(&result, "\n\n_Deprecated compatibility name; use `%s` instead._", documentation.Name)
		}
	}
	if documentation.Manual != "" {
		result.WriteString("\n\n_Source: Siemens Create MyConfig manual, ")
		result.WriteString(documentation.Manual)
		result.WriteString("._")
	}
	return result.String(), true
}

func builtinCompletionItems() []completionItem {
	items := make([]completionItem, 0, len(builtinCallables))
	for _, documentation := range builtinCallables {
		hover, _ := builtinCallableHover(documentation.Name, "en")
		items = append(items, completionItem{
			Label:         documentation.Name,
			Kind:          3,
			Detail:        documentation.Signature,
			Documentation: &markupContent{Kind: "markdown", Value: hover},
			InsertText:    documentation.Name + "()",
		})
	}
	return items
}

func isGermanLocale(locale string) bool {
	return strings.HasPrefix(strings.ToLower(locale), "de")
}

func germanCallableKind(kind string) string {
	switch kind {
	case "Function":
		return "Funktion"
	case "Procedure":
		return "Prozedur"
	case "Instruction":
		return "Anweisung"
	case "Standard-library function":
		return "Standardbibliotheksfunktion"
	case "Standard-library procedure":
		return "Standardbibliotheksprozedur"
	default:
		return kind
	}
}

var builtinCallableAliases = map[string]builtinCallableAlias{
	"match":   {Name: "Match", Replacement: "StringMatch"},
	"replace": {Name: "Replace", Replacement: "StringReplace"},
	"exists":  {Name: "Exists", Replacement: "FileExist"},
	"round":   {Name: "Round", Replacement: "MathRound"},
}

var builtinCallableDocumentationByName = func() map[string]builtinCallableDocumentation {
	result := make(map[string]builtinCallableDocumentation, len(builtinCallables))
	for _, documentation := range builtinCallables {
		result[strings.ToLower(documentation.Name)] = documentation
	}
	return result
}()

var builtinCallables = []builtinCallableDocumentation{
	{
		Name:      "CHANDATA",
		Signature: "CHANDATA(<channel>)",
		Kind:      "Instruction",
		English:   "Selects the NC channel for subsequent unqualified NC data access. It is the long form of a `[C<n>]` section selector.",
		German:    "W\u00e4hlt den NC-Kanal f\u00fcr nachfolgende unqualifizierte NC-Datenzugriffe. Dies ist die Langform der Bereichsangabe `[C<n>]`.",
		Manual:    "section 7.8.4",
	},
	{
		Name:      "SinamicsTecActivate",
		Signature: `SinamicsTecActivate("<TECNAME>", <DoVar>)`,
		Kind:      "Standard-library procedure",
		English:   "Activates a SINAMICS technological function on the specified drive object. The technology name must correspond to parameter `p4955[x]`.",
		German:    "Aktiviert eine technologische SINAMICS-Funktion auf dem angegebenen Drive Object. Der Technologiefunktionsname muss dem Parameter `p4955[x]` entsprechen.",
		Manual:    "section 6.6.1.2",
	},
	{
		Name:      "SinamicsTecDeactivate",
		Signature: `SinamicsTecDeactivate("<TECNAME>", <DoVar>)`,
		Kind:      "Standard-library procedure",
		English:   "Deactivates a SINAMICS technological function on the specified drive object.",
		German:    "Deaktiviert eine technologische SINAMICS-Funktion auf dem angegebenen Drive Object.",
		Manual:    "section 6.6.1.3",
	},
	{
		Name:      "IsBit",
		Signature: "IsBit(<bit>, <value>)",
		Kind:      "Standard-library function",
		English:   "Returns `true` when the zero-based bit is set in the supplied value, otherwise `false`.",
		German:    "Gibt `true` zur\u00fcck, wenn das nullbasiert gez\u00e4hlte Bit im angegebenen Wert gesetzt ist, andernfalls `false`.",
		Manual:    "section 6.6.2.2",
	},
	{
		Name:      "SetBit",
		Signature: "SetBit(<bit>, <value>)",
		Kind:      "Standard-library function",
		English:   "Returns the supplied value with the zero-based bit set to `1`.",
		German:    "Gibt den angegebenen Wert mit dem nullbasiert gez\u00e4hlten Bit auf `1` gesetzt zur\u00fcck.",
		Manual:    "section 6.6.2.3",
	},
	{
		Name:      "ClrBit",
		Signature: "ClrBit(<bit>, <value>)",
		Kind:      "Standard-library function",
		English:   "Returns the supplied value with the zero-based bit reset to `0`.",
		German:    "Gibt den angegebenen Wert mit dem nullbasiert gez\u00e4hlten Bit auf `0` zur\u00fcckgesetzt zur\u00fcck.",
		Manual:    "section 6.6.2.4",
	},
	{
		Name:      "StringLen",
		Signature: `StringLen("<string>")`,
		Kind:      "Function",
		English:   "Returns the number of printable and non-printable characters in a string.",
		German:    "Gibt die Anzahl der druckbaren und nicht druckbaren Zeichen einer Zeichenkette zur\u00fcck.",
		Manual:    "section 7.8.13.1",
	},
	{
		Name:      "StringMatch",
		Signature: `StringMatch("<string>", "<search>")`,
		Kind:      "Function",
		English:   "Searches a string using a regular expression and returns the matched text.",
		German:    "Durchsucht eine Zeichenkette mit einem regul\u00e4ren Ausdruck und gibt den gefundenen Text zur\u00fcck.",
		Manual:    "section 7.8.13.2",
	},
	{
		Name:      "StringPos",
		Signature: `StringPos("<string>", "<search>", <pos>)`,
		Kind:      "Function",
		English:   "Searches from a zero-based position using a case-insensitive regular expression. Returns the first match position or `-1`.",
		German:    "Sucht ab einer nullbasierten Position mit einem regul\u00e4ren Ausdruck ohne Beachtung der Gro\u00df-/Kleinschreibung. Gibt die erste Fundstelle oder `-1` zur\u00fcck.",
		Manual:    "section 7.8.13.3",
	},
	{
		Name:      "StringReplace",
		Signature: `StringReplace("<string>", "<search>", "<replace>")`,
		Kind:      "Function",
		English:   "Replaces every match of a regular expression in a string and returns the resulting string.",
		German:    "Ersetzt alle Treffer eines regul\u00e4ren Ausdrucks in einer Zeichenkette und gibt das Ergebnis zur\u00fcck.",
		Manual:    "section 7.8.13.4",
	},
	{
		Name:      "StringSubStr",
		Signature: `StringSubStr("<string>", <pos>, <len>)`,
		Kind:      "Function",
		English:   "Returns a substring beginning at the zero-based position with the requested length, or an empty string when it cannot be obtained.",
		German:    "Gibt eine Teilzeichenkette ab der nullbasierten Position mit der angegebenen L\u00e4nge zur\u00fcck; ist dies nicht m\u00f6glich, wird eine leere Zeichenkette geliefert.",
		Manual:    "section 7.8.13.5",
	},
	{
		Name:      "FileCopy",
		Signature: `FileCopy(<area>, "<sourcepath>", <area>, "<destinationpath>")`,
		Kind:      "Procedure",
		English:   "Copies a file between source and destination areas. An existing destination file is overwritten without prompting.",
		German:    "Kopiert eine Datei zwischen Quell- und Zielbereich. Eine vorhandene Zieldatei wird ohne R\u00fcckfrage \u00fcberschrieben.",
		Manual:    "section 7.8.14.1",
	},
	{
		Name:      "FileDelete",
		Signature: `FileDelete(<area>, "<path>")`,
		Kind:      "Procedure",
		English:   "Deletes the file at the specified path in an area.",
		German:    "L\u00f6scht die Datei am angegebenen Pfad im gew\u00e4hlten Bereich.",
		Manual:    "section 7.8.14.2",
	},
	{
		Name:      "FileExist",
		Signature: `FileExist(<area>, "<path>")`,
		Kind:      "Function",
		English:   "Returns `true` when the folder or file exists at the absolute or package-relative path in the selected area.",
		German:    "Gibt `true` zur\u00fcck, wenn der Ordner oder die Datei unter dem absoluten oder paketrelativen Pfad im gew\u00e4hlten Bereich existiert.",
		Manual:    "section 7.8.14.3",
	},
	{
		Name:      "FileRead",
		Signature: `FileRead(<area>, "<path>")`,
		Kind:      "Function",
		English:   "Reads a text file from an area and returns its contents as a string.",
		German:    "Liest eine Textdatei aus einem Bereich und gibt ihren Inhalt als Zeichenkette zur\u00fcck.",
		Manual:    "section 7.8.14.4",
	},
	{
		Name:      "FileWrite",
		Signature: `FileWrite(<area>, "<path>", "<string>")`,
		Kind:      "Procedure",
		English:   "Writes a string to a new text file in the selected area. The procedure always creates a new file.",
		German:    "Schreibt eine Zeichenkette in eine neue Textdatei im gew\u00e4hlten Bereich. Die Prozedur erzeugt immer eine neue Datei.",
		Manual:    "section 7.8.14.5",
	},
	{
		Name:      "QueryIni",
		Signature: `QueryIni("<path>", "<section/id>")`,
		Kind:      "Function",
		English:   "Reads a value from an INI-format file. Returns `null` when the file, section, or identifier is not found.",
		German:    "Liest einen Wert aus einer Datei im INI-Format. Gibt `null` zur\u00fcck, wenn Datei, Abschnitt oder Bezeichner nicht gefunden werden.",
		Manual:    "section 7.8.14.6",
	},
	{
		Name:      "QueryXml",
		Signature: `QueryXml("<path>", "<xpath>")`,
		Kind:      "Function",
		English:   "Reads the content selected by an XPath expression from an XML file on the package runtime system. Returns `null` when the file or node is not found.",
		German:    "Liest den durch einen XPath-Ausdruck ausgew\u00e4hlten Inhalt aus einer XML-Datei auf dem Laufzeitsystem. Gibt `null` zur\u00fcck, wenn Datei oder Knoten nicht gefunden werden.",
		Manual:    "section 7.8.14.7",
	},
	{
		Name:      "TraceToFile",
		Signature: `TraceToFile("<path>", "<string>")`,
		Kind:      "Procedure",
		English:   "Appends a string to a file, creating the file when it does not yet exist.",
		German:    "H\u00e4ngt eine Zeichenkette an eine Datei an und erzeugt die Datei, falls sie noch nicht existiert.",
		Manual:    "section 7.8.14.8",
	},
	{
		Name:      "AddXmlElement",
		Signature: `AddXmlElement(<area>, "<path>", "<xpath_expression>", "<NewElement>")`,
		Kind:      "Function",
		English:   "Appends a new XML element as the last child of the element selected by XPath. Returns `0` on success or an error code.",
		German:    "F\u00fcgt ein neues XML-Element als letztes Unterelement des per XPath ausgew\u00e4hlten Elements ein. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.16.1",
	},
	{
		Name:      "InsertXmlElement",
		Signature: `InsertXmlElement(<area>, "<path>", "<xpath_expression>", <index>, "<NewElement>")`,
		Kind:      "Function",
		English:   "Inserts a new XML element below the indexed element selected by XPath. An index beyond the matches selects the last position. Returns `0` on success or an error code.",
		German:    "F\u00fcgt ein neues XML-Element unter dem per XPath und Index ausgew\u00e4hlten Element ein. Ein Index au\u00dferhalb der Treffer w\u00e4hlt die letzte Position. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.16.2",
	},
	{
		Name:      "GetXmlElement",
		Signature: `GetXmlElement(<area>, "<path>", "<xpath_expression>")`,
		Kind:      "Function",
		English:   "Returns the complete matching XML elements, including their content, selected by XPath.",
		German:    "Gibt die vollst\u00e4ndigen, per XPath ausgew\u00e4hlten XML-Elemente einschlie\u00dflich ihres Inhalts zur\u00fcck.",
		Manual:    "section 6.16.3",
	},
	{
		Name:      "GetXmlElementText",
		Signature: `GetXmlElementText(<area>, "<path>", "<xpath_expression>")`,
		Kind:      "Function",
		English:   "Returns the text values of all XML elements selected by XPath, separated by line breaks.",
		German:    "Gibt die Textwerte aller per XPath ausgew\u00e4hlten XML-Elemente durch Zeilenumbr\u00fcche getrennt zur\u00fcck.",
		Manual:    "section 6.16.4",
	},
	{
		Name:      "GetXmlAttribute",
		Signature: `GetXmlAttribute(<area>, "<path>", "<xpath_expression>", "<attribute_name>")`,
		Kind:      "Function",
		English:   "Returns the named attribute values from all XML elements selected by XPath, separated by line breaks.",
		German:    "Gibt die Werte des benannten Attributs aus allen per XPath ausgew\u00e4hlten XML-Elementen durch Zeilenumbr\u00fcche getrennt zur\u00fcck.",
		Manual:    "section 6.16.5",
	},
	{
		Name:      "SetXmlElementText",
		Signature: `SetXmlElementText(<area>, "<path>", "<xpath_expression>", "<value>")`,
		Kind:      "Function",
		English:   "Writes text into every XML element selected by XPath. Returns `0` on success or an error code.",
		German:    "Schreibt Text in jedes per XPath ausgew\u00e4hlte XML-Element. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.16.6",
	},
	{
		Name:      "SetXmlAttribute",
		Signature: `SetXmlAttribute(<area>, "<path>", "<xpath_expression>", "<attribute_name>", "<value>")`,
		Kind:      "Function",
		English:   "Sets the named attribute on every XML element selected by XPath, adding the attribute when necessary. Returns `0` on success or an error code.",
		German:    "Setzt das benannte Attribut an jedem per XPath ausgew\u00e4hlten XML-Element und f\u00fcgt es bei Bedarf hinzu. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.16.7",
	},
	{
		Name:      "RemoveXmlElement",
		Signature: `RemoveXmlElement(<area>, "<path>", "<xpath_expression>")`,
		Kind:      "Function",
		English:   "Removes every XML element selected by XPath. Returns `0` on success or an error code.",
		German:    "Entfernt jedes per XPath ausgew\u00e4hlte XML-Element. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.16.8; signature corrected from the manual example",
	},
	{
		Name:      "RemoveXmlAttribute",
		Signature: `RemoveXmlAttribute(<area>, "<path>", "<xpath_expression>", "<attribute_name>")`,
		Kind:      "Function",
		English:   "Removes the named attribute from every XML element selected by XPath. Returns `0` on success or an error code.",
		German:    "Entfernt das benannte Attribut aus jedem per XPath ausgew\u00e4hlten XML-Element. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.16.9",
	},
	{
		Name:      "ConfigDataItemExists",
		Signature: `ConfigDataItemExists("<dbName>", "<itemName>")`,
		Kind:      "Function",
		English:   "Checks whether an item exists in the PLC configuration data. Returns `0` when it exists, otherwise an error code.",
		German:    "Pr\u00fcft, ob ein Element in den PLC-Konfigurationsdaten existiert. Gibt bei vorhandenem Element `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.17.1",
	},
	{
		Name:      "ReadConfigDataItem",
		Signature: `ReadConfigDataItem("<dbName>", "<itemName>")`,
		Kind:      "Function",
		English:   "Returns the content of a PLC configuration-data item. A missing item or unavailable configuration data produces an empty string.",
		German:    "Gibt den Inhalt eines Elements der PLC-Konfigurationsdaten zur\u00fcck. Bei einem fehlenden Element oder nicht verf\u00fcgbaren Konfigurationsdaten wird eine leere Zeichenkette geliefert.",
		Manual:    "section 6.17.2",
	},
	{
		Name:      "SetConfigDataItem",
		Signature: `SetConfigDataItem("<dbName>", "<itemName>", "<itemType>", "<value>")`,
		Kind:      "Function",
		English:   "Changes or creates an item in the working copy of the PLC configuration data. Returns `0` on success or an error code.",
		German:    "\u00c4ndert oder erzeugt ein Element in der Arbeitskopie der PLC-Konfigurationsdaten. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.17.3",
	},
	{
		Name:      "RemoveConfigDataItem",
		Signature: `RemoveConfigDataItem("<dbName>", "<itemName>")`,
		Kind:      "Function",
		English:   "Removes an item from the PLC configuration-data update set without deleting it directly from the PLC. Returns `0` on success or an error code.",
		German:    "Entfernt ein Element aus dem Aktualisierungssatz der PLC-Konfigurationsdaten, ohne es direkt aus der PLC zu l\u00f6schen. Gibt bei Erfolg `0`, andernfalls einen Fehlercode zur\u00fcck.",
		Manual:    "section 6.17.4",
	},
	{
		Name:      "Msg",
		Signature: `Msg("<label>")`,
		Kind:      "Procedure",
		English:   "Displays a message that the operator confirms with OK and also writes the text to the logbook.",
		German:    "Zeigt eine Meldung an, die mit OK best\u00e4tigt wird, und schreibt den Text zus\u00e4tzlich ins Logbuch.",
		Manual:    "section 7.8.15.1",
	},
	{
		Name:      "Warning",
		Signature: `Warning("<label>")`,
		Kind:      "Procedure",
		English:   "Displays a warning and asks whether package execution should be canceled. The text is also written to the logbook.",
		German:    "Zeigt eine Warnung an und fragt, ob die Paketausf\u00fchrung abgebrochen werden soll. Der Text wird auch ins Logbuch geschrieben.",
		Manual:    "section 7.8.15.2",
	},
	{
		Name:      "Error",
		Signature: `Error("<label>")`,
		Kind:      "Procedure",
		English:   "Displays an error that can only be confirmed by canceling the package. The text is also written to the logbook.",
		German:    "Zeigt einen Fehler an, der nur durch Abbruch des Pakets best\u00e4tigt werden kann. Der Text wird auch ins Logbuch geschrieben.",
		Manual:    "section 7.8.15.3",
	},
	{
		Name:      "Input",
		Signature: `Input("<label>")`,
		Kind:      "Function",
		English:   "Prompts for an untyped value during script execution. The entered value determines its type; an empty input returns `null`.",
		German:    "Fragt w\u00e4hrend der Skriptausf\u00fchrung einen untypisierten Wert ab. Die Eingabe bestimmt den Typ; eine leere Eingabe liefert `null`.",
		Manual:    "section 7.8.15.5",
	},
	{
		Name:      "InputChoice",
		Signature: `InputChoice("<label>", "<button1>;<button2>")`,
		Kind:      "Function",
		English:   "Displays a button-choice dialog and returns the label of the selected button as a string.",
		German:    "Zeigt einen Auswahldialog mit Schaltfl\u00e4chen an und gibt die Beschriftung der gew\u00e4hlten Schaltfl\u00e4che als Zeichenkette zur\u00fcck.",
		Manual:    "section 7.8.15.6",
	},
	{
		Name:      "InputEnum",
		Signature: `InputEnum("<label>", "*<enum1>;<enum2>")`,
		Kind:      "Function",
		English:   "Displays a list of string choices and returns the selected string. Prefix an entry with `*` to preselect it.",
		German:    "Zeigt eine Liste von Textoptionen an und gibt den ausgew\u00e4hlten Text zur\u00fcck. Ein vorangestelltes `*` markiert die Vorauswahl.",
		Manual:    "section 7.8.15.7",
	},
	{
		Name:      "InputInt",
		Signature: `InputInt("<label>", <int>)`,
		Kind:      "Function",
		English:   "Prompts for an integer, using the second argument as the default value, and returns the entered number.",
		German:    "Fragt eine Ganzzahl ab, verwendet das zweite Argument als Vorgabewert und gibt die eingegebene Zahl zur\u00fcck.",
		Manual:    "section 7.8.15.8",
	},
	{
		Name:      "InputReal",
		Signature: `InputReal("<label>", <real>)`,
		Kind:      "Function",
		English:   "Prompts for a real number, using the second argument as the default value, and returns the entered number.",
		German:    "Fragt eine Gleitkommazahl ab, verwendet das zweite Argument als Vorgabewert und gibt die eingegebene Zahl zur\u00fcck.",
		Manual:    "section 7.8.15.9",
	},
	{
		Name:      "InputText",
		Signature: `InputText("<label>", "<string>")`,
		Kind:      "Function",
		English:   "Prompts for text, using the second argument as the preselected value, and returns the entered string.",
		German:    "Fragt einen Text ab, verwendet das zweite Argument als vorausgew\u00e4hlten Wert und gibt die eingegebene Zeichenkette zur\u00fcck.",
		Manual:    "section 7.8.15.10",
	},
	{
		Name:      "InputUInt",
		Signature: `InputUInt("<label>", <uint>)`,
		Kind:      "Function",
		English:   "Prompts for an unsigned integer in a supported decimal, hexadecimal, binary, or BiCo format and returns the entered value.",
		German:    "Fragt eine vorzeichenlose Ganzzahl in einem unterst\u00fctzten Dezimal-, Hexadezimal-, Bin\u00e4r- oder BiCo-Format ab und gibt den eingegebenen Wert zur\u00fcck.",
		Manual:    "section 7.8.15.11",
	},
	{
		Name:      "ResFile",
		Signature: `ResFile("<file>")`,
		Kind:      "Function",
		English:   "Loads a localized `.txt`, `.htm`, or `.html` resource file for the package language, falling back to the resource root.",
		German:    "L\u00e4dt eine lokalisierte `.txt`-, `.htm`- oder `.html`-Ressourcendatei f\u00fcr die Paketsprache und verwendet ersatzweise das Ressourcen-Stammverzeichnis.",
		Manual:    "sections 7.8.16 and 7.10.9",
	},
	{
		Name:      "ResText",
		Signature: `ResText("<resid>"[, "<arg>"]*)`,
		Kind:      "Function",
		English:   "Loads localized text by ID from `User.ts`. Up to nine optional arguments replace `%1` through `%9` placeholders.",
		German:    "L\u00e4dt einen lokalisierten Text anhand seiner ID aus `User.ts`. Bis zu neun optionale Argumente ersetzen die Platzhalter `%1` bis `%9`.",
		Manual:    "sections 7.8.16 and 7.10.10",
	},
	{
		Name:      "Skip",
		Signature: "Skip()",
		Kind:      "Procedure",
		English:   "Skips the current dialog, step, or action. In a CMC Diff script it terminates execution like `Return()`.",
		German:    "\u00dcberspringt den aktuellen Dialog, Schritt oder die aktuelle Aktion. In einem CMC-Diff-Skript beendet die Prozedur die Ausf\u00fchrung wie `Return()`.",
		Manual:    "section 7.8.17.1",
	},
	{
		Name:      "Redo",
		Signature: "Redo()",
		Kind:      "Procedure",
		English:   "Repeats dialog input or the complete dialog processing from an `OnNext` or `OnEnd` dialog script.",
		German:    "Wiederholt aus einem `OnNext`- oder `OnEnd`-Dialogskript die Eingaben oder die gesamte Dialogbearbeitung.",
		Manual:    "section 7.8.17.2",
	},
	{
		Name:      "Return",
		Signature: "Return([<value>])",
		Kind:      "Procedure",
		English:   "Terminates the current script or manipulation task. In a user-defined function, the optional value becomes the function result.",
		German:    "Beendet das aktuelle Skript oder den Manipulationsauftrag. In einer benutzerdefinierten Funktion wird der optionale Wert zum Funktionsergebnis.",
		Manual:    "sections 7.8.17.3 and 7.9.2",
	},
	{
		Name:      "ExtCall",
		Signature: `ExtCall("<path>")`,
		Kind:      "Procedure",
		English:   "Calls an external UTF-8 manipulation task and integrates it into package execution. The path may be absolute or relative to the package execution folder.",
		German:    "Ruft einen externen UTF-8-Manipulationsauftrag auf und bindet ihn in die Paketausf\u00fchrung ein. Der Pfad kann absolut oder relativ zum Ausf\u00fchrungsordner des Pakets angegeben werden.",
		Manual:    "section 7.8.17.4",
	},
	{
		Name:      "DateTime",
		Signature: `DateTime("<dt>")`,
		Kind:      "Function",
		English:   "Returns the current date and time formatted according to the supplied date/time pattern.",
		German:    "Gibt das aktuelle Datum und die aktuelle Uhrzeit entsprechend dem angegebenen Datums-/Zeitformat zur\u00fcck.",
		Manual:    "section 7.8.17.5",
	},
	{
		Name:      "DOVar",
		Signature: `DOVar("<doname>" | <axis>)`,
		Kind:      "Function",
		English:   "Creates a DO variable during package execution from either the `p199` DO name or a fixed machine-axis identifier such as `AX1`.",
		German:    "Erzeugt w\u00e4hrend der Paketausf\u00fchrung eine DO-Variable entweder aus dem DO-Namen in `p199` oder aus einem festen Maschinenachsbezeichner wie `AX1`.",
		Manual:    "section 7.8.17.6",
	},
	{
		Name:      "Log",
		Signature: `Log("<label>")`,
		Kind:      "Procedure",
		English:   "Writes user text to the logbook without displaying it during package execution.",
		German:    "Schreibt einen Benutzertext ins Logbuch, ohne ihn w\u00e4hrend der Paketausf\u00fchrung anzuzeigen.",
		Manual:    "section 7.8.17.7",
	},
	{
		Name:      "Logging",
		Signature: "Logging(<switch>)",
		Kind:      "Procedure",
		English:   "Enables or disables logbook logging with `On` or `Off`. Logging is restored to `On` when the script ends.",
		German:    "Aktiviert oder deaktiviert die Logbuchaufzeichnung mit `On` oder `Off`. Am Skriptende wird die Aufzeichnung wieder auf `On` gesetzt.",
		Manual:    "section 7.8.17.8",
	},
	{
		Name:      "MathRound",
		Signature: "MathRound(<value>, <precision>)",
		Kind:      "Function",
		English:   "Rounds a numeric value to the requested number of places and returns the rounded value.",
		German:    "Rundet einen numerischen Wert auf die angegebene Anzahl von Stellen und gibt den gerundeten Wert zur\u00fcck.",
		Manual:    "section 7.8.17.9",
	},
	{
		Name:      "Version",
		Signature: `Version(<area>, "<app>")`,
		Kind:      "Function",
		English:   "Returns the installed version of an application in the `NCU` or `PCU` area, or `null` when it cannot be determined.",
		German:    "Gibt die installierte Version einer Anwendung im Bereich `NCU` oder `PCU` zur\u00fcck; kann sie nicht ermittelt werden, wird `null` geliefert.",
		Manual:    "section 7.8.17.10",
	},
	{
		Name:      "Prepare",
		Signature: `Prepare("<data>", "<value>")`,
		Kind:      "Procedure",
		English:   "Preassigns one of the specially supported machine data values so the package can apply it at the required later point.",
		German:    "Belegt einen der speziell unterst\u00fctzten Maschinendatenwerte vor, damit das Paket ihn zum erforderlichen sp\u00e4teren Zeitpunkt setzen kann.",
		Manual:    "section 7.8.19.1",
	},
	{
		Name:      "Patch",
		Signature: `Patch("<path>")`,
		Kind:      "Procedure",
		English:   "Calls an internal component by component path, optionally qualified with its project name and without a file extension.",
		German:    "Ruft eine interne Komponente \u00fcber ihren Komponentenpfad auf, optional mit Projektname und ohne Dateierweiterung.",
		Manual:    "section 7.8.19.2",
	},
}
