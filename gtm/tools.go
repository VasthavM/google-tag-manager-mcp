package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// sharedClient is the single GTM client used by all tool handlers.
var sharedClient *Client

// SetSharedClient sets the GTM client used by all handlers.
// Must be called before serving any MCP requests.
func SetSharedClient(c *Client) { sharedClient = c }

// RegisterTools adds all GTM tools to the MCP server.
func RegisterTools(server *mcp.Server) {
	// Read operations
	registerListAccounts(server)
	registerListContainers(server)
	registerListWorkspaces(server)
	registerListTags(server)
	registerGetTag(server)
	registerListTriggers(server)
	registerGetTrigger(server)
	registerListVariables(server)
	registerGetVariable(server)
	registerListFolders(server)
	registerGetFolderEntities(server)
	registerListTemplates(server)
	registerGetTemplate(server)
	registerListVersions(server)

	// Write operations
	registerCreateTag(server)
	registerUpdateTag(server)
	registerDeleteTag(server)
	registerCreateTrigger(server)
	registerUpdateTrigger(server)
	registerDeleteTrigger(server)
	registerCreateVariable(server)
	registerUpdateVariable(server)
	registerDeleteVariable(server)
	registerCreateContainer(server)
	registerDeleteContainer(server)
	registerCreateWorkspace(server)

	// Workspace status
	registerGetWorkspaceStatus(server)

	// Version operations
	registerCreateVersion(server)
	registerPublishVersion(server)

	// Template operations
	registerImportGalleryTemplate(server)
	registerCreateTemplate(server)
	registerUpdateTemplate(server)
	registerDeleteTemplate(server)

	// Built-in variables
	registerListBuiltInVariables(server)
	registerEnableBuiltInVariables(server)
	registerDisableBuiltInVariables(server)

	// Clients (server-side containers)
	registerListClients(server)
	registerGetClient(server)
	registerCreateClient(server)
	registerUpdateClient(server)
	registerDeleteClient(server)

	// Transformations (server-side containers)
	registerListTransformations(server)
	registerGetTransformation(server)
	registerCreateTransformation(server)
	registerUpdateTransformation(server)
	registerDeleteTransformation(server)

	// Templates (help LLMs with correct parameter formats)
	registerGetTagTemplates(server)
	registerGetTriggerTemplates(server)

	// Resources (URI-based read access)
	RegisterResources(server)

	// Prompts (template workflows)
	RegisterPrompts(server)
}

// getClient returns the shared GTM client.
func getClient(ctx context.Context) (*Client, error) {
	if sharedClient == nil {
		return nil, fmt.Errorf("GTM client not initialized")
	}
	return sharedClient, nil
}
