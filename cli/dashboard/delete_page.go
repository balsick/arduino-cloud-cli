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

type deletePageFlags struct {
	id            string
	pageID        string
	moveWidgetsTo string
	deleteWidgets bool
}

func initDeletePageCommand() *cobra.Command {
	flags := &deletePageFlags{}
	deletePageCommand := &cobra.Command{
		Use:   "delete-page",
		Short: "Delete a page of a dashboard",
		Long: "Delete a page of an Arduino IoT Cloud dashboard, keeping the dashboard ID.\n" +
			"A page with widgets can be deleted only passing either --move-widgets-to or --delete-widgets.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runDeletePageCommand(flags); err != nil {
				feedback.Errorf("Error during dashboard delete-page: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	deletePageCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	deletePageCommand.Flags().StringVarP(&flags.pageID, "page-id", "p", "", "ID of the page, as printed by 'dashboard list-pages'")
	deletePageCommand.Flags().StringVar(&flags.moveWidgetsTo, "move-widgets-to", "",
		"Move the widgets of the page to this page, below its widgets")
	deletePageCommand.Flags().BoolVar(&flags.deleteWidgets, "delete-widgets", false, "Delete the widgets of the page")
	deletePageCommand.MarkFlagRequired("id")
	deletePageCommand.MarkFlagRequired("page-id")
	return deletePageCommand
}

func runDeletePageCommand(flags *deletePageFlags) error {
	logrus.Infof("Deleting page %s of dashboard %s", flags.pageID, flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	params := &dashboard.DeletePageParams{
		DashboardID:   flags.id,
		PageID:        flags.pageID,
		DeleteWidgets: flags.deleteWidgets,
	}
	if flags.moveWidgetsTo != "" {
		params.MoveWidgetsTo = &flags.moveWidgetsTo
	}

	if err := dashboard.DeletePage(context.TODO(), cred, params); err != nil {
		return err
	}

	logrus.Info("Page successfully deleted")
	return nil
}
