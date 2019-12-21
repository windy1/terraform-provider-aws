package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func workspacesDirectoryRefreshStateFunc(conn *workspaces.WorkSpaces, directoryID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeWorkspaceDirectories(&workspaces.DescribeWorkspaceDirectoriesInput{
			DirectoryIds: []*string{aws.String(directoryID)},
		})
		if err != nil {
			return nil, workspaces.WorkspaceDirectoryStateError, err
		}
		if len(resp.Directories) == 0 {
			return resp, workspaces.WorkspaceDirectoryStateDeregistered, nil
		}
		return resp, *resp.Directories[0].State, nil
	}
}
