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

type listWidgetsFlags struct {
	id     string
	pageID string
}

func initListWidgetsCommand() *cobra.Command {
	flags := &listWidgetsFlags{}
	listWidgetsCommand := &cobra.Command{
		Use:   "list-widgets",
		Short: "List the widgets of a dashboard",
		Long:  "List the widgets of an Arduino IoT Cloud dashboard, with their page, layout and variables",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runListWidgetsCommand(cmd, flags); err != nil {
				feedback.Errorf("Error during dashboard list-widgets: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	listWidgetsCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	listWidgetsCommand.Flags().StringVar(&flags.pageID, "page-id", "", "List only the widgets of this page")
	listWidgetsCommand.MarkFlagRequired("id")
	return listWidgetsCommand
}

func runListWidgetsCommand(cmd *cobra.Command, flags *listWidgetsFlags) error {
	logrus.Infof("Listing widgets of dashboard %s", flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	var pageID *string
	if cmd.Flags().Changed("page-id") {
		pageID = &flags.pageID
	}
	widgets, err := dashboard.ListWidgets(context.TODO(), cred, flags.id, pageID)
	if err != nil {
		return err
	}

	feedback.PrintResult(listWidgetsResult{widgets})
	return nil
}

type listWidgetsResult struct {
	widgets []dashboard.WidgetInfo
}

func (r listWidgetsResult) Data() interface{} {
	return r.widgets
}

func (r listWidgetsResult) String() string {
	if len(r.widgets) == 0 {
		return "No widgets found."
	}
	return renderWidgets(r.widgets)
}
