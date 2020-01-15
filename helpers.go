// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"errors"
	"net/url"
)

func parse(url *url.URL) (repo, issue string, err error) {
	values, ok := url.Query()["repo"]
	if !ok || len(values[0]) < 1 {
		err = errors.New("No repo specified")
		return
	}
	repo = values[0]

	issue = "all"
	values, ok = url.Query()["issue"]
	if ok && len(values[0]) >= 1 {
		issue = values[0]
	}
	return
}
