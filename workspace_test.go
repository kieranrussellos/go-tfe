package tfe

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaces(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	ws1, ws1Cleanup := createWorkspace(t, client, org)
	defer ws1Cleanup()
	ws2, ws2Cleanup := createWorkspace(t, client, org)
	defer ws2Cleanup()

	// List the workspaces within the organization.
	workspaces, err := client.Workspaces(*org.Name)
	require.Nil(t, err)

	expect := []*Workspace{ws1, ws2}

	// Sort to ensure we get a non-flaky comparison.
	sort.Stable(WorkspaceNameSort(expect))
	sort.Stable(WorkspaceNameSort(workspaces))

	assert.Equal(t, expect, workspaces)
}

func TestWorkspace(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	ws, wsCleanup := createWorkspace(t, client, org)
	defer wsCleanup()

	t.Run("when the workspace exists", func(t *testing.T) {
		result, err := client.Workspace(*org.Name, *ws.Name)
		require.Nil(t, err)
		assert.Equal(t, ws, result)
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		result, err := client.Workspace(*org.Name, "nope")
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("when the organization does not exist", func(t *testing.T) {
		result, err := client.Workspace("nope", "nope")
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("permissions are properly decoded", func(t *testing.T) {
		if !ws.Permissions.Can("destroy") {
			t.Fatal("should be able to destroy")
		}
	})
}

func TestCreateWorkspace(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	t.Run("with valid input", func(t *testing.T) {
		input := &CreateWorkspaceInput{
			Organization:     org.Name,
			Name:             String("foo"),
			AutoApply:        Bool(true),
			TerraformVersion: String("0.11.0"),
			WorkingDirectory: String("bar/"),
		}

		output, err := client.CreateWorkspace(input)
		require.Nil(t, err)

		// Get a refreshed view from the API.
		refreshedWorkspace, err := client.Workspace(*org.Name, *input.Name)
		require.Nil(t, err)

		for _, result := range []*Workspace{
			output.Workspace,
			refreshedWorkspace,
		} {
			assert.NotNil(t, result.ID)
			assert.Equal(t, input.Name, result.Name)
			assert.Equal(t, input.AutoApply, result.AutoApply)
			assert.Equal(t, input.WorkingDirectory, result.WorkingDirectory)
			assert.Equal(t, input.TerraformVersion, result.TerraformVersion)
		}
	})

	t.Run("when input is missing organization", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			Name: String("foo"),
		})
		assert.EqualError(t, err, "Organization is required")
		assert.Nil(t, result)
	})

	t.Run("when input is missing name", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			Organization: org.Name,
		})
		assert.EqualError(t, err, "Name is required")
		assert.Nil(t, result)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		result, err := client.CreateWorkspace(&CreateWorkspaceInput{
			Organization:     org.Name,
			Name:             String("bar"),
			TerraformVersion: String("nope"),
		})
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})
}

func TestModifyWorkspace(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	ws, _ := createWorkspace(t, client, org)

	t.Run("when updating a subset of values", func(t *testing.T) {
		before, err := client.Workspace(*org.Name, *ws.Name)
		require.Nil(t, err)

		input := &ModifyWorkspaceInput{
			Organization:     org.Name,
			Name:             ws.Name,
			TerraformVersion: String("0.10.0"),
		}

		output, err := client.ModifyWorkspace(input)
		require.Nil(t, err)

		after := output.Workspace
		assert.Equal(t, before.Name, after.Name)
		assert.Equal(t, before.AutoApply, after.AutoApply)
		assert.Equal(t, before.WorkingDirectory, after.WorkingDirectory)
		assert.NotEqual(t, before.TerraformVersion, after.TerraformVersion)
	})

	t.Run("with valid input", func(t *testing.T) {
		input := &ModifyWorkspaceInput{
			Organization:     org.Name,
			Name:             ws.Name,
			Rename:           String(randomString(t)),
			AutoApply:        Bool(false),
			TerraformVersion: String("0.11.1"),
			WorkingDirectory: String("baz/"),
		}

		output, err := client.ModifyWorkspace(input)
		require.Nil(t, err)

		// Get a refreshed view of the workspace from the API
		refreshed, err := client.Workspace(*org.Name, *input.Rename)
		require.Nil(t, err)

		for _, result := range []*Workspace{
			output.Workspace,
			refreshed,
		} {
			assert.Equal(t, result.Name, input.Rename)
			assert.Equal(t, result.AutoApply, input.AutoApply)
			assert.Equal(t, result.TerraformVersion, input.TerraformVersion)
			assert.Equal(t, result.WorkingDirectory, input.WorkingDirectory)
		}
	})

	t.Run("when input is missing organization", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			Name: String("foo"),
		})
		assert.EqualError(t, err, "Organization is required")
		assert.Nil(t, result)
	})

	t.Run("when input is missing name", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			Organization: org.Name,
		})
		assert.EqualError(t, err, "Name is required")
		assert.Nil(t, result)
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		result, err := client.ModifyWorkspace(&ModifyWorkspaceInput{
			Organization:     org.Name,
			Name:             ws.Name,
			TerraformVersion: String("nope"),
		})
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})
}

func TestDeleteWorkspace(t *testing.T) {
	client := testClient(t)

	org, cleanup := createOrganization(t, client)
	defer cleanup()

	ws, _ := createWorkspace(t, client, org)

	output, err := client.DeleteWorkspace(&DeleteWorkspaceInput{
		Organization: org.Name,
		Name:         ws.Name,
	})
	require.Nil(t, err)
	require.Equal(t, &DeleteWorkspaceOutput{}, output)

	// Try loading the workspace - it should fail.
	_, err = client.Workspace(*org.Name, *ws.Name)
	assert.EqualError(t, err, "Resource not found")
}