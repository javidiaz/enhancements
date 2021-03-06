/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kepctl

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/enhancements/pkg/kepval/keps"
)

type QueryOpts struct {
	CommonArgs
	SIG        []string
	Status     []string
	Stage      []string
	IncludePRs bool
}

// Validate checks the args and cleans them up if needed
func (c *QueryOpts) Validate(args []string) error {
	if len(c.SIG) > 0 {
		var fixed []string
		for _, s := range c.SIG {
			if strings.HasPrefix(s, "sig-") {
				fixed = append(fixed, s)
			} else {
				fixed = append(fixed, "sig-"+s)
			}
		}
		c.SIG = fixed
	}
	//TODO: check the valid values of stage, status, etc.
	return nil
}

// Query searches the local repo and possibly GitHub for KEPs
// that match the search criteria.
func (c *Client) Query(opts QueryOpts) error {
	fmt.Fprintf(c.Out, "Searching for KEPs...\n")
	repoPath, err := c.findEnhancementsRepo(opts.CommonArgs)
	if err != nil {
		return errors.Wrap(err, "unable to search KEPs")
	}

	c.SetGitHubToken(opts.CommonArgs)

	var allKEPs []*keps.Proposal
	// load the KEPs for each listed SIG
	for _, sig := range opts.SIG {
		// KEPs in the local filesystem
		names, err := findLocalKEPs(repoPath, sig)
		if err != nil {
			return errors.Wrap(err, "unable to search for local KEPs")
		}

		for _, k := range names {
			kep, err := c.readKEP(repoPath, sig, k)
			if err != nil {
				fmt.Fprintf(c.Err, "ERROR READING KEP %s: %s\n", k, err)
			} else {
				allKEPs = append(allKEPs, kep)
			}
		}

		// Open PRs; existing KEPs with open PRs will be shown twice
		if opts.IncludePRs {
			prKeps, err := c.findKEPPullRequests(sig)
			if err != nil {
				return errors.Wrap(err, "unable to search for KEP PRs")
			}
			if prKeps != nil {
				allKEPs = append(allKEPs, prKeps...)
			}
		}
	}

	// filter the KEPs by criteria
	allowedStatus := sliceToMap(opts.Status)
	allowedStage := sliceToMap(opts.Stage)

	var keep []*keps.Proposal
	for _, k := range allKEPs {
		if len(opts.Status) > 0 && !allowedStatus[k.Status] {
			continue
		}
		if len(opts.Stage) > 0 && !allowedStage[k.Stage] {
			continue
		}
		keep = append(keep, k)
	}

	c.PrintTable(DefaultPrintConfigs("LastUpdated", "Stage", "Status", "SIG", "Authors", "Title", "Link"), keep)
	return nil
}

func sliceToMap(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}
