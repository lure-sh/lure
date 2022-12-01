package repos

import (
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/types"
	"go.arsenm.dev/lure/internal/db"
)

func FindPkgs(gdb *genji.DB, pkgs []string) (map[string][]db.Package, error) {
	found := map[string][]db.Package{}

	for _, pkgName := range pkgs {
		result, err := db.GetPkgs(gdb, "name LIKE ?", pkgName)
		if err != nil {
			return nil, err
		}

		added := 0
		err = result.Iterate(func(d types.Document) error {
			var pkg db.Package
			err = document.StructScan(d, &pkg)
			if err != nil {
				return err
			}

			added++
			found[pkgName] = append(found[pkgName], pkg)
			return nil
		})
		result.Close()
		if err != nil {
			return nil, err
		}

		if added == 0 {
			result, err := db.GetPkgs(gdb, "? IN provides", pkgName)
			if err != nil {
				return nil, err
			}

			err = result.Iterate(func(d types.Document) error {
				var pkg db.Package
				err = document.StructScan(d, &pkg)
				if err != nil {
					return err
				}

				added++
				found[pkgName] = append(found[pkgName], pkg)
				return nil
			})
			result.Close()
			if err != nil {
				return nil, err
			}
		}
	}

	return found, nil
}
