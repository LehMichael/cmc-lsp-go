package parser

import (
	"fmt"
	"strings"
)

func IdentifierString(identifier IdentifierExpression) string {
	var segments []string
	for _, segment := range identifier.Segments {
		var value strings.Builder
		for _, part := range segment.Parts {
			switch part := part.(type) {
			case LiteralIdentifier:
				value.WriteString(string(part))
			case ReplacementIdentifier:
				value.WriteString("$(")
				value.WriteString(IdentifierString(IdentifierExpression(part)))
				value.WriteByte(')')
			case IndexIdentifier:
				value.WriteString(string(part))
			}
		}
		segments = append(segments, value.String())
	}
	name := strings.Join(segments, ".")
	if identifier.Section != nil {
		name = SectionString(*identifier.Section) + "." + name
	}
	return name
}

func SectionString(section SectionSwitchKind) string {
	switch section := section.(type) {
	case *ChannelSection:
		if section.Namespace == Chandata {
			return fmt.Sprintf("CHANDATA(%d)", section.Channo)
		}
		prefix := ""
		if section.Namespace == Nc {
			prefix = "NC"
		}
		return fmt.Sprintf("%s[C%d]", prefix, section.Channo)
	case *DriveSection:
		prefix := ""
		if section.Namespace == Ps {
			prefix = "PS"
		}
		return fmt.Sprintf("%s[B%d_S%d_PS%d]", prefix, section.Bus, section.Slave, section.Do)
	case DisplaySection:
		prefix := ""
		if section.Namespace == Bd {
			prefix = "BD"
		}
		return fmt.Sprintf("%s[%s]", prefix, section.Name)
	case DynamicSection:
		return string(section)
	case InvalidSection:
		return "<invalid section>"
	default:
		return "<section>"
	}
}
