// Copyright 2022 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package solution

import (
	"fmt"
	"net/url"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

type StatusData struct {
	InstallTime       string `json:"installTime,omitempty"`
	InstallMessage    string `json:"installMessage,omitempty"`
	SuccessfulInstall bool   `json:"isSuccessful,omitempty"`
	SolutionName      string `json:"solutionName,omitempty"`
	SolutionVersion   string `json:"solutionVersion,omitempty"`
}

type StatusItem struct {
	StatusData StatusData `json:"data"`
	CreatedAt  string     `json:"createdAt"`
}

type ResponseBlob struct {
	Items []StatusItem `json:"items"`
}

var solutionStatusCmd = &cobra.Command{
	Use:   "status [flags]",
	Short: "Get the installation/upload status of a solution",
	Long: `This command provides the ability to see the current installation and upload status of a solution.
	
	Usage:
	fsoc solution status --name <solution-name> --solution-version <optional-solution-version> --status-type [upload | install | all]
	
	Flags/Options:
	--name - Flag to indicate the name of the solution for which you would like to fetch the upload/installation status
	--solution-version - OPTIONAL Flag to indicate the version of the solution for which you would like to fetch the upload/installation status
	--status-type - OPTIONAL Flag to specify the status that you would like to view.  If not specified, the output will contain both solution upload and solution installation status information
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return getSolutionStatus(cmd, args)
	},
	Args:             cobra.ExactArgs(0),
	TraverseChildren: true,
}

func getSolutionStatusCmd() *cobra.Command {
	solutionStatusCmd.Flags().
		String("name", "", "The name of the solution for which you would like to retrieve the upload status")
	_ = solutionStatusCmd.MarkFlagRequired("name")
	solutionStatusCmd.Flags().
		String("solution-version", "", "The version of the solution for which you would like to retrieve the upload status")
	solutionStatusCmd.Flags().
		String("status-type", "", "The status type that you want to see.  This can be one of [upload, install, all] and will default to all if not specified")

	return solutionStatusCmd
}

func getObject(url string, headers map[string]string) StatusItem {
	var res ResponseBlob
	var emptyData StatusItem

	err := api.HTTPGet(url, &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Issue fetching install/upload object: %v", err)
	}

	if len(res.Items) > 0 {
		return res.Items[0]
	} else {
		return emptyData
	}
}

func fetchValuesAndPrint(operation string, query string, requestHeaders map[string]string, cmd *cobra.Command) {
	uploadStatusItem := getObject(fmt.Sprintf(getSolutionReleaseUrl(), query), requestHeaders)
	installStatusItem := getObject(fmt.Sprintf(getSolutionInstallUrl(), query), requestHeaders)

	installStatusData := installStatusItem.StatusData
	uploadStatusData := uploadStatusItem.StatusData
	uploadStatusTimestamp := uploadStatusItem.CreatedAt

	headers := []string{"Solution Name"}
	values := []string{uploadStatusData.SolutionName}

	appendValue := func(header, value string) {
		headers = append(headers, header)
		values = append(values, value)
	}

	if operation == "upload" {
		appendValue("Solution Upload Version", uploadStatusData.SolutionVersion)
		appendValue("Upload Timestamp", uploadStatusTimestamp)
	} else if operation == "install" {
		appendValue("Solution Install Version", installStatusData.SolutionVersion)
		appendValue("Solution Install Successful?", fmt.Sprintf("%v", installStatusData.SuccessfulInstall))
		appendValue("Solution Install Time", installStatusData.InstallTime)
		appendValue("Solution Install Message", installStatusData.InstallMessage)
	} else {
		appendValue("Solution Upload Version", uploadStatusData.SolutionVersion)
		appendValue("Upload Timestamp", uploadStatusTimestamp)
		appendValue("Solution Install Version", installStatusData.SolutionVersion)
		appendValue("Solution Install Successful?", fmt.Sprintf("%v", installStatusData.SuccessfulInstall))
		appendValue("Solution Install Time", installStatusData.InstallTime)
		appendValue("Solution Install Message", installStatusData.InstallMessage)
	}

	output.PrintCmdOutputCustom(cmd, installStatusData, &output.Table{
		Headers: headers,
		Lines:   [][]string{values},
		Detail:  true,
	})
}

func getSolutionStatus(cmd *cobra.Command, args []string) error {
	var err error
	var filterQuery string
	cfg := config.GetCurrentContext()

	layerType := "TENANT"
	solutionName, err := cmd.Flags().GetString("name")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "name", err)
	}

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   cfg.Tenant,
	}
	solutionVersion, _ := cmd.Flags().GetString("solution-version")
	statusTypeToFetch, _ := cmd.Flags().GetString("status-type")

	if solutionVersion != "" {
		filterQuery = fmt.Sprintf(`data.solutionName eq "%s" and data.solutionVersion eq "%s"`, solutionName, solutionVersion)
	} else {
		filterQuery = fmt.Sprintf(`data.solutionName eq "%s"`, solutionName)
	}

	query := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(filterQuery))

	fetchValuesAndPrint(statusTypeToFetch, query, headers, cmd)

	return nil
}

func getSolutionReleaseUrl() string {
	return "objstore/v1beta/objects/extensibility:solutionRelease%s"
}

func getSolutionInstallUrl() string {
	return "objstore/v1beta/objects/extensibility:solutionInstall%s"
}
