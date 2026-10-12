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
	"testing"

	iotclient "github.com/arduino/iot-client-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	page1 = "11111111-1111-1111-1111-111111111111"
	page2 = "22222222-2222-2222-2222-222222222222"
)

func strPtr(s string) *string {
	return &s
}

// testDashboard returns a dashboard with two pages:
// page1 has a 6x6 widget at the top and a 3x3 one at row 6, page2 is empty.
func testDashboard() *iotclient.Dashboardv3 {
	return &iotclient.Dashboardv3{
		Pages: []iotclient.Pagepayload{
			{Id: page2, Name: "Second", Position: 1},
			{Id: page1, Name: "First", Position: 0},
		},
		Widgets: []iotclient.Widgetv3{
			{Id: "w1", Type: "Gauge", PageId: strPtr(page1), X: 0, Y: 0, Width: 6, Height: 6,
				XMobile: int64Ptr(0), YMobile: int64Ptr(0), WidthMobile: int64Ptr(4), HeightMobile: int64Ptr(4),
				Options: map[string]interface{}{"min": 0.0, "max": 100.0}},
			{Id: "w2", Type: "Switch", PageId: strPtr(page1), X: 6, Y: 6, Width: 3, Height: 3,
				XMobile: int64Ptr(0), YMobile: int64Ptr(4), WidthMobile: int64Ptr(4), HeightMobile: int64Ptr(4),
				Options: map[string]interface{}{}},
		},
	}
}

func TestToDashboardPayload(t *testing.T) {
	varName := "temperature"
	payload := toDashboardPayload(&iotclient.ArduinoDashboardv3{
		Id:    "dashboard-id",
		Name:  "Greenhouse",
		Pages: []iotclient.ArduinoPagevariable{{Id: page1, Name: "First", Position: 0, Icon: strPtr("shutter")}},
		Widgets: []iotclient.ArduinoWidgetv3{
			{Id: "w1", Name: &varName, Type: "Value", PageId: page1, Width: 4, Height: 3,
				Variables: []iotclient.ArduinoLinkedvariable{{Id: "prop-1"}, {Id: "prop-2"}}},
			{Id: "w2", Type: "Sticky Note", Width: 6, Height: 4},
		},
	})

	assert.Equal(t, "Greenhouse", *payload.Name)
	assert.Equal(t, []iotclient.Pagepayload{{Id: page1, Name: "First", Position: 0, Icon: strPtr("shutter"),
		AdditionalProperties: map[string]interface{}{}}}, payload.Pages)
	require.Len(t, payload.Widgets, 2)
	assert.Equal(t, []string{"prop-1", "prop-2"}, payload.Widgets[0].Variables)
	assert.Equal(t, page1, *payload.Widgets[0].PageId)
	// Widgets without a page belong to the default page, and options are never nil
	assert.Equal(t, defaultPageID, *payload.Widgets[1].PageId)
	assert.NotNil(t, payload.Widgets[1].Options)
}

func TestNewWidgetDefaults(t *testing.T) {
	d := testDashboard()
	w, err := newWidget(d, &CreateWidgetParams{Type: "Slider", Name: "Speed"}, []string{"prop-1"})
	require.NoError(t, err)

	// Default size of the type, first page by position, below the existing widgets
	assert.Equal(t, page1, *w.PageId)
	assert.Equal(t, []int64{0, 9, 6, 4}, []int64{w.X, w.Y, w.Width, w.Height})
	assert.Equal(t, []int64{0, 8, 8, 4}, []int64{*w.XMobile, *w.YMobile, *w.WidthMobile, *w.HeightMobile})
	assert.Equal(t, []string{"prop-1"}, w.Variables)
	assert.NotEmpty(t, w.Id)
	assert.NotNil(t, w.Options)
}

func TestNewWidgetLayoutAndPage(t *testing.T) {
	d := testDashboard()
	w, err := newWidget(d, &CreateWidgetParams{
		Type:    "Value",
		Name:    "Temp",
		PageID:  strPtr(page2),
		Options: map[string]interface{}{"decimals": 1.0},
		Layout:  WidgetLayout{X: int64Ptr(10), Width: int64Ptr(5), YMobile: int64Ptr(2)},
	}, nil)
	require.NoError(t, err)

	assert.Equal(t, page2, *w.PageId)
	// Empty page: placed at the top; passed values win over defaults
	assert.Equal(t, []int64{10, 0, 5, 3}, []int64{w.X, w.Y, w.Width, w.Height})
	assert.Equal(t, int64(2), *w.YMobile)
	assert.Equal(t, 1.0, w.Options["decimals"])
}

func TestNewWidgetErrors(t *testing.T) {
	tests := []struct {
		name   string
		params CreateWidgetParams
		errMsg string
	}{
		{"unknown type", CreateWidgetParams{Type: "switch", Name: "x"}, "unknown widget type"},
		{"missing name", CreateWidgetParams{Type: "Switch"}, "widget name not specified"},
		{"unknown page", CreateWidgetParams{Type: "Switch", Name: "x", PageID: strPtr("nope")}, "page nope not found"},
		{"desktop overflow", CreateWidgetParams{Type: "Switch", Name: "x", Layout: WidgetLayout{X: int64Ptr(22)}}, "24 columns"},
		{"mobile overflow", CreateWidgetParams{Type: "Switch", Name: "x", Layout: WidgetLayout{XMobile: int64Ptr(5)}}, "8 columns"},
		{"zero size", CreateWidgetParams{Type: "Switch", Name: "x", Layout: WidgetLayout{Width: int64Ptr(0)}}, "must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newWidget(testDashboard(), &tt.params, nil)
			assert.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func TestNewWidgetWithoutPages(t *testing.T) {
	d := &iotclient.Dashboardv3{}
	w, err := newWidget(d, &CreateWidgetParams{Type: "LED", Name: "Led"}, nil)
	require.NoError(t, err)
	assert.Equal(t, defaultPageID, *w.PageId)
}

func TestUpdateWidget(t *testing.T) {
	d := testDashboard()
	err := updateWidget(d, &UpdateWidgetParams{
		WidgetID: "w1",
		Name:     strPtr("Temperature"),
		Options:  map[string]interface{}{"max": 50.0, "min": nil, "decimals": 2.0},
		Layout:   WidgetLayout{Height: int64Ptr(7)},
	}, nil)
	require.NoError(t, err)

	w := findWidget(d, "w1")
	assert.Equal(t, "Temperature", *w.Name)
	assert.Equal(t, int64(7), w.Height)
	assert.Equal(t, map[string]interface{}{"max": 50.0, "decimals": 2.0}, w.Options)
	assert.Nil(t, w.Variables)
}

func TestUpdateWidgetMoveToPage(t *testing.T) {
	d := testDashboard()
	d.Widgets = append(d.Widgets, iotclient.Widgetv3{Id: "w3", PageId: strPtr(page2), Y: 0, Height: 2, Width: 2,
		YMobile: int64Ptr(0), HeightMobile: int64Ptr(5)})

	err := updateWidget(d, &UpdateWidgetParams{WidgetID: "w2", PageID: strPtr(page2), Variables: []string{"prop-9"}}, []string{"prop-9"})
	require.NoError(t, err)

	w := findWidget(d, "w2")
	assert.Equal(t, page2, *w.PageId)
	// Placed below the widgets of the target page
	assert.Equal(t, []int64{0, 2}, []int64{w.X, w.Y})
	assert.Equal(t, []int64{0, 5}, []int64{*w.XMobile, *w.YMobile})
	assert.Equal(t, []string{"prop-9"}, w.Variables)
}

func TestUpdateWidgetErrors(t *testing.T) {
	assert.ErrorContains(t, updateWidget(testDashboard(), &UpdateWidgetParams{WidgetID: "nope"}, nil), "widget nope not found")
	assert.ErrorContains(t, updateWidget(testDashboard(), &UpdateWidgetParams{WidgetID: "w1", Name: strPtr("")}, nil), "cannot be empty")
	assert.ErrorContains(t, updateWidget(testDashboard(), &UpdateWidgetParams{WidgetID: "w1", PageID: strPtr("nope")}, nil), "page nope not found")
	assert.ErrorContains(t, updateWidget(testDashboard(), &UpdateWidgetParams{WidgetID: "w1", Layout: WidgetLayout{X: int64Ptr(20)}}, nil), "24 columns")
}

func TestDeleteWidget(t *testing.T) {
	d := testDashboard()
	require.NoError(t, deleteWidget(d, "w1"))
	require.Len(t, d.Widgets, 1)
	assert.Equal(t, "w2", d.Widgets[0].Id)
	assert.ErrorContains(t, deleteWidget(d, "w1"), "widget w1 not found")
}

type fakeThings struct {
	things map[string]*iotclient.ArduinoThing
	calls  int
}

func (f *fakeThings) ThingShow(_ context.Context, id string) (*iotclient.ArduinoThing, error) {
	f.calls++
	thing, ok := f.things[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return thing, nil
}

func TestResolveVariables(t *testing.T) {
	things := &fakeThings{things: map[string]*iotclient.ArduinoThing{
		"thing-1": {Id: "thing-1", Properties: []iotclient.ArduinoProperty{
			{Id: "prop-temp", VariableName: strPtr("temperature")},
			{Id: "prop-pump", VariableName: strPtr("pump")},
		}},
	}}

	ids, err := resolveVariables(context.Background(), things, []string{"thing-1:pump", "thing-1:temperature"})
	require.NoError(t, err)
	assert.Equal(t, []string{"prop-pump", "prop-temp"}, ids)
	assert.Equal(t, 1, things.calls, "each thing should be retrieved once")

	_, err = resolveVariables(context.Background(), things, []string{"thing-1:nope"})
	assert.ErrorContains(t, err, "doesn't have a variable named nope")
	_, err = resolveVariables(context.Background(), things, []string{"thing-2:pump"})
	assert.ErrorContains(t, err, "getting variables of thing thing-2")
	_, err = resolveVariables(context.Background(), things, []string{"pump"})
	assert.ErrorContains(t, err, "expected format <thingID>:<variable_name>")
}
