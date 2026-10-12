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

type createWidgetFlags struct {
	id        string
	widget    string
	name      string
	pageID    string
	variables []string
	options   string
	layout    layoutFlags
}

func initCreateWidgetCommand() *cobra.Command {
	flags := &createWidgetFlags{}
	createWidgetCommand := &cobra.Command{
		Use:   "create-widget",
		Short: "Add a widget to a dashboard",
		Long: "Add a widget to an existing dashboard of Arduino IoT Cloud, keeping the dashboard ID.\n" +
			"Without size flags the widget gets the default size of its type; without position flags\n" +
			"it is placed below the widgets already in the page.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runCreateWidgetCommand(cmd, flags); err != nil {
				feedback.Errorf("Error during dashboard create-widget: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	createWidgetCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	createWidgetCommand.Flags().StringVarP(&flags.widget, "type", "t", "", "Widget type, case-sensitive (e.g. Value, Gauge, Switch, Chart, 'Sticky Note')")
	createWidgetCommand.Flags().StringVarP(&flags.name, "name", "n", "", "Widget name")
	createWidgetCommand.Flags().StringVar(&flags.pageID, "page-id", "", "Page of the widget, as printed by 'dashboard list-pages'. Defaults to the first page")
	createWidgetCommand.Flags().StringSliceVar(&flags.variables, "variable", nil,
		"Variable linked to the widget, with format <thingID>:<variable_name>. Can be repeated")
	createWidgetCommand.Flags().StringVarP(&flags.options, "options", "o", "", "Widget options as a JSON object, e.g. '{\"min\": 0, \"max\": 100}'")
	addLayoutFlags(createWidgetCommand, &flags.layout)
	createWidgetCommand.MarkFlagRequired("id")
	createWidgetCommand.MarkFlagRequired("type")
	createWidgetCommand.MarkFlagRequired("name")
	return createWidgetCommand
}

func runCreateWidgetCommand(cmd *cobra.Command, flags *createWidgetFlags) error {
	logrus.Infof("Adding widget %s to dashboard %s", flags.name, flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	options, err := parseOptions(flags.options)
	if err != nil {
		return err
	}

	params := &dashboard.CreateWidgetParams{
		DashboardID: flags.id,
		Type:        flags.widget,
		Name:        flags.name,
		Variables:   flags.variables,
		Options:     options,
		Layout:      flags.layout.layout(cmd),
	}
	if flags.pageID != "" {
		params.PageID = &flags.pageID
	}

	widget, err := dashboard.CreateWidget(context.TODO(), cred, params)
	if err != nil {
		return err
	}

	feedback.PrintResult(widgetResult{widget})
	return nil
}
