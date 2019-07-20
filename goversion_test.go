// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvingVersionFromTag(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		tag          string
		expectingNil bool
	}{
		{"go1", false},
		{"go1.0.1", false},
		{"go1.10.5", false},
		{"go1.10beta2", false},
		{"go1.4", false},
		{"go1234", true},
		{"go1.", true},
	}

	for _, test := range tests {
		t.Run("resolve_tag_"+test.tag, func(t *testing.T) {
			t.Parallel()
			v := ResolveGoVersion(test.tag)
			if test.expectingNil {
				assert.Nil(v)
			} else {
				assert.Equal(test.tag, v.Name, "Wrong version returned")
			}
		})
	}
}

func TestMatchGoVersion(t *testing.T) {
	assert := assert.New(t)
	padding := "teststringPadding"
	for _, goversion := range goversions {
		t.Run("match_"+goversion.Name, func(t *testing.T) {
			actual := matchGoVersionString([]byte(goversion.Name + padding))
			assert.Equal(goversion.Name, actual, "Wrong version matched")
		})
	}
}

func TestVersionComparer(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		a   string
		b   string
		val int
	}{
		{"go2.0.0", "go1.0.0", 1},
		{"go1.0.0", "go2.0.0", -1},
		{"go1.7.1", "go1.7.1", 0},
		{"go1.7.1", "go1.7.2", -1},
		{"go1.7.2", "go1.7.1", 1},
		{"go1.8.1", "go1.7.2", 1},
		{"go1.7.1", "go1.8.2", -1},
		{"go1.7.1", "go1.7", 1},
		{"go1.7", "go1.7.2", -1},
		{"go1.7beta1", "go1.7beta2", -1},
		{"go1.7beta2", "go1.7beta1", 1},
		{"go1.7", "go1.7beta1", 1},
		{"go1.7rc1", "go1.7beta1", 1},
		{"go1.7beta2", "go1.7rc1", -1},
		{"go1.7rc2", "go1.7rc1", 1},
		{"go1.7rc1", "go1.7rc2", -1},
		{"go1.7", "go1.7rc2", 1},
		{"go1.7rc1", "go1.7", -1},
		{"go1", "go1.4beta1", -1},
		{"go1.4beta1", "go1", 1},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Testing case %d", i+1), func(t *testing.T) {
			t.Parallel()
			assert.Equal(test.val, GoVersionCompare(test.a, test.b), fmt.Sprintf("Case %d failed", i+1))
		})
	}
}
