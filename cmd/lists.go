package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"

	"github.com/labstack/echo/v4"
)

type listsWrap struct {
	Results []models.List `json:"results"`

	Total   int `json:"total"`
	PerPage int `json:"per_page"`
	Page    int `json:"page"`
}

var (
	listQuerySortFields = []string{"name", "type", "subscriber_count", "created_at", "updated_at"}
)

// handleGetLists retrieves lists with additional metadata like subscriber counts. This may be slow.
func handleGetLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out listsWrap

		pg         = getPagination(c.QueryParams(), 20)
		query      = strings.TrimSpace(c.FormValue("query"))
		orderBy    = c.FormValue("order_by")
		order      = c.FormValue("order")
		minimal, _ = strconv.ParseBool(c.FormValue("minimal"))
		listID, _  = strconv.Atoi(c.Param("id"))
	)

	// Fetch one list.
	single := false
	if listID > 0 {
		single = true
	}

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	if !single && minimal {
		res, err := app.core.GetLists("")
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return c.JSON(http.StatusOK, okResp{[]struct{}{}})
		}

		// Meta.
		out.Results = res
		out.Total = len(res)
		out.Page = 1
		out.PerPage = out.Total

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Full list query.
	res, err := app.core.QueryLists(query, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	if single && len(res) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}
	if len(res) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	// Replace null tags.
	for i, v := range res {
		if v.Tags == nil {
			res[i].Tags = make([]string, 0)
		}

		// Total counts.
		for _, c := range v.SubscriberCounts {
			res[i].SubscriberCount += c
		}
	}

	if single {
		return c.JSON(http.StatusOK, okResp{res[0]})
	}

	// Meta.
	// TODO: add .query?
	out.Results = res
	out.Total = res[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage
	if out.PerPage == 0 {
		out.PerPage = out.Total
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateList handles list creation.
func handleCreateList(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		l   = models.List{}
	)

	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("lists.invalidName"))
	}

	out, err := app.core.CreateList(l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateList handles list modification.
func handleUpdateList(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var l models.List
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("lists.invalidName"))
	}

	out, err := app.core.UpdateList(id, l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteLists handles list deletion, either a single one (ID in the URI), or a list.
func handleDeleteLists(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	if err := app.core.DeleteLists(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
