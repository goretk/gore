// This file is part of GoRE.
//
// Copyright (C) 2019-2021 GoRE Authors
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

package gore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringer(t *testing.T) {
	assert := assert.New(t)

	// Setup
	tests := []struct {
		name     string
		obj      fmt.Stringer
		expected string
	}{
		{"main", &Function{Name: "main", PackageName: "main"}, "main"},
		{"setup", &Function{Name: "setup", PackageName: "main"}, "setup"},
		{"method", &Method{Receiver: "(T)", Function: &Function{Name: "MyMethod"}}, "(T)MyMethod"},
	}

	for _, test := range tests {
		t.Run("stringer_"+test.name, func(t *testing.T) {
			assert.Equal(test.expected, test.obj.String())
		})
	}

}
