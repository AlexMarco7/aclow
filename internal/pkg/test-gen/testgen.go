package testgen

import (
	"bufio"
	"encoding/json"
	"os"

	"html/template"
)

func Generate(src string, dest string) {

	fileHandle, _ := os.Open(src)
	defer fileHandle.Close()
	fileScanner := bufio.NewScanner(fileHandle)

	dict := map[string][]interface{}{}

	for fileScanner.Scan() {
		line := fileScanner.Text()

		if len(line) > 10 && line[0:10] == "aclow:>>>" {
			line = line[10:len(line)]
			var obj interface{}
			json.Unmarshal([]byte(line), &obj)
			executionID := obj.(map[string]interface{})["execution_id"].(string)

			if dict[executionID] != nil {
				dict[executionID] = append(dict[executionID], obj)
			} else {
				dict[executionID] = []interface{}{obj}
			}
		}
	}

	for _, v := range dict {
		buildTest(dest, v)
	}

	if fileScanner.Err() != nil {
		panic(fileScanner.Err())
	}

}

func buildTest(dest string, events []interface{}) {
	fn := template.FuncMap{
		"noescape": noescape,
	}

	tmpl := template.Must(template.New("template.tmpl").Funcs(fn).ParseFiles("./template.tmpl"))

	f, err := os.Create(dest)

	t := TestResource{}

	err = tmpl.Execute(f, t)

	if err != nil {
		panic(err)
	}
	defer f.Close()
}

func noescape(str string) template.HTML {
	return template.HTML(str)
}

type TestResource struct {
}
