/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Arsen Musayelyan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package overrides_test

import (
	"reflect"
	"testing"

	"go.elara.ws/lure/distro"
	"go.elara.ws/lure/internal/overrides"
	"golang.org/x/text/language"
)

var info = &distro.OSRelease{
	ID:   "centos",
	Like: []string{"rhel", "fedora"},
}

func TestResolve(t *testing.T) {
	names, err := overrides.Resolve(info, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	expected := []string{
		"amd64_centos_en",
		"centos_en",
		"amd64_rhel_en",
		"rhel_en",
		"amd64_fedora_en",
		"fedora_en",
		"amd64_en",
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
	names, err := overrides.Resolve(info, &overrides.Opts{
		Name:        "deps",
		Overrides:   true,
		LikeDistros: true,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

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
	names, err := overrides.Resolve(info, &overrides.Opts{
		Overrides:   true,
		LikeDistros: false,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

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
	names, err := overrides.Resolve(info, &overrides.Opts{
		Name:        "deps",
		Overrides:   false,
		LikeDistros: false,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	expected := []string{"deps"}

	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}

func TestResolveLangs(t *testing.T) {
	names, err := overrides.Resolve(info, &overrides.Opts{
		Overrides:    true,
		Languages:    []string{"ru_RU", "en", "en_US"},
		LanguageTags: []language.Tag{language.BritishEnglish},
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	expected := []string{
		"amd64_centos_en",
		"centos_en",
		"amd64_en",
		"amd64_centos_ru",
		"centos_ru",
		"amd64_ru",
		"amd64_centos",
		"centos",
		"amd64",
		"",
	}

	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}
