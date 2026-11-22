package db

import "testing"

func TestParseCreateTableAndDiff(t *testing.T) {
	ddl := `CREATE TABLE IF NOT EXISTS demo (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT ''
	);`
	cols, err := ParseCreateTable(ddl)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 3 {
		t.Fatalf("want 3 columns, got %d: %+v", len(cols), cols)
	}

	items := DiffTable(DiffInput{
		Table:       "demo",
		ExpectedDDL: ddl,
		ActualCols: []CatalogColumn{
			{Name: "id", TypeFull: "integer", Nullable: "NO", Extra: "auto_increment"},
			{Name: "name", TypeFull: "text", Nullable: "NO"},
		},
	})
	if len(items) == 0 {
		t.Fatal("expected missing column diff")
	}
}
