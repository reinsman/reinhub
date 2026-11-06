package dbs

import (
	"testing"
)

type firstMigrationModel struct{}
type secondMigrationModel struct{}

func TestRegisterModelsAppendsModels(t *testing.T) {
	previous := autoMigrateModels
	autoMigrateModels = nil
	t.Cleanup(func() {
		autoMigrateModels = previous
	})

	first := &firstMigrationModel{}
	second := &secondMigrationModel{}
	RegisterModels(first)
	RegisterModels(second, first)

	if len(autoMigrateModels) != 3 {
		t.Fatalf("registered %d models, want 3", len(autoMigrateModels))
	}
	if autoMigrateModels[0] != first || autoMigrateModels[1] != second || autoMigrateModels[2] != first {
		t.Fatal("models were not appended in registration order")
	}
}
