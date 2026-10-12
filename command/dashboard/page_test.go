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
	"testing"

	iotclient "github.com/arduino/iot-client-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pageNames(d *iotclient.Dashboardv3) []string {
	names := []string{}
	for _, p := range sortedPages(d) {
		names = append(names, p.Name)
	}
	return names
}

func TestCreatePage(t *testing.T) {
	d := testDashboard()
	id, err := createPage(d, &CreatePageParams{Name: "Last"})
	require.NoError(t, err)
	assert.Equal(t, []string{"First", "Second", "Last"}, pageNames(d))
	p := sortedPages(d)[2]
	assert.Equal(t, id, p.Id)
	assert.Equal(t, defaultPageIcon, *p.Icon)

	_, err = createPage(d, &CreatePageParams{Name: "Zero", Position: int64Ptr(0), Icon: strPtr("shutter")})
	require.NoError(t, err)
	assert.Equal(t, []string{"Zero", "First", "Second", "Last"}, pageNames(d))
	for i, p := range sortedPages(d) {
		assert.Equal(t, int64(i), p.Position)
	}
}

func TestCreatePageErrors(t *testing.T) {
	_, err := createPage(testDashboard(), &CreatePageParams{})
	assert.ErrorContains(t, err, "page name not specified")
	_, err = createPage(testDashboard(), &CreatePageParams{Name: "x", Position: int64Ptr(3)})
	assert.ErrorContains(t, err, "must be between 0 and 2")
}

func TestCreateFirstPageAdoptsWidgets(t *testing.T) {
	d := &iotclient.Dashboardv3{Widgets: []iotclient.Widgetv3{{Id: "w1", PageId: strPtr(defaultPageID)}, {Id: "w2"}}}
	id, err := createPage(d, &CreatePageParams{Name: "Page 1"})
	require.NoError(t, err)
	for _, w := range d.Widgets {
		assert.Equal(t, id, *w.PageId)
	}
}

func TestUpdatePage(t *testing.T) {
	d := testDashboard()
	require.NoError(t, updatePage(d, &UpdatePageParams{PageID: page2, Name: strPtr("Controls"), Icon: strPtr("small"), Position: int64Ptr(0)}))
	assert.Equal(t, []string{"Controls", "First"}, pageNames(d))
	assert.Equal(t, "small", *sortedPages(d)[0].Icon)

	assert.ErrorContains(t, updatePage(d, &UpdatePageParams{PageID: "nope"}), "page nope not found")
	assert.ErrorContains(t, updatePage(d, &UpdatePageParams{PageID: page1, Name: strPtr("")}), "cannot be empty")
	assert.ErrorContains(t, updatePage(d, &UpdatePageParams{PageID: page1, Position: int64Ptr(2)}), "must be between 0 and 1")
}

func TestDeletePageMoveWidgets(t *testing.T) {
	d := testDashboard()
	d.Widgets = append(d.Widgets, iotclient.Widgetv3{Id: "w3", PageId: strPtr(page2), Y: 1, Height: 2,
		YMobile: int64Ptr(1), HeightMobile: int64Ptr(2)})

	require.NoError(t, deletePage(d, &DeletePageParams{PageID: page2, MoveWidgetsTo: strPtr(page1)}))
	assert.Equal(t, []string{"First"}, pageNames(d))
	require.Len(t, d.Widgets, 3)
	w := findWidget(d, "w3")
	assert.Equal(t, page1, *w.PageId)
	// Shifted below the widgets of the target page, keeping its relative position
	assert.Equal(t, int64(10), w.Y)
	assert.Equal(t, int64(9), *w.YMobile)
}

func TestDeletePageDeleteWidgets(t *testing.T) {
	d := testDashboard()
	require.NoError(t, deletePage(d, &DeletePageParams{PageID: page1, DeleteWidgets: true}))
	assert.Equal(t, []string{"Second"}, pageNames(d))
	assert.Empty(t, d.Widgets)
}

func TestDeletePageErrors(t *testing.T) {
	tests := []struct {
		name   string
		params DeletePageParams
		errMsg string
	}{
		{"widgets without flag", DeletePageParams{PageID: page1}, "contains widgets"},
		{"unknown page", DeletePageParams{PageID: "nope"}, "page nope not found"},
		{"unknown target", DeletePageParams{PageID: page1, MoveWidgetsTo: strPtr("nope")}, "page nope not found"},
		{"target is the deleted page", DeletePageParams{PageID: page1, MoveWidgetsTo: strPtr(page1)}, "page being deleted"},
		{"move and delete", DeletePageParams{PageID: page1, MoveWidgetsTo: strPtr(page2), DeleteWidgets: true}, "cannot both"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorContains(t, deletePage(testDashboard(), &tt.params), tt.errMsg)
		})
	}

	d := testDashboard()
	require.NoError(t, deletePage(d, &DeletePageParams{PageID: page2}))
	assert.ErrorContains(t, deletePage(d, &DeletePageParams{PageID: page1, DeleteWidgets: true}), "only page")
}

func TestGetPagesInfo(t *testing.T) {
	assert.Equal(t, []PageInfo{
		{ID: page1, Name: "First", Position: 0, Widgets: 2},
		{ID: page2, Name: "Second", Position: 1, Widgets: 0},
	}, getPagesInfo(testDashboard()))
}
