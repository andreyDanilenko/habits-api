package project

const (
	RouteList           = "/projects"
	RouteCreate         = "/projects"
	RouteGet            = "/projects/:projectId"
	RouteUpdate         = "/projects/:projectId"
	RouteDelete         = "/projects/:projectId"
	RouteEntities       = "/projects/:projectId/entities"
	RouteAttachEntity   = "/projects/:projectId/entities"
	RouteDetachEntity   = "/projects/:projectId/entities/:entityType/:entityId"
	RouteEntityProjects = "/entities/:entityType/:entityId/projects"
)
