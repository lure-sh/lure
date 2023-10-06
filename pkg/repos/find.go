/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Elara Musayelyan
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

package repos

import (
	"context"

	"go.elara.ws/lure/internal/db"
)

// FindPkgs looks for packages matching the inputs inside the database.
// It returns a map that maps the package name input to any packages found for it.
// It also returns a slice that contains the names of all packages that were not found.
func FindPkgs(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error) {
	found := map[string][]db.Package{}
	notFound := []string(nil)

	for _, pkgName := range pkgs {
		if pkgName == "" {
			continue
		}

		result, err := db.GetPkgs(ctx, "name LIKE ?", pkgName)
		if err != nil {
			return nil, nil, err
		}

		added := 0
		for result.Next() {
			var pkg db.Package
			err = result.StructScan(&pkg)
			if err != nil {
				return nil, nil, err
			}

			added++
			found[pkgName] = append(found[pkgName], pkg)
		}
		result.Close()

		if added == 0 {
			result, err := db.GetPkgs(ctx, "json_array_contains(provides, ?)", pkgName)
			if err != nil {
				return nil, nil, err
			}

			for result.Next() {
				var pkg db.Package
				err = result.StructScan(&pkg)
				if err != nil {
					return nil, nil, err
				}

				added++
				found[pkgName] = append(found[pkgName], pkg)
			}

			result.Close()
		}

		if added == 0 {
			notFound = append(notFound, pkgName)
		}
	}

	return found, notFound, nil
}
