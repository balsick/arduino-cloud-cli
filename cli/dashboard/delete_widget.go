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

type deleteWidgetFlags struct {
	id       string
	widgetID string
}

func initDeleteWidgetCommand() *cobra.Command {
	flags := &deleteWidgetFlags{}
	deleteWidgetCommand := &cobra.Command{
		Use:   "delete-widget",
		Short: "Delete a widget of a dashboard",
		Long:  "Delete a widget of an Arduino IoT Cloud dashboard, keeping the dashboard ID",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runDeleteWidgetCommand(flags); err != nil {
				feedback.Errorf("Error during dashboard delete-widget: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	deleteWidgetCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	deleteWidgetCommand.Flags().StringVarP(&flags.widgetID, "widget-id", "w", "", "ID of the widget, as printed by 'dashboard list-widgets'")
	deleteWidgetCommand.MarkFlagRequired("id")
	deleteWidgetCommand.MarkFlagRequired("widget-id")
	return deleteWidgetCommand
}

func runDeleteWidgetCommand(flags *deleteWidgetFlags) error {
	logrus.Infof("Deleting widget %s of dashboard %s", flags.widgetID, flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	if err := dashboard.DeleteWidget(context.TODO(), cred, flags.id, flags.widgetID); err != nil {
		return err
	}

	logrus.Info("Widget successfully deleted")
	return nil
}
