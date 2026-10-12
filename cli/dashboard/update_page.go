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

type updatePageFlags struct {
	id       string
	pageID   string
	name     string
	icon     string
	position int64
}

func initUpdatePageCommand() *cobra.Command {
	flags := &updatePageFlags{}
	updatePageCommand := &cobra.Command{
		Use:   "update-page",
		Short: "Rename, move or change the icon of a dashboard page",
		Long:  "Rename, move or change the icon of a page of an Arduino IoT Cloud dashboard. Only the passed flags are changed",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runUpdatePageCommand(cmd, flags); err != nil {
				feedback.Errorf("Error during dashboard update-page: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	updatePageCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	updatePageCommand.Flags().StringVarP(&flags.pageID, "page-id", "p", "", "ID of the page, as printed by 'dashboard list-pages'")
	updatePageCommand.Flags().StringVarP(&flags.name, "name", "n", "", "New page name")
	updatePageCommand.Flags().StringVar(&flags.icon, "icon", "", "New page icon")
	updatePageCommand.Flags().Int64Var(&flags.position, "position", 0, "New 0-based position among the pages")
	updatePageCommand.MarkFlagRequired("id")
	updatePageCommand.MarkFlagRequired("page-id")
	return updatePageCommand
}

func runUpdatePageCommand(cmd *cobra.Command, flags *updatePageFlags) error {
	logrus.Infof("Updating page %s of dashboard %s", flags.pageID, flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	params := &dashboard.UpdatePageParams{
		DashboardID: flags.id,
		PageID:      flags.pageID,
	}
	if cmd.Flags().Changed("name") {
		params.Name = &flags.name
	}
	if cmd.Flags().Changed("icon") {
		params.Icon = &flags.icon
	}
	if cmd.Flags().Changed("position") {
		params.Position = &flags.position
	}

	page, err := dashboard.UpdatePage(context.TODO(), cred, params)
	if err != nil {
		return err
	}

	feedback.PrintResult(pageResult{page})
	return nil
}
