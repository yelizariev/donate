// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"code.dumpstack.io/tools/donate/database"
)

var dashboardAccessToken = os.Getenv("DASHBOARD_ACCESS_TOKEN")

func dashboardPing(totalUSD float64, issue database.Issue) {
	if dashboardAccessToken == "" {
		return
	}

	c := strings.Replace(fmt.Sprintf("%.02f", totalUSD), ".", "", -1)
	cents, _ := strconv.ParseInt(c, 10, 64)

	dashboardURL := "https://donate.dumpstack.io"

	fullIssueURL := fmt.Sprintf("%s/issues/%d", issue.Repo, issue.ID)

	url := fmt.Sprintf("%s/put?url=%s&sum=%d&key=%s",
		dashboardURL, fullIssueURL, cents, dashboardAccessToken)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
