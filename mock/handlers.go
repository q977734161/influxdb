package mock

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/influxdata/mrfusion"
	"github.com/influxdata/mrfusion/models"
	op "github.com/influxdata/mrfusion/restapi/operations"
	"golang.org/x/net/context"
)

type Handler struct {
	Store      mrfusion.ExplorationStore
	TimeSeries mrfusion.TimeSeries
}

func NewHandler() Handler {
	return Handler{
		DefaultExplorationStore,
		DefaultTimeSeries,
	}
}

func sampleSource() *models.Source {
	name := "muh name"
	url := "http://localhost:8086"

	return &models.Source{
		ID: "1",
		Links: &models.SourceLinks{
			Self:  "/chronograf/v1/sources/1",
			Proxy: "/chronograf/v1/sources/1/proxy",
		},
		Name:     &name,
		Type:     "influx-enterprise",
		Username: "HOWDY!",
		Password: "changeme",
		URL:      &url,
	}
}

func (m *Handler) NewSource(ctx context.Context, params op.PostSourcesParams) middleware.Responder {
	return op.NewPostSourcesCreated()
}

func (m *Handler) Sources(ctx context.Context, params op.GetSourcesParams) middleware.Responder {
	res := &models.Sources{
		Sources: []*models.Source{
			sampleSource(),
		},
	}

	return op.NewGetSourcesOK().WithPayload(res)
}

func (m *Handler) SourcesID(ctx context.Context, params op.GetSourcesIDParams) middleware.Responder {
	if params.ID != "1" {
		return op.NewGetSourcesIDNotFound()
	}
	return op.NewGetSourcesIDOK().WithPayload(sampleSource())
}

func (m *Handler) Proxy(ctx context.Context, params op.PostSourcesIDProxyParams) middleware.Responder {
	query := mrfusion.Query{
		Command: *params.Query.Query,
		DB:      params.Query.Db,
		RP:      params.Query.Rp,
	}
	response, err := m.TimeSeries.Query(ctx, mrfusion.Query(query))
	if err != nil {
		return op.NewPostSourcesIDProxyDefault(500)
	}

	res := &models.ProxyResponse{
		Results: response,
	}
	return op.NewPostSourcesIDProxyOK().WithPayload(res)
}

func (m *Handler) MonitoredServices(ctx context.Context, params op.GetSourcesIDMonitoredParams) middleware.Responder {
	srvs, err := m.TimeSeries.MonitoredServices(ctx)
	if err != nil {
		return op.NewGetSourcesIDMonitoredDefault(500)
	}
	res := &models.Services{}
	for _, s := range srvs {
		res.Services = append(res.Services, &models.Service{
			TagKey:   s.TagKey,
			TagValue: s.TagValue,
			Type:     s.Type,
		})
	}
	return op.NewGetSourcesIDMonitoredOK().WithPayload(res)
}

func (m *Handler) Explorations(ctx context.Context, params op.GetSourcesIDUsersUserIDExplorationsParams) middleware.Responder {
	id, err := strconv.Atoi(params.UserID)
	if err != nil {
		return op.NewGetSourcesIDUsersUserIDExplorationsDefault(500)
	}
	exs, err := m.Store.Query(ctx, mrfusion.UserID(id))
	if err != nil {
		return op.NewGetSourcesIDUsersUserIDExplorationsNotFound()
	}
	res := &models.Explorations{}
	for i, e := range exs {
		rel := "self"
		href := fmt.Sprintf("/chronograf/v1/sources/1/users/%d/explorations/%d", id, i)
		res.Explorations = append(res.Explorations, &models.Exploration{
			Data:      e.Data,
			Name:      e.Name,
			UpdatedAt: strfmt.DateTime(e.UpdatedAt),
			CreatedAt: strfmt.DateTime(e.CreatedAt),
			Link: &models.Link{
				Rel:  &rel,
				Href: &href,
			},
		},
		)
	}
	return op.NewGetSourcesIDUsersUserIDExplorationsOK().WithPayload(res)
}

func (m *Handler) Exploration(ctx context.Context, params op.GetSourcesIDUsersUserIDExplorationsExplorationIDParams) middleware.Responder {
	id, err := strconv.Atoi(params.UserID)
	if err != nil {
		errMsg := &models.Error{Code: 500, Message: "Error converting user id"}
		return op.NewGetSourcesIDUsersUserIDExplorationsDefault(500).WithPayload(errMsg)
	}

	eID, err := strconv.Atoi(params.ExplorationID)
	if err != nil {
		errMsg := &models.Error{Code: 500, Message: "Error converting exploration id"}
		return op.NewGetSourcesIDUsersUserIDExplorationsExplorationIDDefault(500).WithPayload(errMsg)
	}

	e, err := m.Store.Get(ctx, mrfusion.ExplorationID(eID))
	if err != nil {
		log.Printf("Error unknown exploration id: %d: %v", eID, err)
		errMsg := &models.Error{Code: 404, Message: "Error unknown exploration id"}
		return op.NewGetSourcesIDUsersUserIDExplorationsExplorationIDNotFound().WithPayload(errMsg)
	}

	rel := "self"
	href := fmt.Sprintf("/chronograf/v1/sources/1/users/%d/explorations/%d", id, eID)
	res := &models.Exploration{
		Data:      e.Data,
		Name:      e.Name,
		UpdatedAt: strfmt.DateTime(e.UpdatedAt),
		CreatedAt: strfmt.DateTime(e.CreatedAt),
		Link: &models.Link{
			Rel:  &rel,
			Href: &href,
		},
	}
	return op.NewGetSourcesIDUsersUserIDExplorationsExplorationIDOK().WithPayload(res)
}

func (m *Handler) UpdateExploration(ctx context.Context, params op.PatchSourcesIDUsersUserIDExplorationsExplorationIDParams) middleware.Responder {
	eID, err := strconv.Atoi(params.ExplorationID)
	if err != nil {
		return op.NewPatchSourcesIDUsersUserIDExplorationsExplorationIDDefault(500)
	}

	e, err := m.Store.Get(ctx, mrfusion.ExplorationID(eID))
	if err != nil {
		log.Printf("Error unknown exploration id: %d: %v", eID, err)
		errMsg := &models.Error{Code: 404, Message: "Error unknown exploration id"}
		return op.NewPatchSourcesIDUsersUserIDExplorationsExplorationIDNotFound().WithPayload(errMsg)
	}
	if params.Exploration != nil {
		e.ID = mrfusion.ExplorationID(eID)
		e.Data = params.Exploration.Data.(string)
		e.Name = params.Exploration.Name
		m.Store.Update(ctx, e)
	}
	return op.NewPatchSourcesIDUsersUserIDExplorationsExplorationIDNoContent()
}

func (m *Handler) NewExploration(ctx context.Context, params op.PostSourcesIDUsersUserIDExplorationsParams) middleware.Responder {
	id, err := strconv.Atoi(params.UserID)
	if err != nil {
		return op.NewPostSourcesIDUsersUserIDExplorationsDefault(500)
	}

	exs, err := m.Store.Query(ctx, mrfusion.UserID(id))
	if err != nil {
		log.Printf("Error unknown user id: %d: %v", id, err)
		errMsg := &models.Error{Code: 404, Message: "Error unknown user id"}
		return op.NewPostSourcesIDUsersUserIDExplorationsNotFound().WithPayload(errMsg)
	}
	eID := len(exs)

	if params.Exploration != nil {
		e := mrfusion.Exploration{
			Data: params.Exploration.Data.(string),
			Name: params.Exploration.Name,
			ID:   mrfusion.ExplorationID(eID),
		}
		m.Store.Add(ctx, e)
	}
	params.Exploration.UpdatedAt = strfmt.DateTime(time.Now())
	params.Exploration.CreatedAt = strfmt.DateTime(time.Now())

	loc := fmt.Sprintf("/chronograf/v1/sources/1/users/%d/explorations/%d", id, eID)
	rel := "self"

	link := &models.Link{
		Href: &loc,
		Rel:  &rel,
	}
	params.Exploration.Link = link
	return op.NewPostSourcesIDUsersUserIDExplorationsCreated().WithPayload(params.Exploration).WithLocation(loc)

}

func (m *Handler) DeleteExploration(ctx context.Context, params op.DeleteSourcesIDUsersUserIDExplorationsExplorationIDParams) middleware.Responder {
	ID, err := strconv.Atoi(params.ExplorationID)
	if err != nil {
		return op.NewDeleteSourcesIDUsersUserIDExplorationsExplorationIDDefault(500)
	}

	if err := m.Store.Delete(ctx, mrfusion.Exploration{ID: mrfusion.ExplorationID(ID)}); err != nil {
		log.Printf("Error unknown explorations id: %d: %v", ID, err)
		errMsg := &models.Error{Code: 404, Message: "Error unknown user id"}
		return op.NewDeleteSourcesIDUsersUserIDExplorationsExplorationIDNotFound().WithPayload(errMsg)
	}
	return op.NewDeleteSourcesIDUsersUserIDExplorationsExplorationIDNoContent()
}
