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

package main

import (
	"github.com/jmoiron/sqlx"
	"go.elara.ws/logger/log"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
)

var gdb *sqlx.DB

func init() {
	var err error
	gdb, err = db.Open(config.DBPath)
	if err != nil {
		log.Fatal("Error opening database").Err(err).Send()
	}
}
