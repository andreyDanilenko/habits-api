package workspace

const (
	RouteList       = ""
	RouteCreate     = ""
	RouteCurrent    = "/current"
	RouteMyLicenses = "/me/module-licenses"
	RouteGet        = "/:workspaceId"
	RouteUpdate     = "/:workspaceId"
	RouteDelete     = "/:workspaceId"
	RouteLogo       = "/:workspaceId/logo"
	RouteMembers    = "/:workspaceId/members"
	RouteMemberOne  = "/:workspaceId/members/:userId"
	RouteSwitch     = "/:workspaceId/switch"
	RouteModules    = "/:workspaceId/modules"
	RouteModuleOne  = "/:workspaceId/modules/:moduleCode"
)
