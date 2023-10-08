package message

import (
	"fmt"
	"strings"
)

// ParseTemplateParam 通过模板内容和参数进行填充生成以及内容转义.
func ParseTemplateParam(content string, args map[string]string) (string, error) {
	var rows = strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var result []string
	for _, row := range rows {
		for key, value := range args {
			row = strings.ReplaceAll(row, fmt.Sprintf("#{%s}", key), escape(value))
		}
		result = append(result, row)
	}
	return strings.Join(result, "\n"), nil
}

// escape 转义内容
func escape(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(value, "*", ""), "> ", ""), "\n", " "), "<br/>", "\n")
}
