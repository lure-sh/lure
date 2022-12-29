package repos_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/repos"
	"go.arsenm.dev/lure/internal/types"
)

func TestFindPkgs(t *testing.T) {
	gdb, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	defer gdb.Close()

	err = db.Init(gdb)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	setCfgDirs(t)
	defer removeCacheDir(t)

	ctx := context.Background()

	err = repos.Pull(ctx, gdb, []types.Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	found, notFound, err := repos.FindPkgs(gdb, []string{"itd", "nonexistentpackage1", "nonexistentpackage2"})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if !reflect.DeepEqual(notFound, []string{"nonexistentpackage1", "nonexistentpackage2"}) {
		t.Errorf("Expected 'nonexistentpackage{1,2} not to be found")
	}

	if len(found) != 1 {
		t.Errorf("Expected 1 package found, got %d", len(found))
	}

	itdPkgs, ok := found["itd"]
	if !ok {
		t.Fatalf("Expected 'itd' packages to be found")
	}

	if len(itdPkgs) < 2 {
		t.Errorf("Expected two 'itd' packages to be found")
	}

	for i, pkg := range itdPkgs {
		if !strings.HasPrefix(pkg.Name, "itd") {
			t.Errorf("Expected package name of all found packages to start with 'itd', got %s on element %d", pkg.Name, i)
		}
	}
}

func TestFindPkgsEmpty(t *testing.T) {
	gdb, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	defer gdb.Close()

	err = db.Init(gdb)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	setCfgDirs(t)
	defer removeCacheDir(t)

	err = db.InsertPackage(gdb, db.Package{
		Name:        "test1",
		Repository:  "default",
		Version:     "0.0.1",
		Release:     1,
		Description: "Test package 1",
		Provides:    db.NewJSON([]string{""}),
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = db.InsertPackage(gdb, db.Package{
		Name:        "test2",
		Repository:  "default",
		Version:     "0.0.1",
		Release:     1,
		Description: "Test package 2",
		Provides:    db.NewJSON([]string{"test"}),
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	found, notFound, err := repos.FindPkgs(gdb, []string{"test", ""})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if len(notFound) != 0 {
		t.Errorf("Expected all packages to be found")
	}

	if len(found) != 1 {
		t.Errorf("Expected 1 package found, got %d", len(found))
	}

	testPkgs, ok := found["test"]
	if !ok {
		t.Fatalf("Expected 'test' packages to be found")
	}

	if len(testPkgs) != 1 {
		t.Errorf("Expected one 'test' package to be found, got %d", len(testPkgs))
	}

	if testPkgs[0].Name != "test2" {
		t.Errorf("Expected 'test2' package, got '%s'", testPkgs[0].Name)
	}
}
