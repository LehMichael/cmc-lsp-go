//go:build !windows

package database

func installedDatabaseCandidates() []string {
	return nil
}
