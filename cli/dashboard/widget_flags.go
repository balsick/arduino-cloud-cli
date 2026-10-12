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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arduino/arduino-cli/table"
	"github.com/arduino/arduino-cloud-cli/command/dashboard"
	"github.com/spf13/cobra"
)

type layoutFlags struct {
	x, y, width, height                         int64
	xMobile, yMobile, widthMobile, heightMobile int64
}

func addLayoutFlags(cmd *cobra.Command, flags *layoutFlags) {
	cmd.Flags().Int64Var(&flags.x, "x", 0, "Column of the widget on desktop (0-23)")
	cmd.Flags().Int64Var(&flags.y, "y", 0, "Row of the widget on desktop")
	cmd.Flags().Int64Var(&flags.width, "width", 0, "Width of the widget on desktop, in columns (max 24)")
	cmd.Flags().Int64Var(&flags.height, "height", 0, "Height of the widget on desktop, in rows")
	cmd.Flags().Int64Var(&flags.xMobile, "x-mobile", 0, "Column of the widget on mobile (0-7)")
	cmd.Flags().Int64Var(&flags.yMobile, "y-mobile", 0, "Row of the widget on mobile")
	cmd.Flags().Int64Var(&flags.widthMobile, "width-mobile", 0, "Width of the widget on mobile, in columns (max 8)")
	cmd.Flags().Int64Var(&flags.heightMobile, "height-mobile", 0, "Height of the widget on mobile, in rows")
}

// layout returns the layout with only the flags passed on the command line.
func (f *layoutFlags) layout(cmd *cobra.Command) dashboard.WidgetLayout {
	changed := func(name string, v *int64) *int64 {
		if cmd.Flags().Changed(name) {
			return v
		}
		return nil
	}
	return dashboard.WidgetLayout{
		X:            changed("x", &f.x),
		Y:            changed("y", &f.y),
		Width:        changed("width", &f.width),
		Height:       changed("height", &f.height),
		XMobile:      changed("x-mobile", &f.xMobile),
		YMobile:      changed("y-mobile", &f.yMobile),
		WidthMobile:  changed("width-mobile", &f.widthMobile),
		HeightMobile: changed("height-mobile", &f.heightMobile),
	}
}

func parseOptions(options string) (map[string]interface{}, error) {
	if options == "" {
		return nil, nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(options), &parsed); err != nil {
		return nil, fmt.Errorf("invalid options, a JSON object is expected: %w", err)
	}
	return parsed, nil
}

type widgetResult struct {
	widget *dashboard.WidgetInfo
}

func (r widgetResult) Data() interface{} {
	return r.widget
}

func (r widgetResult) String() string {
	return renderWidgets([]dashboard.WidgetInfo{*r.widget})
}

func renderWidgets(widgets []dashboard.WidgetInfo) string {
	t := table.New()
	t.SetHeader("Name", "ID", "Type", "Page", "Desktop x,y w×h", "Mobile x,y w×h", "Variables")
	for _, w := range widgets {
		mobile := ""
		if w.XMobile != nil && w.YMobile != nil && w.WidthMobile != nil && w.HeightMobile != nil {
			mobile = fmt.Sprintf("%d,%d %d×%d", *w.XMobile, *w.YMobile, *w.WidthMobile, *w.HeightMobile)
		}
		t.AddRow(
			w.Name,
			w.ID,
			w.Type,
			w.PageID,
			fmt.Sprintf("%d,%d %d×%d", w.X, w.Y, w.Width, w.Height),
			mobile,
			strings.Join(w.Variables, ", "),
		)
	}
	return t.Render()
}
