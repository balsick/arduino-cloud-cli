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

type createPageFlags struct {
	id       string
	name     string
	icon     string
	position int64
}

func initCreatePageCommand() *cobra.Command {
	flags := &createPageFlags{}
	createPageCommand := &cobra.Command{
		Use:   "create-page",
		Short: "Add a page to a dashboard",
		Long:  "Add a page to an existing dashboard of Arduino IoT Cloud, keeping the dashboard ID",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runCreatePageCommand(cmd, flags); err != nil {
				feedback.Errorf("Error during dashboard create-page: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	createPageCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	createPageCommand.Flags().StringVarP(&flags.name, "name", "n", "", "Page name")
	createPageCommand.Flags().StringVar(&flags.icon, "icon", "", "Page icon (default \"dashboard_filled\")")
	createPageCommand.Flags().Int64Var(&flags.position, "position", 0, "0-based position among the pages. Defaults to the last one")
	createPageCommand.MarkFlagRequired("id")
	createPageCommand.MarkFlagRequired("name")
	return createPageCommand
}

func runCreatePageCommand(cmd *cobra.Command, flags *createPageFlags) error {
	logrus.Infof("Adding page %s to dashboard %s", flags.name, flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	params := &dashboard.CreatePageParams{
		DashboardID: flags.id,
		Name:        flags.name,
	}
	if cmd.Flags().Changed("icon") {
		params.Icon = &flags.icon
	}
	if cmd.Flags().Changed("position") {
		params.Position = &flags.position
	}

	page, err := dashboard.CreatePage(context.TODO(), cred, params)
	if err != nil {
		return err
	}

	feedback.PrintResult(pageResult{page})
	return nil
}
