package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func init() {
	resource.AddTestSweepers("aws_workspaces_directory", &resource.Sweeper{
		Name: "aws_workspaces_directory",
		F:    testSweepWorkspacesDirectories,
	})
}

func testSweepWorkspacesDirectories(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).workspacesconn

	input := &workspaces.DescribeWorkspaceDirectoriesInput{}
	for {
		resp, err := conn.DescribeWorkspaceDirectories(input)

		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping Workspace Directory sweep for %s: %s", region, err)
			return nil
		}

		if err != nil {
			return fmt.Errorf("error listing Workspace Directories: %s", err)
		}

		for _, directory := range resp.Directories {
			id := aws.StringValue(directory.DirectoryId)

			deregisterInput := workspaces.DeregisterWorkspaceDirectoryInput{
				DirectoryId: directory.DirectoryId,
			}

			log.Printf("[INFO] Deregistering Workspace Directory %q", deregisterInput)
			_, err := conn.DeregisterWorkspaceDirectory(&deregisterInput)
			if err != nil {
				return fmt.Errorf("error deregistering Workspace Directory %q: %s", id, err)
			}

			log.Printf("[INFO] Waiting for Workspace Directory %q to be deregistered", id)
			stateConf := &resource.StateChangeConf{
				Pending: []string{
					workspaces.WorkspaceDirectoryStateRegistering,
					workspaces.WorkspaceDirectoryStateRegistered,
					workspaces.WorkspaceDirectoryStateDeregistering,
				},
				Target: []string{
					workspaces.WorkspaceDirectoryStateDeregistered,
				},
				Refresh: workspacesDirectoryRefreshStateFunc(conn, id),
				Timeout: 10 * time.Minute,
			}

			_, err = stateConf.WaitForState()
			if err != nil {
				return fmt.Errorf("error waiting for Workspace Directory %q to be deregistered: %s", id, err)
			}
		}

		if resp.NextToken == nil {
			break
		}

		input.NextToken = resp.NextToken
	}

	return nil
}
