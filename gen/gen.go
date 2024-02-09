// This file is part of GoRE.
//
// Copyright (C) 2019-2024 GoRE Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// This program generates stdpkgs_gen.go, goversion_gen.go and moduledata_gen.go. It can be invoked by running
// go generate

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("go run ./gen [stdpkgs|goversion|moduledata]")
		return
	}

	switch os.Args[1] {
	case "stdpkgs":
		generateStdPkgs()
	case "goversion":
		generateGoVersions()
	case "moduledata":
		generateModuleData()
	default:
		fmt.Println("go run ./gen [stdpkgs|goversion|moduledata]")
	}
}
