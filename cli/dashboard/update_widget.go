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
	"fmt"
	"os"

	"github.com/arduino/arduino-cli/cli/errorcodes"
	"github.com/arduino/arduino-cli/cli/feedback"
	"github.com/arduino/arduino-cloud-cli/command/dashboard"
	"github.com/arduino/arduino-cloud-cli/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type updateWidgetFlags struct {
	id        string
	widgetID  string
	name      string
	pageID    string
	variables []string
	options   string
	layout    layoutFlags
}

func initUpdateWidgetCommand() *cobra.Command {
	flags := &updateWidgetFlags{}
	updateWidgetCommand := &cobra.Command{
		Use:   "update-widget",
		Short: "Change a widget of a dashboard",
		Long: "Change name, page, position, size, options or variables of a widget of an Arduino IoT Cloud dashboard.\n" +
			"Only the passed flags are changed. Options are merged into the current ones; a null value removes an option.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runUpdateWidgetCommand(cmd, flags); err != nil {
				feedback.Errorf("Error during dashboard update-widget: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	updateWidgetCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	updateWidgetCommand.Flags().StringVarP(&flags.widgetID, "widget-id", "w", "", "ID of the widget, as printed by 'dashboard list-widgets'")
	updateWidgetCommand.Flags().StringVarP(&flags.name, "name", "n", "", "New widget name")
	updateWidgetCommand.Flags().StringVar(&flags.pageID, "page-id", "",
		"Move the widget to this page; without position flags it is placed below the widgets of that page")
	updateWidgetCommand.Flags().StringSliceVar(&flags.variables, "variable", nil,
		"Replace the linked variables, with format <thingID>:<variable_name>. Can be repeated")
	updateWidgetCommand.Flags().StringVarP(&flags.options, "options", "o", "", "Options to merge as a JSON object, e.g. '{\"max\": 50, \"decimals\": null}'")
	addLayoutFlags(updateWidgetCommand, &flags.layout)
	updateWidgetCommand.MarkFlagRequired("id")
	updateWidgetCommand.MarkFlagRequired("widget-id")
	return updateWidgetCommand
}

func runUpdateWidgetCommand(cmd *cobra.Command, flags *updateWidgetFlags) error {
	logrus.Infof("Updating widget %s of dashboard %s", flags.widgetID, flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	options, err := parseOptions(flags.options)
	if err != nil {
		return err
	}

	params := &dashboard.UpdateWidgetParams{
		DashboardID: flags.id,
		WidgetID:    flags.widgetID,
		Options:     options,
		Layout:      flags.layout.layout(cmd),
	}
	if cmd.Flags().Changed("name") {
		params.Name = &flags.name
	}
	if cmd.Flags().Changed("page-id") {
		params.PageID = &flags.pageID
	}
	if cmd.Flags().Changed("variable") {
		params.Variables = flags.variables
	}

	widget, err := dashboard.UpdateWidget(context.TODO(), cred, params)
	if err != nil {
		return err
	}

	feedback.PrintResult(widgetResult{widget})
	return nil
}
