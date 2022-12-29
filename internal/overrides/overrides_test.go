package overrides_test

import (
	"reflect"
	"testing"

	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/overrides"
)

var info = &distro.OSRelease{
	ID:   "centos",
	Like: []string{"rhel", "fedora"},
}

func TestResolve(t *testing.T) {
	names := overrides.Resolve(info, nil)

	expected := []string{
		"amd64_centos",
		"centos",
		"amd64_rhel",
		"rhel",
		"amd64_fedora",
		"fedora",
		"amd64",
		"",
	}

	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}

func TestResolveName(t *testing.T) {
	names := overrides.Resolve(info, &overrides.Opts{
		Name:        "deps",
		Overrides:   true,
		LikeDistros: true,
	})

	expected := []string{
		"deps_amd64_centos",
		"deps_centos",
		"deps_amd64_rhel",
		"deps_rhel",
		"deps_amd64_fedora",
		"deps_fedora",
		"deps_amd64",
		"deps",
	}

	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}

func TestResolveNoLikeDistros(t *testing.T) {
	names := overrides.Resolve(info, &overrides.Opts{
		Overrides:   true,
		LikeDistros: false,
	})

	expected := []string{
		"amd64_centos",
		"centos",
		"amd64",
		"",
	}

	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}

func TestResolveNoOverrides(t *testing.T) {
	names := overrides.Resolve(info, &overrides.Opts{
		Name:        "deps",
		Overrides:   false,
		LikeDistros: false,
	})

	expected := []string{"deps"}

	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}
