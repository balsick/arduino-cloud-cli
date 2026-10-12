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

	"github.com/arduino/arduino-cloud-cli/config"
	"github.com/arduino/arduino-cloud-cli/internal/iot"
	iotclient "github.com/arduino/iot-client-go/v3"
)

// defaultPageID is the page of the widgets of a dashboard without pages.
const defaultPageID = "0"

// editDashboard retrieves a dashboard, lets edit change it and saves it back
// in place, so the dashboard keeps its ID, sharing and links.
func editDashboard(
	ctx context.Context,
	cred *config.Credentials,
	dashboardID string,
	edit func(ctx context.Context, client *iot.Client, dashboard *iotclient.Dashboardv3) error,
) (*iotclient.ArduinoDashboardv3, error) {
	iotClient, err := iot.NewClient(cred)
	if err != nil {
		return nil, err
	}

	current, err := iotClient.DashboardShow(ctx, dashboardID)
	if err != nil {
		return nil, err
	}

	dashboard := toDashboardPayload(current)
	if err := edit(ctx, iotClient, dashboard); err != nil {
		return nil, err
	}

	return iotClient.DashboardUpdate(ctx, dashboardID, dashboard)
}

// toDashboardPayload converts a dashboard as returned by Arduino IoT Cloud
// into the payload accepted to update it.
func toDashboardPayload(d *iotclient.ArduinoDashboardv3) *iotclient.Dashboardv3 {
	name := d.Name
	payload := &iotclient.Dashboardv3{
		Name:                 &name,
		CoverImage:           d.CoverImage,
		Pages:                make([]iotclient.Pagepayload, 0, len(d.Pages)),
		Widgets:              make([]iotclient.Widgetv3, 0, len(d.Widgets)),
		AdditionalProperties: map[string]interface{}{},
	}

	for _, p := range d.Pages {
		payload.Pages = append(payload.Pages, iotclient.Pagepayload{
			Id:                   p.Id,
			Name:                 p.Name,
			Position:             p.Position,
			Icon:                 p.Icon,
			AdditionalProperties: map[string]interface{}{},
		})
	}

	for _, w := range d.Widgets {
		widget := iotclient.Widgetv3{
			Id:                   w.Id,
			Name:                 w.Name,
			Type:                 w.Type,
			Options:              w.Options,
			X:                    w.X,
			Y:                    w.Y,
			Width:                w.Width,
			Height:               w.Height,
			XMobile:              w.XMobile,
			YMobile:              w.YMobile,
			WidthMobile:          w.WidthMobile,
			HeightMobile:         w.HeightMobile,
			AdditionalProperties: map[string]interface{}{},
		}
		if widget.Options == nil {
			widget.Options = map[string]interface{}{}
		}
		pageID := w.PageId
		if pageID == "" {
			pageID = defaultPageID
		}
		widget.PageId = &pageID
		for _, v := range w.Variables {
			widget.Variables = append(widget.Variables, v.Id)
		}
		payload.Widgets = append(payload.Widgets, widget)
	}

	return payload
}

func widgetPageID(w *iotclient.Widgetv3) string {
	if w.PageId == nil || *w.PageId == "" {
		return defaultPageID
	}
	return *w.PageId
}
