package storage

import "fmt"

// ProjectPath returns the S3 key for project.json.
func ProjectPath(project string) string {
	return project + "/project.json"
}

// MembersPath returns the S3 key for members.json.
func MembersPath(project string) string {
	return project + "/members.json"
}

// KeyringPath returns the S3 key for an environment's keyring.json.
func KeyringPath(project, env string) string {
	return fmt.Sprintf("%s/environments/%s/keyring.json", project, env)
}

// CurrentPath returns the S3 key for an environment's current.json.
func CurrentPath(project, env string) string {
	return fmt.Sprintf("%s/environments/%s/current.json", project, env)
}

// HistoryPath returns the S3 key for a specific history version.
func HistoryPath(project, env string, version int) string {
	return fmt.Sprintf("%s/environments/%s/history/%06d.json", project, env, version)
}

// HistoryPrefix returns the S3 prefix for listing history entries.
func HistoryPrefix(project, env string) string {
	return fmt.Sprintf("%s/environments/%s/history/", project, env)
}

// EnvironmentsPrefix returns the S3 prefix for listing environments.
func EnvironmentsPrefix(project string) string {
	return project + "/environments/"
}
