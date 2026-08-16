// Command skill-review 根据测试用例和 Agent Trace 生成 Skill 评审报告。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ziangsun/szabot/internal/skillreview"
)

func main() {
	casesPath := flag.String("cases", "", "测试用例 JSON 文件")
	pathsPath := flag.String("paths", "", "Path 定义 JSON 文件，可选，用于校验 Path 模型")
	runsPath := flag.String("runs", "", "实际执行 Trace JSON 文件")
	serve := flag.Bool("serve", false, "启动本地评审仪表盘")
	addr := flag.String("addr", ":8090", "仪表盘监听地址")
	markdownPath := flag.String("markdown", "", "Markdown 报告输出路径，默认 stdout")
	jsonPath := flag.String("json", "", "JSON 报告输出路径，可选")
	version := flag.String("skill-version", "", "被评审的 Skill 版本")
	flag.Parse()

	if !*serve && (*casesPath == "" || *runsPath == "") {
		fmt.Fprintln(os.Stderr, "用法：skill-review -cases cases.json -runs runs.json [-markdown report.md] [-json report.json]")
		os.Exit(2)
	}
	var cases []skillreview.Case
	var runs []skillreview.Run
	var paths []skillreview.PathDefinition
	var err error
	if *casesPath != "" {
		cases, err = skillreview.LoadCases(*casesPath)
		if err != nil {
			fatal(err)
		}
	}
	if *pathsPath != "" {
		paths, err = skillreview.LoadPaths(*pathsPath)
		if err != nil {
			fatal(err)
		}
	}
	if *runsPath != "" {
		runs, err = skillreview.LoadRuns(*runsPath)
		if err != nil {
			fatal(err)
		}
	}
	report := skillreview.Evaluate(cases, runs, *version)
	report.SortResults()
	if *serve {
		if err := serveDashboard(*addr, report, paths); err != nil {
			fatal(err)
		}
		return
	}

	if *markdownPath == "" {
		if err := skillreview.WriteMarkdown(os.Stdout, report); err != nil {
			fatal(err)
		}
	} else if err := writeMarkdownFile(*markdownPath, report); err != nil {
		fatal(err)
	}
	if *jsonPath != "" {
		file, err := os.Create(*jsonPath)
		if err != nil {
			fatal(err)
		}
		err = skillreview.WriteJSON(file, report)
		closeErr := file.Close()
		if err != nil {
			fatal(err)
		}
		if closeErr != nil {
			fatal(closeErr)
		}
	}
}

func writeMarkdownFile(path string, report skillreview.Report) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return skillreview.WriteMarkdown(file, report)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "skill-review:", err)
	os.Exit(1)
}
