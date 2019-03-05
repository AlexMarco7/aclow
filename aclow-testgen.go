package aclow

import (
	"bufio"
	"encoding/json"
	"os"

	"html/template"
)

func GenerateTests(src string, dest string) {

	fileHandle, _ := os.Open(src)
	defer fileHandle.Close()

	destFile, err := os.Create(dest)
	check(err)

	writer := bufio.NewWriter(destFile)
	defer destFile.Close()

	fileScanner := bufio.NewScanner(fileHandle)

	dict := map[string][]interface{}{}

	for fileScanner.Scan() {
		line := fileScanner.Text()

		if len(line) > 9 && line[0:9] == "aclow:>>>" {
			line = line[10:len(line)]
		} else if len(line) > 29 && line[20:29] == "aclow:>>>" {
			line = line[29:len(line)]
		} else {
			continue
		}

		var obj interface{}
		json.Unmarshal([]byte(line), &obj)

		if obj == nil {
			continue
		}

		executionID := obj.(map[string]interface{})["execution_id"].(string)

		if dict[executionID] != nil {
			dict[executionID] = append(dict[executionID], obj)
		} else {
			dict[executionID] = []interface{}{obj}
		}
	}

	for _, v := range dict {
		buildTest(v, writer)
	}

	destFile.Sync()
	writer.Flush()

	check(fileScanner.Err())

}

func buildTest(events []interface{}, writer *bufio.Writer) {
	fn := template.FuncMap{
		"noescape": noescape,
	}

	tmpl := template.Must(template.New("template.tmpl").Funcs(fn).Parse("teste"))

	t := TestResource{}

	err := tmpl.Execute(writer, t)

	check(err)
}

func noescape(str string) template.HTML {
	return template.HTML(str)
}

type TestResource struct {
}
