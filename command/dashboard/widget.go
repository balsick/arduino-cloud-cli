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
	"strings"

	"github.com/arduino/arduino-cloud-cli/config"
	"github.com/arduino/arduino-cloud-cli/internal/iot"
	iotclient "github.com/arduino/iot-client-go/v3"
	"github.com/gofrs/uuid"
)

type widgetSize struct {
	width, height, widthMobile, heightMobile int64
}

// widgetTypes contains the widget types of Arduino IoT Cloud dashboards
// and the size they get when none is specified.
// Type names are case-sensitive.
var widgetTypes = map[string]widgetSize{
	"Value":          {4, 3, 4, 3},
	"Gauge":          {6, 6, 4, 4},
	"Percentage":     {5, 5, 4, 5},
	"LED":            {2, 2, 4, 2},
	"Status":         {2, 2, 4, 2},
	"Chart":          {9, 6, 8, 6},
	"Advanced Chart": {10, 9, 8, 9},
	"Map":            {8, 8, 8, 8},
	"Advanced Map":   {10, 9, 8, 9},
	"Image Map":      {12, 10, 8, 9},
	"Switch":         {3, 3, 4, 4},
	"Push Button":    {4, 4, 4, 4},
	"Slider":         {6, 4, 8, 4},
	"Stepper":        {4, 3, 4, 3},
	"Value Selector": {6, 3, 8, 3},
	"Value Dropdown": {5, 3, 8, 3},
	"Messenger":      {8, 12, 8, 10},
	"Color":          {6, 7, 8, 7},
	"Colored light":  {7, 8, 8, 7},
	"Dimmed light":   {7, 8, 8, 7},
	"Time Picker":    {7, 5, 8, 5},
	"Scheduler":      {10, 7, 8, 7},
	"Sticky Note":    {6, 4, 8, 4},
	"Link":           {7, 4, 8, 4},
	"Image":          {6, 6, 8, 6},
}

// WidgetInfo contains the main parameters of a dashboard widget.
type WidgetInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	PageID       string                 `json:"page_id"`
	X            int64                  `json:"x"`
	Y            int64                  `json:"y"`
	Width        int64                  `json:"width"`
	Height       int64                  `json:"height"`
	XMobile      *int64                 `json:"x_mobile,omitempty"`
	YMobile      *int64                 `json:"y_mobile,omitempty"`
	WidthMobile  *int64                 `json:"width_mobile,omitempty"`
	HeightMobile *int64                 `json:"height_mobile,omitempty"`
	Variables    []string               `json:"variables,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// WidgetLayout contains the position and size of a widget, in grid cells.
// Desktop grid has 24 columns, mobile grid has 8 columns.
// Nil fields are left unchanged, or computed when creating a widget.
type WidgetLayout struct {
	X, Y, Width, Height                         *int64
	XMobile, YMobile, WidthMobile, HeightMobile *int64
}

// CreateWidgetParams contains the parameters needed to add a widget to a dashboard.
type CreateWidgetParams struct {
	DashboardID string
	Type        string
	Name        string
	PageID      *string                // Page of the widget; the first page if nil
	Variables   []string               // Linked variables, with format <thingID>:<variable_name>
	Options     map[string]interface{} // Widget options, e.g. {"min": 0, "max": 100} for a Gauge
	Layout      WidgetLayout           // Missing size defaults to the size of the type, missing position to the bottom of the page
}

// UpdateWidgetParams contains the parameters needed to change a widget.
// Nil fields are left unchanged.
type UpdateWidgetParams struct {
	DashboardID string
	WidgetID    string
	Name        *string
	PageID      *string                // Moving a widget to another page without a new position places it at the bottom of that page
	Variables   []string               // If not nil, replaces the linked variables
	Options     map[string]interface{} // Merged into the current options; a null value removes the option
	Layout      WidgetLayout
}

// thingShower retrieves a thing, used to resolve variable names into IDs.
type thingShower interface {
	ThingShow(ctx context.Context, id string) (*iotclient.ArduinoThing, error)
}

// ListWidgets returns the widgets of a dashboard, optionally only those of a page.
func ListWidgets(ctx context.Context, cred *config.Credentials, dashboardID string, pageID *string) ([]WidgetInfo, error) {
	iotClient, err := iot.NewClient(cred)
	if err != nil {
		return nil, err
	}
	dashboard, err := iotClient.DashboardShow(ctx, dashboardID)
	if err != nil {
		return nil, err
	}
	if pageID != nil && !hasPage(dashboard, *pageID) {
		return nil, fmt.Errorf("page %s not found in dashboard %s", *pageID, dashboardID)
	}

	widgets := []WidgetInfo{}
	for _, w := range dashboard.Widgets {
		info := getWidgetInfo(&w)
		if pageID != nil && info.PageID != *pageID {
			continue
		}
		widgets = append(widgets, info)
	}
	sortWidgets(widgets, dashboard.Pages)
	return widgets, nil
}

// sortWidgets sorts widgets by page position, then top to bottom and left to right.
func sortWidgets(widgets []WidgetInfo, pages []iotclient.ArduinoPagevariable) {
	position := map[string]int64{}
	for _, p := range pages {
		position[p.Id] = p.Position
	}
	sort.SliceStable(widgets, func(i, j int) bool {
		a, b := widgets[i], widgets[j]
		if position[a.PageID] != position[b.PageID] {
			return position[a.PageID] < position[b.PageID]
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
}

// CreateWidget adds a widget to an existing dashboard.
func CreateWidget(ctx context.Context, cred *config.Credentials, params *CreateWidgetParams) (*WidgetInfo, error) {
	var widgetID string
	updated, err := editDashboard(ctx, cred, params.DashboardID, func(ctx context.Context, client *iot.Client, d *iotclient.Dashboardv3) error {
		variables, err := resolveVariables(ctx, client, params.Variables)
		if err != nil {
			return err
		}
		widget, err := newWidget(d, params, variables)
		if err != nil {
			return err
		}
		widgetID = widget.Id
		d.Widgets = append(d.Widgets, *widget)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return findWidgetInfo(updated, widgetID)
}

// UpdateWidget changes name, page, layout, options or variables of a widget.
func UpdateWidget(ctx context.Context, cred *config.Credentials, params *UpdateWidgetParams) (*WidgetInfo, error) {
	updated, err := editDashboard(ctx, cred, params.DashboardID, func(ctx context.Context, client *iot.Client, d *iotclient.Dashboardv3) error {
		var variables []string
		if params.Variables != nil {
			var err error
			if variables, err = resolveVariables(ctx, client, params.Variables); err != nil {
				return err
			}
		}
		return updateWidget(d, params, variables)
	})
	if err != nil {
		return nil, err
	}
	return findWidgetInfo(updated, params.WidgetID)
}

// DeleteWidget removes a widget from a dashboard.
func DeleteWidget(ctx context.Context, cred *config.Credentials, dashboardID, widgetID string) error {
	_, err := editDashboard(ctx, cred, dashboardID, func(_ context.Context, _ *iot.Client, d *iotclient.Dashboardv3) error {
		return deleteWidget(d, widgetID)
	})
	return err
}

func newWidget(d *iotclient.Dashboardv3, params *CreateWidgetParams, variables []string) (*iotclient.Widgetv3, error) {
	size, ok := widgetTypes[params.Type]
	if !ok {
		return nil, fmt.Errorf("unknown widget type %q, available types are: %s", params.Type, strings.Join(widgetTypeNames(), ", "))
	}
	if params.Name == "" {
		return nil, errors.New("widget name not specified")
	}

	pageID, err := resolvePageID(d, params.PageID)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create a uuid for new widget: %w", err)
	}

	name := params.Name
	options := params.Options
	if options == nil {
		options = map[string]interface{}{}
	}
	widget := &iotclient.Widgetv3{
		Id:                   id.String(),
		Name:                 &name,
		Type:                 params.Type,
		PageId:               &pageID,
		Options:              options,
		Variables:            variables,
		Width:                size.width,
		Height:               size.height,
		WidthMobile:          int64Ptr(size.widthMobile),
		HeightMobile:         int64Ptr(size.heightMobile),
		AdditionalProperties: map[string]interface{}{},
	}

	l := params.Layout
	setIfNotNil(&widget.Width, l.Width)
	setIfNotNil(&widget.Height, l.Height)
	setPtrIfNotNil(&widget.WidthMobile, l.WidthMobile)
	setPtrIfNotNil(&widget.HeightMobile, l.HeightMobile)

	bottom, bottomMobile := pageBottom(d, pageID, "")
	widget.Y, widget.YMobile = bottom, int64Ptr(bottomMobile)
	widget.XMobile = int64Ptr(0)
	setIfNotNil(&widget.X, l.X)
	setIfNotNil(&widget.Y, l.Y)
	setPtrIfNotNil(&widget.XMobile, l.XMobile)
	setPtrIfNotNil(&widget.YMobile, l.YMobile)

	if err := validateLayout(widget); err != nil {
		return nil, err
	}
	return widget, nil
}

func updateWidget(d *iotclient.Dashboardv3, params *UpdateWidgetParams, variables []string) error {
	widget := findWidget(d, params.WidgetID)
	if widget == nil {
		return fmt.Errorf("widget %s not found in dashboard %s", params.WidgetID, params.DashboardID)
	}

	if params.Name != nil {
		if *params.Name == "" {
			return errors.New("widget name cannot be empty")
		}
		widget.Name = params.Name
	}

	l := params.Layout
	if params.PageID != nil && *params.PageID != widgetPageID(widget) {
		pageID, err := resolvePageID(d, params.PageID)
		if err != nil {
			return err
		}
		widget.PageId = &pageID
		// Without an explicit position, append the widget below the ones already in the target page
		bottom, bottomMobile := pageBottom(d, pageID, widget.Id)
		if l.X == nil && l.Y == nil {
			widget.X, widget.Y = 0, bottom
		}
		if l.XMobile == nil && l.YMobile == nil {
			widget.XMobile, widget.YMobile = int64Ptr(0), int64Ptr(bottomMobile)
		}
	}

	setIfNotNil(&widget.X, l.X)
	setIfNotNil(&widget.Y, l.Y)
	setIfNotNil(&widget.Width, l.Width)
	setIfNotNil(&widget.Height, l.Height)
	setPtrIfNotNil(&widget.XMobile, l.XMobile)
	setPtrIfNotNil(&widget.YMobile, l.YMobile)
	setPtrIfNotNil(&widget.WidthMobile, l.WidthMobile)
	setPtrIfNotNil(&widget.HeightMobile, l.HeightMobile)
	if err := validateLayout(widget); err != nil {
		return err
	}

	for k, v := range params.Options {
		if v == nil {
			delete(widget.Options, k)
		} else {
			widget.Options[k] = v
		}
	}

	if params.Variables != nil {
		widget.Variables = variables
	}
	return nil
}

func deleteWidget(d *iotclient.Dashboardv3, widgetID string) error {
	for i, w := range d.Widgets {
		if w.Id == widgetID {
			d.Widgets = append(d.Widgets[:i], d.Widgets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("widget %s not found", widgetID)
}

// resolveVariables converts variables with format <thingID>:<variable_name> into property IDs.
func resolveVariables(ctx context.Context, things thingShower, variables []string) ([]string, error) {
	ids := make([]string, 0, len(variables))
	cache := map[string]*iotclient.ArduinoThing{}
	for _, v := range variables {
		thingID, variableName, ok := strings.Cut(v, ":")
		if !ok || thingID == "" || variableName == "" {
			return nil, fmt.Errorf("invalid variable %q: expected format <thingID>:<variable_name>", v)
		}
		thing, found := cache[thingID]
		if !found {
			var err error
			if thing, err = things.ThingShow(ctx, thingID); err != nil {
				return nil, fmt.Errorf("getting variables of thing %s: %w", thingID, err)
			}
			cache[thingID] = thing
		}
		id := ""
		for _, p := range thing.Properties {
			if p.VariableName != nil && *p.VariableName == variableName {
				id = p.Id
				break
			}
		}
		if id == "" {
			return nil, fmt.Errorf("thing %s doesn't have a variable named %s", thingID, variableName)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// resolvePageID returns the page to use for a widget: the requested one,
// or the first page of the dashboard when none is requested.
func resolvePageID(d *iotclient.Dashboardv3, pageID *string) (string, error) {
	if pageID != nil {
		if len(d.Pages) == 0 && *pageID == defaultPageID {
			return defaultPageID, nil
		}
		for _, p := range d.Pages {
			if p.Id == *pageID {
				return p.Id, nil
			}
		}
		return "", fmt.Errorf("page %s not found", *pageID)
	}
	if len(d.Pages) == 0 {
		return defaultPageID, nil
	}
	first := d.Pages[0]
	for _, p := range d.Pages {
		if p.Position < first.Position {
			first = p
		}
	}
	return first.Id, nil
}

// pageBottom returns the first free row, on desktop and on mobile,
// below the widgets of a page, ignoring the widget with ID skipID.
func pageBottom(d *iotclient.Dashboardv3, pageID, skipID string) (int64, int64) {
	var bottom, bottomMobile int64
	for i := range d.Widgets {
		w := &d.Widgets[i]
		if w.Id == skipID || widgetPageID(w) != pageID {
			continue
		}
		bottom = max(bottom, w.Y+w.Height)
		if w.YMobile != nil && w.HeightMobile != nil {
			bottomMobile = max(bottomMobile, *w.YMobile+*w.HeightMobile)
		}
	}
	return bottom, bottomMobile
}

func validateLayout(w *iotclient.Widgetv3) error {
	if w.Width <= 0 || w.Height <= 0 {
		return errors.New("widget width and height must be positive")
	}
	if w.X < 0 || w.Y < 0 {
		return errors.New("widget position cannot be negative")
	}
	if w.X+w.Width > 24 {
		return fmt.Errorf("widget exceeds the 24 columns of the desktop grid (x %d + width %d)", w.X, w.Width)
	}
	if w.WidthMobile != nil && w.XMobile != nil && *w.XMobile+*w.WidthMobile > 8 {
		return fmt.Errorf("widget exceeds the 8 columns of the mobile grid (x %d + width %d)", *w.XMobile, *w.WidthMobile)
	}
	return nil
}

func findWidget(d *iotclient.Dashboardv3, widgetID string) *iotclient.Widgetv3 {
	for i := range d.Widgets {
		if d.Widgets[i].Id == widgetID {
			return &d.Widgets[i]
		}
	}
	return nil
}

func findWidgetInfo(d *iotclient.ArduinoDashboardv3, widgetID string) (*WidgetInfo, error) {
	for _, w := range d.Widgets {
		if w.Id == widgetID {
			info := getWidgetInfo(&w)
			return &info, nil
		}
	}
	return nil, fmt.Errorf("widget %s not found in the updated dashboard", widgetID)
}

func hasPage(d *iotclient.ArduinoDashboardv3, pageID string) bool {
	if len(d.Pages) == 0 {
		return pageID == defaultPageID
	}
	for _, p := range d.Pages {
		if p.Id == pageID {
			return true
		}
	}
	return false
}

func getWidgetInfo(w *iotclient.ArduinoWidgetv3) WidgetInfo {
	info := WidgetInfo{
		ID:           w.Id,
		Type:         w.Type,
		PageID:       w.PageId,
		X:            w.X,
		Y:            w.Y,
		Width:        w.Width,
		Height:       w.Height,
		XMobile:      w.XMobile,
		YMobile:      w.YMobile,
		WidthMobile:  w.WidthMobile,
		HeightMobile: w.HeightMobile,
		Options:      w.Options,
	}
	if info.PageID == "" {
		info.PageID = defaultPageID
	}
	if w.Name != nil {
		info.Name = *w.Name
	}
	for _, v := range w.Variables {
		info.Variables = append(info.Variables, v.ThingName+"/"+v.VariableName)
	}
	return info
}

func widgetTypeNames() []string {
	names := make([]string, 0, len(widgetTypes))
	for n := range widgetTypes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func int64Ptr(v int64) *int64 {
	return &v
}

func setIfNotNil(dst *int64, src *int64) {
	if src != nil {
		*dst = *src
	}
}

func setPtrIfNotNil(dst **int64, src *int64) {
	if src != nil {
		*dst = int64Ptr(*src)
	}
}
