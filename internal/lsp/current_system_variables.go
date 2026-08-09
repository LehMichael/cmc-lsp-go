package lsp

import "strings"

var systemVariableDocumentationByName = mergeSystemVariableDocumentation()

func mergeSystemVariableDocumentation() map[string]systemVariableDocumentation {
	result := make(map[string]systemVariableDocumentation, len(legacySystemVariableDocumentationByName)+len(currentSystemVariableDocumentationByName))
	for name, documentation := range legacySystemVariableDocumentationByName {
		result[name] = documentation
	}
	for name, documentation := range currentSystemVariableDocumentationByName {
		result[name] = documentation
	}
	return result
}

var currentSystemVariableDocumentationByName = buildCurrentSystemVariableDocumentation()

func buildCurrentSystemVariableDocumentation() map[string]systemVariableDocumentation {
	result := map[string]systemVariableDocumentation{}
	add := func(name, valueType, access, english, german, manual string) {
		result[strings.ToLower(name)] = systemVariableDocumentation{
			Name: name, Type: valueType, Access: access, English: english, German: german, Manual: manual,
		}
	}
	readOnly := "Read-only"

	add("Up.$Pack", "OBJECT", "", "Current package configuration and runtime metadata.", "Aktuelle Paketkonfiguration und Laufzeitmetadaten.", "section 8.9.2")
	add("Up.$Pack.DeployName", "STRING", readOnly, "Package name defined in the configuration and displayed in the title bar during Windows execution.", "In der Konfiguration festgelegter Paketname, der bei der Ausf\u00fchrung unter Windows in der Titelleiste angezeigt wird.", "table 8-20")
	add("Up.$Pack.DeployDir", "STRING", "", "Directory in which CMC Expert places the deployed package.", "Verzeichnis, in dem CMC Expert das bereitgestellte Paket ablegt.", "table 8-20")
	add("Up.$Pack.ProdVersion", "VERSION", readOnly, "Version of the CMC software; this value is not configurable.", "Version der CMC-Software; dieser Wert ist nicht konfigurierbar.", "table 8-20")
	add("Up.$Pack.UserVersion", "STRING", "", "User-defined package version displayed in the Windows title bar; it can also be changed in the `.upcfg` file.", "Benutzerdefinierte Paketversion in der Windows-Titelleiste; sie kann auch in der `.upcfg`-Datei ge\u00e4ndert werden.", "table 8-20")
	add("Up.$Pack.DeployTarget", "DeployTarget", "", "Runtime system for which the package is generated.", "Laufzeitsystem, f\u00fcr das das Paket erzeugt wird.", "table 8-20")
	add("Up.$Pack.ARC", "BOOL", "", "Whether the package uses the archive data area. It cannot be selected later in the deployed package.", "Gibt an, ob das Paket den Archiv-Datenbereich verwendet. Dieser kann im bereitgestellten Paket nicht nachtr\u00e4glich ausgew\u00e4hlt werden.", "table 8-20")
	add("Up.$Pack.NCU", "BOOL", "", "Whether the package uses the NCU/PPU data area. It cannot be selected later in the deployed package.", "Gibt an, ob das Paket den NCU/PPU-Datenbereich verwendet. Dieser kann im bereitgestellten Paket nicht nachtr\u00e4glich ausgew\u00e4hlt werden.", "table 8-20")
	add("Up.$Pack.PCU", "BOOL", "", "Whether the package uses the PCU data area. It cannot be selected later in the deployed package.", "Gibt an, ob das Paket den PCU-Datenbereich verwendet. Dieser kann im bereitgestellten Paket nicht nachtr\u00e4glich ausgew\u00e4hlt werden.", "table 8-20")
	add("Up.$Pack.HmiDataHandling", "HmiDataHandling", readOnly, "Selected source of the HMI data.", "Ausgew\u00e4hlte Quelle der HMI-Daten.", "table 8-20")
	add("Up.$Pack.Name", "STRING", readOnly, "File-system name of the package that was started.", "Dateisystemname des gestarteten Pakets.", "table 8-20")
	add("Up.$Pack.Dir", "STRING", readOnly, "Directory containing the package that was started.", "Verzeichnis des gestarteten Pakets.", "table 8-20")
	add("Up.$Pack.DeployTargets", "DeployTarget", readOnly, "Runtime-system enumeration values.", "Aufz\u00e4hlungswerte f\u00fcr das Laufzeitsystem.", "section 8.9.3")
	add("Up.$Pack.DeployTargets.LINUX", "DeployTarget", readOnly, "Generate and execute the package for Linux on an NCU.", "Paket f\u00fcr Linux auf einer NCU erzeugen und ausf\u00fchren.", "section 8.9.3")
	add("Up.$Pack.DeployTargets.WINDOWS", "DeployTarget", readOnly, "Generate and execute the package for Microsoft Windows on a PC or IPC.", "Paket f\u00fcr Microsoft Windows auf einem PC oder IPC erzeugen und ausf\u00fchren.", "section 8.9.3")
	add("Up.$Pack.HmiDataHandlings", "HmiDataHandling", readOnly, "HMI-data source enumeration values.", "Aufz\u00e4hlungswerte f\u00fcr die Herkunft der HMI-Daten.", "section 8.9.4")
	add("Up.$Pack.HmiDataHandlings.UseHmiArchive", "HmiDataHandling", readOnly, "Use the HMI data from the archive.", "HMI-Daten aus dem Archiv verwenden.", "section 8.9.4")
	add("Up.$Pack.HmiDataHandlings.UseOperateNodeInActions", "HmiDataHandling", readOnly, "Use Operate nodes supplied by actions.", "Operate-Knoten aus Aktionen verwenden.", "section 8.9.4")

	add("Up.$Dialog", "OBJECT", "", "Configuration and state for the current CMC Expert dialog pages.", "Konfiguration und Zustand der aktuellen CMC-Expert-Dialogseiten.", "section 8.9.5")
	type dialogDefinition struct {
		name, section, english, german string
		activated, activatedReadOnly   bool
	}
	dialogs := []dialogDefinition{
		{name: "PackageConfig", section: "8.9.5.1", english: "Package configuration", german: "Paketkonfiguration", activated: true, activatedReadOnly: true},
		{name: "ConfirmNotes", section: "8.9.5.2", english: "Notes about acknowledgment", german: "Hinweise zur Best\u00e4tigung", activated: true},
		{name: "PackageNotes", section: "8.9.5.3", english: "Notes on the package", german: "Hinweise zum Paket", activated: true},
		{name: "AccessData", section: "8.9.5.4", english: "Access to target system", german: "Zugriff auf Zielsystem"},
		{name: "ArcSelection", section: "8.9.5.5", english: "Select offline archive", german: "Auswahl Offline-Archiv"},
		{name: "NcuOrigin", section: "8.9.5.6", english: "Backup initial state", german: "Sicherung Ausgangszustand", activated: true},
		{name: "PcuSetup", section: "8.9.5.7", english: "PCU setup", german: "PCU-Setup", activated: true},
		{name: "NcuSetup", section: "8.9.5.8", english: "CNC software", german: "CNC-Software", activated: true},
		{name: "PlcConfig", section: "8.9.5.9", english: "PLC configuration", german: "PLC-Konfiguration"},
		{name: "SystemConfig", section: "8.9.5.10", english: "System configuration", german: "Systemkonfiguration"},
		{name: "DriveTopology", section: "8.9.5.11", english: "SINAMICS topology", german: "SINAMICS-Topologie", activated: true},
		{name: "VersionView", section: "8.9.5.12", english: "Version Display", german: "Versionsanzeige", activated: true},
		{name: "StepSelection", section: "8.9.5.13", english: "Step tree", german: "Schrittbaum", activated: true},
		{name: "ArcFileInstall", section: "8.9.5.14", english: "Language and install archive", german: "Sprach- und Install-Archiv", activated: true},
		{name: "NcuResult", section: "8.9.5.17", english: "Backup result state", german: "Sicherung Ergebniszustand", activated: true},
		{name: "PackageEnd", section: "8.9.5.18", english: "Finish", german: "Ende"},
	}
	for _, dialog := range dialogs {
		prefix := "Up.$Dialog." + dialog.name
		add(prefix, "OBJECT", "", dialog.english+" dialog system variables.", "Systemvariablen des Dialogs "+dialog.german+".", "section "+dialog.section)
		add(prefix+".ProcessMode", "ProcessMode", "", "Processing mode for this dialog.", "Bearbeitungsmodus dieses Dialogs.", "section "+dialog.section)
		if dialog.activated {
			access := ""
			if dialog.activatedReadOnly {
				access = readOnly
			}
			add(prefix+".Activated", "BOOL", access, "Whether processing of this dialog is enabled.", "Gibt an, ob die Bearbeitung dieses Dialogs aktiviert ist.", "section "+dialog.section)
		}
	}

	add("Up.$Dialog.PackageConfig.CfgFile", "STRING", "", "Configuration file used as the package preselection.", "Konfigurationsdatei f\u00fcr die Paketvorbelegung.", "section 8.9.5.1")
	add("Up.$Dialog.AccessData.Target", "AccessData", "", "Selected target system for package access.", "Ausgew\u00e4hltes Zielsystem f\u00fcr den Paketzugriff.", "section 8.9.5.4")
	add("Up.$Dialog.AccessData.NcuAddress", "STRING", "", "Network name or IP address of the NCU/PPU.", "Netzwerkname oder IP-Adresse der NCU/PPU.", "section 8.9.5.4")
	add("Up.$Dialog.AccessData.IpcAddress", "STRING", "", "Address of the IPC/PCU.", "Adresse des IPC beziehungsweise der PCU.", "section 8.9.5.4")
	add("Up.$Dialog.ArcSelection.ArchiveIn", "STRING", "", "Path of the input archive.", "Pfad des Eingangsarchivs.", "section 8.9.5.5")
	add("Up.$Dialog.ArcSelection.NcFile", "STRING", "", "Path of the NC data.", "Pfad der NC-Daten.", "section 8.9.5.5")
	add("Up.$Dialog.ArcSelection.PlcFile", "STRING", "", "Path of the PLC data.", "Pfad der PLC-Daten.", "section 8.9.5.5")
	add("Up.$Dialog.ArcSelection.DrvFile", "STRING", "", "Path of the drive data.", "Pfad der Antriebsdaten.", "section 8.9.5.5")
	add("Up.$Dialog.ArcSelection.HmiFile", "STRING", "", "Path of the archive containing HMI data.", "Pfad des Archivs mit HMI-Daten.", "section 8.9.5.5")
	add("Up.$Dialog.ArcSelection.SysFile", "STRING", "", "Path of the archive containing system settings.", "Pfad des Archivs mit Systemeinstellungen.", "section 8.9.5.5")
	add("Up.$Dialog.ArcSelection.ArchiveOut", "STRING", "", "Path of the output archive.", "Pfad des Ausgangsarchivs.", "section 8.9.5.5")

	addBackupDialogDocumentation(add, "NcuOrigin", "before package execution", "vor der Paketausf\u00fchrung", "8.9.5.6")
	add("Up.$Dialog.NcuSetup.Mode", "Mode", "", "Selected CNC software installation mode.", "Ausgew\u00e4hlter Installationsmodus der CNC-Software.", "section 8.9.5.8")
	add("Up.$Dialog.NcuSetup.TgzFile", "STRING", "", "Preselected CNC software `.tgz` file.", "Vorausgew\u00e4hlte `.tgz`-Datei der CNC-Software.", "section 8.9.5.8")
	add("Up.$Dialog.NcuSetup.AddTgzFilesSelection", "STRING", "", "Semicolon-separated additional `.tgz` files selected at runtime; `*` selects all transferred files.", "Durch Semikolon getrennte, zur Laufzeit ausgew\u00e4hlte zus\u00e4tzliche `.tgz`-Dateien; `*` w\u00e4hlt alle \u00fcbertragenen Dateien.", "section 8.9.5.8")
	add("Up.$Dialog.PlcConfig.PlcSource", "PlcSource", "", "Selected source of the PLC data.", "Ausgew\u00e4hlte Herkunft der PLC-Daten.", "section 8.9.5.9")
	add("Up.$Dialog.PlcConfig.PlcFile", "STRING", "", "Path of the PLC data file.", "Pfad der PLC-Datendatei.", "section 8.9.5.9")
	add("Up.$Dialog.PlcConfig.ConfigDataItemsSource", "ConfigDataItemsSource", "", "Selected source of the PLC configuration data.", "Ausgew\u00e4hlte Herkunft der PLC-Konfigurationsdaten.", "section 8.9.5.9")
	add("Up.$Dialog.PlcConfig.ConfigDataItemsFile", "STRING", "", "Path of the PLC configuration-data file.", "Pfad der Datei mit PLC-Konfigurationsdaten.", "section 8.9.5.9")
	add("Up.$Dialog.PlcConfig.ClearPLCAccessLevelPwd", "BOOL", "", "Remove the current PLC access-level password from SINUMERIK Operate. This can only be changed by a runtime script.", "Entfernt das aktuelle PLC-Zugriffsstufenpasswort aus SINUMERIK Operate. Der Wert kann nur durch ein Skript zur Laufzeit ge\u00e4ndert werden.", "section 8.9.5.9")
	add("Up.$Dialog.PlcConfig.ClearPLCConfigurationPwd", "BOOL", "", "Remove the current PLC configuration-data password from SINUMERIK Operate. This can only be changed by a runtime script.", "Entfernt das aktuelle Passwort f\u00fcr PLC-Konfigurationsdaten aus SINUMERIK Operate. Der Wert kann nur durch ein Skript zur Laufzeit ge\u00e4ndert werden.", "section 8.9.5.9")
	add("Up.$Dialog.SystemConfig.NcSource", "NcSource", "", "Selected source of the NC data.", "Ausgew\u00e4hlte Herkunft der NC-Daten.", "section 8.9.5.10")
	add("Up.$Dialog.SystemConfig.NcFile", "STRING", "", "Path of the NC data file.", "Pfad der NC-Datendatei.", "section 8.9.5.10")
	add("Up.$Dialog.SystemConfig.DrvSource", "DrvSource", "", "Selected source of the drive data.", "Ausgew\u00e4hlte Herkunft der Antriebsdaten.", "section 8.9.5.10")
	add("Up.$Dialog.SystemConfig.DrvFile", "STRING", "", "Path of the drive data file.", "Pfad der Antriebsdatendatei.", "section 8.9.5.10")
	add("Up.$Dialog.SystemConfig.HmiSource", "HmiSource", "", "Selected source of the HMI data.", "Ausgew\u00e4hlte Herkunft der HMI-Daten.", "section 8.9.5.10")
	add("Up.$Dialog.SystemConfig.HmiFile", "STRING", "", "Path of the archive containing HMI data.", "Pfad des Archivs mit HMI-Daten.", "section 8.9.5.10")
	add("Up.$Dialog.SystemConfig.SysSource", "SysSource", "", "Selected source of the system settings.", "Ausgew\u00e4hlte Herkunft der Systemeinstellungen.", "section 8.9.5.10")
	add("Up.$Dialog.SystemConfig.SysFile", "STRING", "", "Path of the archive containing system settings.", "Pfad des Archivs mit Systemeinstellungen.", "section 8.9.5.10")
	add("Up.$Dialog.DriveTopology.AxisDriveAssignment", "BOOL", "", "Whether axis-to-drive assignment is selected.", "Gibt an, ob die Achs-Antriebs-Zuordnung ausgew\u00e4hlt ist.", "section 8.9.5.11")
	add("Up.$Dialog.DriveTopology.UstFile", "STRING", "", "Path of the comparison-topology `.ust` file.", "Pfad der `.ust`-Datei f\u00fcr die Vergleichstopologie.", "section 8.9.5.11")
	add("Up.$Dialog.DriveTopology.UtzFile", "STRING", "", "Path of the user-specified-topology `.utz2` file.", "Pfad der `.utz2`-Datei f\u00fcr die benutzerdefinierte Topologie.", "section 8.9.5.11")
	add("Up.$Dialog.DriveTopology.MclFile", "STRING", "", "Path of the machine-concept-list `.mcl` file.", "Pfad der Maschinenkonzeptlisten-Datei `.mcl`.", "section 8.9.5.11")
	add("Up.$Dialog.StepSelection.ArchiveBeg", "BOOL", "", "Create a backup archive of the NC, drive, HMI, and system-settings areas before the step tree.", "Erzeugt vor dem Schrittbaum ein Sicherungsarchiv der NC-, Antriebs-, HMI- und Systemeinstellungsbereiche.", "section 8.9.5.13")
	add("Up.$Dialog.StepSelection.ArchiveEnd", "BOOL", "", "Create a backup archive of the NC, drive, HMI, and system-settings areas after the step tree.", "Erzeugt nach dem Schrittbaum ein Sicherungsarchiv der NC-, Antriebs-, HMI- und Systemeinstellungsbereiche.", "section 8.9.5.13")
	add("Up.$Dialog.StepSelection.ArchiveSkipHMI", "BOOL", "", "Create the step-tree archive without its HMI portion.", "Erzeugt das Schrittbaumarchiv ohne HMI-Anteil.", "section 8.9.5.13")
	add("Up.$Dialog.ArcFileInstall.Entry[?]", "OBJECT", "", "Language-archive entry selected by ID.", "\u00dcber die ID ausgew\u00e4hlter Spracharchiveintrag.", "section 8.9.5.14")
	add("Up.$Dialog.ArcFileInstall.Entry[?].File", "STRING", "", "Path of the language archive file.", "Pfad der Spracharchivdatei.", "section 8.9.5.14")
	add("Up.$Dialog.ArcFileInstall.Entry[?].Install", "BOOL", "", "Whether this language archive is installed.", "Gibt an, ob dieses Spracharchiv installiert wird.", "section 8.9.5.14")
	addBackupDialogDocumentation(add, "NcuResult", "after package execution", "nach der Paketausf\u00fchrung", "8.9.5.17")
	add("Up.$Dialog.PackageEnd.LogDir", "STRING", "", "Directory in which the package logbook is stored.", "Verzeichnis, in dem das Paketlogbuch gespeichert wird.", "section 8.9.5.18")
	add("Up.$Dialog.PackageEnd.LogName", "STRING", "", "File name of the package logbook.", "Dateiname des Paketlogbuchs.", "section 8.9.5.18")

	add("Up.$AccessLevelPWDConfig", "OBJECT", "", "System variables for the SINUMERIK access-level-password dialog.", "Systemvariablen des Dialogs f\u00fcr SINUMERIK-Zugriffsstufenpassw\u00f6rter.", "section 8.9.5.15")
	add("Up.$AccessLevelPWDConfig.Activated", "BOOL", "", "Whether processing of the access-level-password dialog is enabled.", "Gibt an, ob die Bearbeitung des Zugriffsstufenpasswort-Dialogs aktiviert ist.", "section 8.9.5.15")
	add("Up.$AccessLevelPWDConfig.ProcessMode", "ProcessMode", "", "Processing mode for the access-level-password dialog.", "Bearbeitungsmodus des Zugriffsstufenpasswort-Dialogs.", "section 8.9.5.15")
	add("Up.$AccessLevelPWDConfig.ManufactFile", "STRING", "", "Path of the manufacturer access-level password file.", "Pfad der Passwortdatei f\u00fcr die Hersteller-Zugriffsstufe.", "section 8.9.5.15")
	add("Up.$AccessLevelPWDConfig.ServiceFile", "STRING", "", "Path of the service access-level password file.", "Pfad der Passwortdatei f\u00fcr die Service-Zugriffsstufe.", "section 8.9.5.15")
	add("Up.$AccessLevelPWDConfig.UserFile", "STRING", "", "Path of the user access-level password file.", "Pfad der Passwortdatei f\u00fcr die Benutzer-Zugriffsstufe.", "section 8.9.5.15")
	add("Up.$BasicSecSettings", "OBJECT", "", "System variables for the security-settings dialog.", "Systemvariablen des Dialogs f\u00fcr Security-Einstellungen.", "section 8.9.5.16")
	add("Up.$BasicSecSettings.Activated", "BOOL", "", "Whether processing of the security-settings dialog is enabled.", "Gibt an, ob die Bearbeitung des Dialogs f\u00fcr Security-Einstellungen aktiviert ist.", "section 8.9.5.16")
	add("Up.$BasicSecSettings.ProcessMode", "ProcessMode", "", "Processing mode for the security-settings dialog.", "Bearbeitungsmodus des Dialogs f\u00fcr Security-Einstellungen.", "section 8.9.5.16")
	add("Up.$BasicSecSettings.OverrideSecArchPassword", "BOOL", "", "Whether the security-archive password is overridden.", "Gibt an, ob das Passwort des Security-Archivs \u00fcberschrieben wird.", "section 8.9.5.16")
	add("Up.$BasicSecSettings.SecArchPasswordFile", "STRING", "", "Path of the security-archive password file.", "Pfad der Passwortdatei f\u00fcr das Security-Archiv.", "section 8.9.5.16")

	addCurrentEnumerationDocumentation(add, readOnly)
	addCurrentStepAndEnvironmentDocumentation(add, readOnly)
	return result
}

func addBackupDialogDocumentation(add func(string, string, string, string, string, string), dialog, timingEnglish, timingGerman, section string) {
	prefix := "Up.$Dialog." + dialog
	manual := "section " + section
	add(prefix+".Backup", "BOOL", "", "Create a complete TGZ backup "+timingEnglish+".", "Erzeugt "+timingGerman+" eine vollst\u00e4ndige TGZ-Sicherung.", manual)
	add(prefix+".Archive", "BOOL", "", "Create a DSF archive "+timingEnglish+"; at least one archive area must be selected.", "Erzeugt "+timingGerman+" ein DSF-Archiv; mindestens ein Archivbereich muss ausgew\u00e4hlt sein.", manual)
	areas := []struct{ name, english, german string }{
		{name: "NC", english: "NC", german: "NC-Bereich"},
		{name: "PLC", english: "PLC", german: "PLC-Bereich"},
		{name: "DRV", english: "drive", german: "Antriebsdatenbereich"},
		{name: "HMI", english: "HMI", german: "HMI-Bereich"},
		{name: "SYS", english: "system-settings", german: "Systemeinstellungsbereich"},
	}
	for _, area := range areas {
		add(prefix+".Archive"+area.name, "BOOL", "", "Include the "+area.english+" area in the archive.", "Nimmt den "+area.german+" in das Archiv auf.", manual)
	}
	add(prefix+".ConfigDataItemsSelection", "ConfigDataItemsSelection", "", "Select which PLC configuration data are included in the backup.", "W\u00e4hlt aus, welche PLC-Konfigurationsdaten in die Sicherung aufgenommen werden.", manual)
	add(prefix+".ConfigDataItemsFilterFile", "STRING", "", "XML filter file that selects the PLC configuration data to back up.", "XML-Filterdatei zur Auswahl der zu sichernden PLC-Konfigurationsdaten.", manual)
}

func addCurrentEnumerationDocumentation(add func(string, string, string, string, string, string), readOnly string) {
	add("Up.$Dialog.ProcessModes", "ProcessMode", readOnly, "Dialog processing-mode values.", "Werte f\u00fcr den Bearbeitungsmodus von Dialogen.", "section 8.9.6.2")
	add("Up.$Dialog.ProcessModes.AUTOMATIC", "ProcessMode", readOnly, "Process the dialog automatically.", "Dialog automatisch bearbeiten.", "section 8.9.6.2")
	add("Up.$Dialog.ProcessModes.MANUAL", "ProcessMode", readOnly, "Process the dialog manually.", "Dialog manuell bearbeiten.", "section 8.9.6.2")
	add("Up.$Dialog.ProcessModes.PROGRESS", "ProcessMode", readOnly, "Use progress processing mode.", "Bearbeitungsmodus Fortschritt verwenden.", "section 8.9.6.2")
	add("Up.$Dialog.AccessData.Targets", "AccessData", readOnly, "Target-system selection values.", "Werte f\u00fcr die Auswahl des Zielsystems.", "section 8.9.6.3")
	add("Up.$Dialog.AccessData.Targets.NCU", "AccessData", readOnly, "Select the NCU target system.", "Zielsystem NCU ausw\u00e4hlen.", "section 8.9.6.3")
	add("Up.$Dialog.AccessData.Targets.IPC", "AccessData", readOnly, "Select the NCU and PCU target system.", "Zielsystem NCU und PCU ausw\u00e4hlen.", "section 8.9.6.3")
	add("Up.$Dialog.AccessData.Targets.VMC", "AccessData", readOnly, "Select the CMVM target system.", "Zielsystem CMVM ausw\u00e4hlen.", "section 8.9.6.3")
	add("Up.$Dialog.NcuSetup.Modes", "Mode", readOnly, "CNC software installation-mode values.", "Werte f\u00fcr den Installationsmodus der CNC-Software.", "section 8.9.6.4")
	add("Up.$Dialog.NcuSetup.Modes.NONE", "Mode", readOnly, "Do not install CNC or additional software.", "Keine CNC- oder Zusatzsoftware installieren.", "section 8.9.6.4")
	add("Up.$Dialog.NcuSetup.Modes.INSTALL", "Mode", readOnly, "Perform a new installation.", "Neuinstallation durchf\u00fchren.", "section 8.9.6.4")
	add("Up.$Dialog.NcuSetup.Modes.UPDATE", "Mode", readOnly, "Perform an upgrade.", "Hochr\u00fcstung durchf\u00fchren.", "section 8.9.6.4")
	add("Up.$Dialog.NcuSetup.Modes.SOFTWAREONLY", "Mode", readOnly, "Install additional software only.", "Nur Zusatzsoftware installieren.", "section 8.9.6.4")

	type enumDefinition struct {
		prefix, valueType, section string
		values                     []struct{ name, english, german string }
	}
	enums := []enumDefinition{
		{prefix: "Up.$Dialog.SystemConfig.NcSources", valueType: "NcSource", section: "8.9.6.5", values: []struct{ name, english, german string }{
			{name: "ORIGIN", english: "Use the current NC data at package execution.", german: "Aktuelle NC-Daten bei der Paketausf\u00fchrung verwenden."},
			{name: "FACTORY", english: "Use the NC default data after a general reset.", german: "NC-Standarddaten nach einem General Reset verwenden."},
			{name: "ARCHIVE", english: "Use NC data from an archive.", german: "NC-Daten aus einem Archiv verwenden."},
			{name: "UNUSED", english: "Do not edit or process NC data.", german: "NC-Daten nicht bearbeiten oder verarbeiten."},
		}},
		{prefix: "Up.$Dialog.PlcConfig.PlcSources", valueType: "PlcSource", section: "8.9.6.6", values: []struct{ name, english, german string }{
			{name: "ORIGIN", english: "Use the current PLC data at package execution.", german: "Aktuelle PLC-Daten bei der Paketausf\u00fchrung verwenden."},
			{name: "ARCHIVE", english: "Use PLC data from an archive.", german: "PLC-Daten aus einem Archiv verwenden."},
			{name: "UNUSED", english: "Do not process PLC data.", german: "PLC-Daten nicht verarbeiten."},
		}},
		{prefix: "Up.$Dialog.PlcConfig.ConfigDataItemsSources", valueType: "ConfigDataItemsSource", section: "8.9.6.7", values: []struct{ name, english, german string }{
			{name: "ORIGIN", english: "Use the current PLC configuration data from the control.", german: "Aktuelle PLC-Konfigurationsdaten aus der Steuerung verwenden."},
			{name: "UPDATE", english: "Update PLC configuration data from the selected file.", german: "PLC-Konfigurationsdaten aus der ausgew\u00e4hlten Datei aktualisieren."},
			{name: "UNUSED", english: "Do not edit PLC configuration data.", german: "PLC-Konfigurationsdaten nicht bearbeiten."},
		}},
		{prefix: "Up.$Dialog.SystemConfig.DrvSources", valueType: "DrvSource", section: "8.9.6.8", values: []struct{ name, english, german string }{
			{name: "ORIGIN", english: "Use the current drive data at package execution.", german: "Aktuelle Antriebsdaten bei der Paketausf\u00fchrung verwenden."},
			{name: "AUTOMATIC", english: "Use data after automatic device configuration.", german: "Daten nach einer automatischen Ger\u00e4tekonfiguration verwenden."},
			{name: "TARGET", english: "Use a user-specified topology.", german: "Benutzerdefinierte Topologie verwenden."},
			{name: "ARCHIVE", english: "Use drive data from an archive.", german: "Antriebsdaten aus einem Archiv verwenden."},
			{name: "UNUSED", english: "Do not edit or process drive data.", german: "Antriebsdaten nicht bearbeiten oder verarbeiten."},
		}},
		{prefix: "Up.$Dialog.SystemConfig.HmiSources", valueType: "HmiSource", section: "8.9.6.9", values: []struct{ name, english, german string }{
			{name: "ORIGIN", english: "Use the current HMI archive data at package execution.", german: "Aktuelle HMI-Archivdaten bei der Paketausf\u00fchrung verwenden."},
			{name: "ARCHIVE", english: "Use HMI data from an archive.", german: "HMI-Daten aus einem Archiv verwenden."},
			{name: "UNUSED", english: "Do not edit or process HMI data.", german: "HMI-Daten nicht bearbeiten oder verarbeiten."},
		}},
		{prefix: "Up.$Dialog.SystemConfig.SysSources", valueType: "SysSource", section: "8.9.6.10", values: []struct{ name, english, german string }{
			{name: "ORIGIN", english: "Use the current system settings at package execution.", german: "Aktuelle Systemeinstellungen bei der Paketausf\u00fchrung verwenden."},
			{name: "ARCHIVE", english: "Use system settings from an archive.", german: "Systemeinstellungen aus einem Archiv verwenden."},
			{name: "UNUSED", english: "Do not edit or process system settings.", german: "Systemeinstellungen nicht bearbeiten oder verarbeiten."},
		}},
	}
	for _, enum := range enums {
		manual := "section " + enum.section
		add(enum.prefix, enum.valueType, readOnly, enum.valueType+" enumeration values.", "Aufz\u00e4hlungswerte f\u00fcr "+enum.valueType+".", manual)
		for _, value := range enum.values {
			add(enum.prefix+"."+value.name, enum.valueType, readOnly, value.english, value.german, manual)
		}
	}
	for _, dialog := range []string{"NcuOrigin", "NcuResult"} {
		prefix := "Up.$Dialog." + dialog + ".ConfigDataItemsSelections"
		manual := "section 8.9.6.11"
		add(prefix, "ConfigDataItemsSelection", readOnly, "PLC configuration-data backup selection values.", "Auswahlwerte f\u00fcr die Sicherung der PLC-Konfigurationsdaten.", manual)
		add(prefix+".NONE", "ConfigDataItemsSelection", readOnly, "Do not back up PLC configuration data.", "PLC-Konfigurationsdaten nicht sichern.", manual)
		add(prefix+".ALL", "ConfigDataItemsSelection", readOnly, "Back up all PLC configuration data.", "Alle PLC-Konfigurationsdaten sichern.", manual)
		add(prefix+".FILTERED", "ConfigDataItemsSelection", readOnly, "Back up only PLC configuration data selected by the XML filter file.", "Nur die durch die XML-Filterdatei ausgew\u00e4hlten PLC-Konfigurationsdaten sichern.", manual)
	}
}

func addCurrentStepAndEnvironmentDocumentation(add func(string, string, string, string, string, string), readOnly string) {
	add("Up.$Step", "OBJECT", "", "Access step-tree items by ID.", "Greift \u00fcber die ID auf Eintr\u00e4ge des Schrittbaums zu.", "section 8.9.7")
	add("Up.$Step[?]", "OBJECT", readOnly, "Returns the step with the specified ID, or `null` when it does not exist.", "Gibt den Schritt mit der angegebenen ID oder `null` zur\u00fcck, wenn er nicht existiert.", "table 8-39")
	add("Up.$Step[?].Activated", "BOOL", "", "Activation state of the step; `true` means its activation checkbox is selected.", "Aktivierungszustand des Schritts; `true` bedeutet, dass sein Aktivierungs-Kontrollk\u00e4stchen ausgew\u00e4hlt ist.", "table 8-39")
	add("Up.$Step[?].Locked", "BOOL", "", "Whether changing activation of the step is locked.", "Gibt an, ob das \u00c4ndern der Schrittaktivierung gesperrt ist.", "table 8-39")
	add("Up.$Step[?].Collapsed", "BOOL", "", "Whether the step's substeps are collapsed and hidden from the operator.", "Gibt an, ob die Unterschritte eingeklappt und f\u00fcr den Bediener verborgen sind.", "table 8-39")
	add("Up.$Step[?].Processing", "BOOL", readOnly, "Runtime feedback indicating whether the step was executed.", "Laufzeitr\u00fcckmeldung, ob der Schritt ausgef\u00fchrt wurde.", "table 8-39")
	add("Up.$Dialog.?.Step[?]", "OBJECT", readOnly, "Returns the specified step in a dialog, or `null` when it does not exist.", "Gibt den angegebenen Schritt eines Dialogs oder `null` zur\u00fcck, wenn er nicht existiert.", "table 8-39")
	add("Up.$Dialog.?.Step[?].Activated", "BOOL", "", "Activation state of the dialog step.", "Aktivierungszustand des Dialogschritts.", "table 8-39")
	add("Up.$Dialog.?.Step[?].Locked", "BOOL", "", "Whether changing activation of the dialog step is locked.", "Gibt an, ob das \u00c4ndern der Aktivierung des Dialogschritts gesperrt ist.", "table 8-39")
	add("Up.$Dialog.?.Step[?].Collapsed", "BOOL", "", "Whether the dialog step's substeps are collapsed.", "Gibt an, ob die Unterschritte des Dialogschritts eingeklappt sind.", "table 8-39")
	add("Up.$Dialog.?.Step[?].Processing", "BOOL", readOnly, "Runtime feedback indicating whether the dialog step was executed.", "Laufzeitr\u00fcckmeldung, ob der Dialogschritt ausgef\u00fchrt wurde.", "table 8-39")

	add("Up.$Env", "OBJECT", readOnly, "Information from the environment of the running package.", "Informationen aus der Umgebung des laufenden Pakets.", "section 8.9.8")
	add("Up.$Env.RunTime", "RunTimes", readOnly, "Runtime environment in which the package is executing.", "Laufzeitumgebung, in der das Paket ausgef\u00fchrt wird.", "section 8.9.8.1")
	add("Up.$Env.NCU", "STRING", readOnly, "NCU/PPU hardware type from `hwversions.xml`.", "NCU/PPU-Hardwaretyp aus `hwversions.xml`.", "section 8.9.8.1")
	add("Up.$Env.PLC", "STRING", readOnly, "PLC hardware type from `hwversions.xml`.", "PLC-Hardwaretyp aus `hwversions.xml`.", "section 8.9.8.1")
	add("Up.$Env.SDID", "STRING", readOnly, "SD card identifier from `hwversions.xml`.", "SD-Kartenkennung aus `hwversions.xml`.", "section 8.9.8.1")
	add("Up.$Env.BatchMode", "BOOL", readOnly, "Whether package or script execution was started in command-line batch mode.", "Gibt an, ob die Paket- oder Skriptausf\u00fchrung im Kommandozeilen-Batchmodus gestartet wurde.", "section 8.9.8.1")
	add("Up.$Env.RunTimes", "RunTimes", readOnly, "Runtime-environment enumeration values.", "Aufz\u00e4hlungswerte f\u00fcr die Laufzeitumgebung.", "section 8.9.8.2")
	add("Up.$Env.RunTimes.LINUX", "RunTimes", readOnly, "Package execution under Linux.", "Paketausf\u00fchrung unter Linux.", "section 8.9.8.2")
	add("Up.$Env.RunTimes.WINDOWS", "RunTimes", readOnly, "Package execution under Microsoft Windows.", "Paketausf\u00fchrung unter Microsoft Windows.", "section 8.9.8.2")
}
