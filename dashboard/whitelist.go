// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

//
// The dashboard is working on a whitelist-based approach.
//
// There's no restriction on being in whitelist for users or
// organizations, so all pull requests with usernames will be
// accepted without questions. The only reason why it exists is to
// suppress abuse of the service (like faking data) and my
// unwillingness to implement complicated authentication systems
// which is for sure will be less secure than this.
//
// `Key` is the username on GitHub, `value` is sha256(DASHBOARD_ACCESS_TOKEN).
// Keep DASHBOARD_ACCESS_TOKEN in secret and pass in the environment variables
// (https://git.io/JvkjH) to donate-ci (see .github/workflows/donate.yml).
//
// Example:
//   $ DASHBOARD_ACCESS_TOKEN=$(head /dev/random| sha256sum | awk '{print $1}')
//   $ echo -n $DASHBOARD_ACCESS_TOKEN | sha256sum
//   75b78c574cdba3f4a558e3329e7369d24cd40b5357d800e3b006d69e92407e7c
//

var whitelist = map[string]string{
	"jollheef": "75b78c574cdba3f4a558e3329e7369d24cd40b5357d800e3b006d69e92407e7c",
	// add yourself here
}
