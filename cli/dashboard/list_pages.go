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

type listPagesFlags struct {
	id string
}

func initListPagesCommand() *cobra.Command {
	flags := &listPagesFlags{}
	listPagesCommand := &cobra.Command{
		Use:   "list-pages",
		Short: "List the pages of a dashboard",
		Long:  "List the pages of an Arduino IoT Cloud dashboard, sorted by position",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runListPagesCommand(flags); err != nil {
				feedback.Errorf("Error during dashboard list-pages: %v", err)
				os.Exit(errorcodes.ErrGeneric)
			}
		},
	}
	listPagesCommand.Flags().StringVarP(&flags.id, "id", "i", "", "Dashboard ID")
	listPagesCommand.MarkFlagRequired("id")
	return listPagesCommand
}

func runListPagesCommand(flags *listPagesFlags) error {
	logrus.Infof("Listing pages of dashboard %s", flags.id)

	cred, err := config.RetrieveCredentials()
	if err != nil {
		return fmt.Errorf("retrieving credentials: %w", err)
	}

	pages, err := dashboard.ListPages(context.TODO(), cred, flags.id)
	if err != nil {
		return err
	}

	feedback.PrintResult(listPagesResult{pages})
	return nil
}

type listPagesResult struct {
	pages []dashboard.PageInfo
}

func (r listPagesResult) Data() interface{} {
	return r.pages
}

func (r listPagesResult) String() string {
	if len(r.pages) == 0 {
		return "No pages found."
	}
	return renderPages(r.pages)
}
