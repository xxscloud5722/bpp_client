package common

import (
	"github.com/olekukonko/tablewriter"
	"os"
)

// PrintTable 输出Table.
func PrintTable(header []string, dataSources [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	for _, v := range dataSources {
		table.Append(v)
	}
	table.Render()
}
