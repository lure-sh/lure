package repos

import (
	"github.com/jmoiron/sqlx"
	"go.arsenm.dev/lure/internal/db"
)

// FindPkgs looks for packages matching the inputs inside the database.
// It returns a map that maps the package name input to the packages found for it.
// It also returns a slice that contains the names of all packages that were not found.
func FindPkgs(gdb *sqlx.DB, pkgs []string) (map[string][]db.Package, []string, error) {
	found := map[string][]db.Package{}
	notFound := []string(nil)

	for _, pkgName := range pkgs {
		result, err := db.GetPkgs(gdb, "name LIKE ?", pkgName)
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
			result, err := db.GetPkgs(gdb, "provides.value = ?", pkgName)
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
