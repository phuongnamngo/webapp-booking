package testutil

import "testing"

func TestResolveTestPostgresURLUsesExplicitOverride(t *testing.T) {
	resolved, err := resolveTestPostgresURL(
		"postgres://seatsurfing:secret@127.0.0.1:5432/seatsurfing?sslmode=disable",
		"postgres://seatsurfing:secret@127.0.0.1:5432/custom_test?sslmode=disable",
	)

	CheckTestIsNil(t, err)
	CheckTestString(t, "postgres://seatsurfing:secret@127.0.0.1:5432/custom_test?sslmode=disable", resolved)
}

func TestResolveTestPostgresURLDerivesTestDatabaseName(t *testing.T) {
	resolved, err := resolveTestPostgresURL(
		"postgres://seatsurfing:secret@127.0.0.1:5432/seatsurfing?sslmode=disable",
		"",
	)

	CheckTestIsNil(t, err)
	CheckTestString(t, "postgres://seatsurfing:secret@127.0.0.1:5432/seatsurfing_test?sslmode=disable", resolved)
}

func TestResolveTestPostgresURLKeepsExistingTestDatabaseName(t *testing.T) {
	resolved, err := resolveTestPostgresURL(
		"postgres://seatsurfing:secret@127.0.0.1:5432/seatsurfing_test?sslmode=disable",
		"",
	)

	CheckTestIsNil(t, err)
	CheckTestString(t, "postgres://seatsurfing:secret@127.0.0.1:5432/seatsurfing_test?sslmode=disable", resolved)
}

func TestRequireTestDatabaseURLRejectsNonTestDatabase(t *testing.T) {
	defer func() {
		recovered := recover()
		CheckTestBool(t, true, recovered != nil)
	}()

	requireTestDatabaseURL("postgres://seatsurfing:secret@127.0.0.1:5432/seatsurfing?sslmode=disable")
}

func TestRequireTestDatabaseURLAllowsTestDatabase(t *testing.T) {
	requireTestDatabaseURL("postgres://seatsurfing:secret@127.0.0.1:5432/seatsurfing_test?sslmode=disable")
}
