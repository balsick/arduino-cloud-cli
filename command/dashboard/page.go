// This file is part of arduino-cloud-cli.
//
// Copyright (C) ARDUINO SRL (http://www.arduino.cc)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package dashboard

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/arduino/arduino-cloud-cli/config"
	"github.com/arduino/arduino-cloud-cli/internal/iot"
	iotclient "github.com/arduino/iot-client-go/v3"
	"github.com/gofrs/uuid"
)

// defaultPageIcon is the icon the Arduino Cloud UI gives to new pages.
const defaultPageIcon = "dashboard_filled"

// PageInfo contains the main parameters of a dashboard page.
type PageInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int64  `json:"position"`
	Icon     string `json:"icon,omitempty"`
	Widgets  int    `json:"widgets"`
}

// CreatePageParams contains the parameters needed to add a page to a dashboard.
type CreatePageParams struct {
	DashboardID string
	Name        string
	Icon        *string // Defaults to the icon used by the Arduino Cloud UI
	Position    *int64  // 0-based position among the pages; defaults to the last one
}

// UpdatePageParams contains the parameters needed to change a page.
// Nil fields are left unchanged.
type UpdatePageParams struct {
	DashboardID string
	PageID      string
	Name        *string
	Icon        *string
	Position    *int64 // 0-based position among the pages
}

// DeletePageParams contains the parameters needed to delete a page.
// If the page contains widgets, either MoveWidgetsTo or DeleteWidgets is required.
type DeletePageParams struct {
	DashboardID   string
	PageID        string
	MoveWidgetsTo *string // Page that receives the widgets of the deleted page
	DeleteWidgets bool    // Delete the widgets of the deleted page
}

// ListPages returns the pages of a dashboard, sorted by position.
func ListPages(ctx context.Context, cred *config.Credentials, dashboardID string) ([]PageInfo, error) {
	iotClient, err := iot.NewClient(cred)
	if err != nil {
		return nil, err
	}
	dashboard, err := iotClient.DashboardShow(ctx, dashboardID)
	if err != nil {
		return nil, err
	}
	return getPagesInfo(toDashboardPayload(dashboard)), nil
}

// CreatePage adds a page to a dashboard.
func CreatePage(ctx context.Context, cred *config.Credentials, params *CreatePageParams) (*PageInfo, error) {
	var pageID string
	updated, err := editDashboard(ctx, cred, params.DashboardID, func(_ context.Context, _ *iot.Client, d *iotclient.Dashboardv3) error {
		var err error
		pageID, err = createPage(d, params)
		return err
	})
	if err != nil {
		return nil, err
	}
	return findPageInfo(updated, pageID)
}

// UpdatePage renames, moves or changes the icon of a page.
func UpdatePage(ctx context.Context, cred *config.Credentials, params *UpdatePageParams) (*PageInfo, error) {
	updated, err := editDashboard(ctx, cred, params.DashboardID, func(_ context.Context, _ *iot.Client, d *iotclient.Dashboardv3) error {
		return updatePage(d, params)
	})
	if err != nil {
		return nil, err
	}
	return findPageInfo(updated, params.PageID)
}

// DeletePage deletes a page, moving or deleting its widgets.
func DeletePage(ctx context.Context, cred *config.Credentials, params *DeletePageParams) error {
	_, err := editDashboard(ctx, cred, params.DashboardID, func(_ context.Context, _ *iot.Client, d *iotclient.Dashboardv3) error {
		return deletePage(d, params)
	})
	return err
}

func createPage(d *iotclient.Dashboardv3, params *CreatePageParams) (string, error) {
	if params.Name == "" {
		return "", errors.New("page name not specified")
	}

	id, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("cannot create a uuid for new page: %w", err)
	}
	icon := defaultPageIcon
	if params.Icon != nil {
		icon = *params.Icon
	}
	page := iotclient.Pagepayload{
		Id:                   id.String(),
		Name:                 params.Name,
		Icon:                 &icon,
		AdditionalProperties: map[string]interface{}{},
	}

	pages := sortedPages(d)
	if len(pages) == 0 {
		// Widgets of a dashboard without pages belong to the default page:
		// move them to the new page so they don't get lost.
		for i := range d.Widgets {
			if widgetPageID(&d.Widgets[i]) == defaultPageID {
				d.Widgets[i].PageId = &page.Id
			}
		}
	}

	position := int64(len(pages))
	if params.Position != nil {
		if *params.Position < 0 || *params.Position > position {
			return "", fmt.Errorf("invalid position %d: it must be between 0 and %d", *params.Position, position)
		}
		position = *params.Position
	}
	pages = append(pages[:position], append([]iotclient.Pagepayload{page}, pages[position:]...)...)
	d.Pages = renumberPages(pages)
	return page.Id, nil
}

func updatePage(d *iotclient.Dashboardv3, params *UpdatePageParams) error {
	pages := sortedPages(d)
	index := pageIndex(pages, params.PageID)
	if index < 0 {
		return fmt.Errorf("page %s not found", params.PageID)
	}

	if params.Name != nil {
		if *params.Name == "" {
			return errors.New("page name cannot be empty")
		}
		pages[index].Name = *params.Name
	}
	if params.Icon != nil {
		pages[index].Icon = params.Icon
	}
	if params.Position != nil {
		last := int64(len(pages) - 1)
		if *params.Position < 0 || *params.Position > last {
			return fmt.Errorf("invalid position %d: it must be between 0 and %d", *params.Position, last)
		}
		page := pages[index]
		pages = append(pages[:index], pages[index+1:]...)
		pos := *params.Position
		pages = append(pages[:pos], append([]iotclient.Pagepayload{page}, pages[pos:]...)...)
	}
	d.Pages = renumberPages(pages)
	return nil
}

func deletePage(d *iotclient.Dashboardv3, params *DeletePageParams) error {
	if params.MoveWidgetsTo != nil && params.DeleteWidgets {
		return errors.New("cannot both move and delete the widgets of the page")
	}

	pages := sortedPages(d)
	index := pageIndex(pages, params.PageID)
	if index < 0 {
		return fmt.Errorf("page %s not found", params.PageID)
	}
	if len(pages) == 1 {
		return errors.New("cannot delete the only page of a dashboard")
	}

	if params.MoveWidgetsTo != nil {
		target := *params.MoveWidgetsTo
		if target == params.PageID {
			return errors.New("cannot move the widgets to the page being deleted")
		}
		if pageIndex(pages, target) < 0 {
			return fmt.Errorf("page %s not found", target)
		}
		// Keep the widgets' layout, shifted below the widgets already in the target page
		bottom, bottomMobile := pageBottom(d, target, "")
		for i := range d.Widgets {
			w := &d.Widgets[i]
			if widgetPageID(w) != params.PageID {
				continue
			}
			w.PageId = &target
			w.Y += bottom
			if w.YMobile != nil {
				w.YMobile = int64Ptr(*w.YMobile + bottomMobile)
			}
		}
	}

	widgets := d.Widgets[:0]
	for _, w := range d.Widgets {
		if widgetPageID(&w) != params.PageID {
			widgets = append(widgets, w)
		} else if !params.DeleteWidgets {
			return fmt.Errorf("page %s contains widgets: move them to another page or delete them", params.PageID)
		}
	}
	d.Widgets = widgets

	pages = append(pages[:index], pages[index+1:]...)
	d.Pages = renumberPages(pages)
	return nil
}

// sortedPages returns a copy of the pages of a dashboard, sorted by position.
func sortedPages(d *iotclient.Dashboardv3) []iotclient.Pagepayload {
	pages := append([]iotclient.Pagepayload{}, d.Pages...)
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].Position < pages[j].Position })
	return pages
}

func renumberPages(pages []iotclient.Pagepayload) []iotclient.Pagepayload {
	for i := range pages {
		pages[i].Position = int64(i)
	}
	return pages
}

func pageIndex(pages []iotclient.Pagepayload, pageID string) int {
	for i, p := range pages {
		if p.Id == pageID {
			return i
		}
	}
	return -1
}

func getPagesInfo(d *iotclient.Dashboardv3) []PageInfo {
	count := map[string]int{}
	for i := range d.Widgets {
		count[widgetPageID(&d.Widgets[i])]++
	}
	pages := []PageInfo{}
	for _, p := range sortedPages(d) {
		info := PageInfo{ID: p.Id, Name: p.Name, Position: p.Position, Widgets: count[p.Id]}
		if p.Icon != nil {
			info.Icon = *p.Icon
		}
		pages = append(pages, info)
	}
	return pages
}

func findPageInfo(d *iotclient.ArduinoDashboardv3, pageID string) (*PageInfo, error) {
	for _, p := range getPagesInfo(toDashboardPayload(d)) {
		if p.ID == pageID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("page %s not found in the updated dashboard", pageID)
}
